package i18n

import (
	"fmt"

	"github.com/go-telegram/bot/models"
)

// AdminChapter is one chapter row (Index is 0-based).
type AdminChapter struct {
	Index   int
	Title   string
	Visible bool
}

// AdminChapterList builds the "🛠 Выберите действие" menu: one title row
// plus one "✏️ 🗑 👁" action row per chapter (already paged).
func AdminChapterList(globalVisible bool, chapters []AdminChapter, page, pages int, owner int64) Message {
	header := "🛠 Выберите действие"
	if !globalVisible {
		header += "\n⚠️ Правила СКРЫТЫ глобально (/rules_toggle чтобы вернуть)"
	}
	if len(chapters) == 0 {
		return Message{Text: header + "\n\nГлав пока нет."}
	}
	var rows [][]models.InlineKeyboardButton
	for _, c := range chapters {
		rows = append(rows, []models.InlineKeyboardButton{
			button(fmt.Sprintf("%d. %s", c.Index+1, c.Title), CbAdminOpen(c.Index, owner)),
		})
		rows = append(rows, []models.InlineKeyboardButton{
			button("✏️", CbAdminOpen(c.Index, owner)),
			button("🗑", CbAdminDeleteChapter(c.Index, owner)),
			button(eye(c.Visible), CbAdminToggleChapter(c.Index, owner)),
		})
	}
	prev, next := pageNav(page, pages, func(p int) string { return CbAdminPage(p, owner) })
	rows = pager(rows, page, pages, prev, next)
	text := header
	if pages > 1 {
		text += fmt.Sprintf("\nСтраница %d/%d", page+1, pages)
	}
	return Message{Text: text, Markup: markup(rows...)}
}

// AdminItem is one rule row (Index is 0-based).
type AdminItem struct {
	Index   int
	Text    string
	Visible bool
}

// AdminItemList builds the per-chapter item menu (already paged).
// chIdx is the 0-based chapter index used in callbacks.
func AdminItemList(chIdx int, title string, chapVisible bool, items []AdminItem, page, pages int, owner int64) Message {
	num := chIdx + 1
	header := fmt.Sprintf("🛠 Глава %d. %s\nВыберите действие:", num, title)
	if len(items) == 0 {
		return Message{Text: header + "\n\nПунктов пока нет.", Markup: backToAdmin(owner)}
	}
	var rows [][]models.InlineKeyboardButton
	for _, it := range items {
		rows = append(rows, []models.InlineKeyboardButton{
			button(fmt.Sprintf("%d.%d %s", num, it.Index+1, preview(it.Text, 40)), CbNoop),
		})
		rows = append(rows, []models.InlineKeyboardButton{
			button("✏️", CbAdminEditItem(chIdx, it.Index, owner)),
			button("🗑", CbAdminDeleteItem(chIdx, it.Index, owner)),
			button(eye(it.Visible), CbAdminToggleItem(chIdx, it.Index, owner)),
		})
	}
	prev, next := pageNav(page, pages, func(p int) string { return CbAdminItemsPage(chIdx, p, owner) })
	rows = pager(rows, page, pages, prev, next)
	rows = append(rows, []models.InlineKeyboardButton{button("⬅️ К главам", CbAdminBack(owner))})
	text := header
	if pages > 1 {
		text += fmt.Sprintf("\nСтраница %d/%d", page+1, pages)
	}
	return Message{Text: text, Markup: markup(rows...)}
}

func backToAdmin(owner int64) *models.InlineKeyboardMarkup {
	return markup([]models.InlineKeyboardButton{button("⬅️ К главам", CbAdminBack(owner))})
}

// ConfirmDeleteChapter asks "really delete chapter idx?" (title may be empty).
func ConfirmDeleteChapter(idx int, title string, owner int64) Message {
	name := fmt.Sprintf("главу %d", idx+1)
	if title != "" {
		name = fmt.Sprintf("главу %d. %s", idx+1, title)
	}
	return Message{
		Text: fmt.Sprintf("⚠️ Удалить %s?\nДействие нельзя отменить.", name),
		Markup: markup([]models.InlineKeyboardButton{
			button("✅ Да, удалить", CbAdminDeleteChapterOK(idx, owner)),
			button("❌ Нет", CbAdminBack(owner)),
		}),
	}
}

// ConfirmDeleteItem asks "really delete item chIdx.itemIdx?".
func ConfirmDeleteItem(chIdx, itemIdx int, owner int64) Message {
	return Message{
		Text: fmt.Sprintf("⚠️ Удалить пункт %d.%d?\nДействие нельзя отменить.", chIdx+1, itemIdx+1),
		Markup: markup([]models.InlineKeyboardButton{
			button("✅ Да, удалить", CbAdminDeleteItemOK(chIdx, itemIdx, owner)),
			button("❌ Нет", CbAdminItemsPage(chIdx, itemIdx/ItemsPerPage, owner)),
		}),
	}
}

// ChapterNotFound replaces the item menu for a bad chapter index.
func ChapterNotFound() Message {
	return Message{Text: "Глава не найдена."}
}

// EditTitlePrompt asks the admin to send the new top-level title next.
func EditTitlePrompt(current string) Message {
	text := "✏️ Пришлите новый заголовок правил следующим сообщением.\nНапишите «отмена» для отмены."
	if current != "" {
		text += "\n\nТекущий: " + current
	}
	return Message{Text: text}
}

// TitleUpdated confirms the new top-level title was stored.
func TitleUpdated() Message {
	return Message{Text: "✅ Заголовок обновлён."}
}

// EditItemPrompt asks the admin to send the new item text next.
func EditItemPrompt(chapterNum, itemNum int) Message {
	return Message{Text: fmt.Sprintf("✏️ Пришлите новый текст пункта %d.%d следующим сообщением.\nНапишите «отмена» для отмены.", chapterNum, itemNum)}
}
