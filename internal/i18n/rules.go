package i18n

import (
	"fmt"
	"strings"

	"github.com/go-telegram/bot/models"
)

// DefaultRulesTitle is used when rules.json has an empty title.
const DefaultRulesTitle = "📋 Правила"

// ChapterButton is one chapter row (Index is 0-based).
type ChapterButton struct {
	Index int
	Title string
}

// RulesList builds the header message with chapter buttons (already paged).
// owner receives the menu and is the only one who may press its buttons.
func RulesList(title string, chapters []ChapterButton, page, pages int, owner int64) Message {
	if strings.TrimSpace(title) == "" {
		title = DefaultRulesTitle
	}
	if len(chapters) == 0 {
		return Message{Text: title + "\n\nПока нет доступных глав."}
	}
	var rows [][]models.InlineKeyboardButton
	for _, c := range chapters {
		rows = append(rows, []models.InlineKeyboardButton{
			button(fmt.Sprintf("%d. %s", c.Index+1, c.Title), CbUserChapter(c.Index, owner)),
		})
	}
	prev, next := pageNav(page, pages, func(p int) string { return CbUserPage(p, owner) })
	rows = pager(rows, page, pages, prev, next)
	text := title
	if pages > 1 {
		text += fmt.Sprintf("\n\nСтраница %d/%d", page+1, pages)
	}
	return Message{Text: text, Markup: markup(rows...)}
}

// RuleLine is one visible rule; numbers are 1-based and stable.
type RuleLine struct {
	ChapterNum int
	ItemNum    int
	Text       string
}

// ChapterDetail builds the "📌 N. Title" message with "N.M text" lines.
func ChapterDetail(chapterNum int, title string, items []RuleLine, owner int64) Message {
	var b strings.Builder
	fmt.Fprintf(&b, "📌 %d. %s\n\n", chapterNum, title)
	if len(items) == 0 {
		b.WriteString("Пока нет доступных пунктов.")
	}
	for _, it := range items {
		fmt.Fprintf(&b, "%d.%d %s\n\n", it.ChapterNum, it.ItemNum, it.Text)
	}
	rows := [][]models.InlineKeyboardButton{
		{button("⬅️ К главам", CbUserBack(owner))},
	}
	return Message{Text: strings.TrimSpace(b.String()), Markup: markup(rows...)}
}

// RulesHidden is shown to users when the global toggle is off.
func RulesHidden() Message {
	return Message{Text: "🚫 Правила временно скрыты администратором."}
}

// ChapterUnavailable replaces the chapter view for hidden/bad indexes.
func ChapterUnavailable() Message {
	return Message{Text: "Глава недоступна."}
}
