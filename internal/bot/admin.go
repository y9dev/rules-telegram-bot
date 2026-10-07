package bot

import (
	"context"
	"strconv"
	"strings"

	tgbot "github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"

	"rules-telegram-bot/internal/i18n"
)

// --- callback handlers (one per registered CallbackQuery pattern) ---

// callbackTarget extracts chat, message and callback IDs plus data.
func (h *Handler) callbackTarget(u *models.Update) (chatID int64, msgID int, cbID, data string) {
	q := u.CallbackQuery
	cbID, data = q.ID, q.Data
	if q.Message.Message != nil {
		chatID = q.Message.Message.Chat.ID
		msgID = q.Message.Message.ID
	}
	return chatID, msgID, cbID, data
}

// owned verifies the presser owns the menu and returns the payload body.
func (h *Handler) owned(ctx context.Context, u *models.Update) (chatID int64, msgID int, cbID, body string, presser int64, ok bool) {
	chatID, msgID, cbID, data := h.callbackTarget(u)
	presser = h.updateUserID(u)
	body, ok = checkOwner(data, presser)
	if !ok {
		h.answerCb(ctx, cbID, i18n.NotYourMenuAck())
		return 0, 0, "", "", 0, false
	}
	return chatID, msgID, cbID, body, presser, true
}

func (h *Handler) handleNoop(ctx context.Context, b *tgbot.Bot, u *models.Update) {
	h.answerCb(ctx, u.CallbackQuery.ID, "")
}

func (h *Handler) handleUserBack(ctx context.Context, b *tgbot.Bot, u *models.Update) {
	chatID, msgID, cbID, _, presser, ok := h.owned(ctx, u)
	if !ok {
		return
	}
	h.answerCb(ctx, cbID, "")
	h.renderUserChaptersEdit(ctx, chatID, msgID, 0, presser)
}

func (h *Handler) handleAdminBack(ctx context.Context, b *tgbot.Bot, u *models.Update) {
	chatID, msgID, cbID, _, presser, ok := h.owned(ctx, u)
	if !ok {
		return
	}
	h.answerCb(ctx, cbID, "")
	h.renderAdminChapters(ctx, chatID, msgID, 0, false, presser)
}

func (h *Handler) handleAdminOpenChapter(ctx context.Context, b *tgbot.Bot, u *models.Update) {
	chatID, msgID, cbID, body, presser, ok := h.owned(ctx, u)
	if !ok {
		return
	}
	h.answerCb(ctx, cbID, "")
	idx, ok := parseIndex(strings.TrimPrefix(body, i18n.CbPAdminOpen))
	if !ok {
		return
	}
	h.renderAdminItems(ctx, chatID, msgID, idx, 0, false, presser)
}

func (h *Handler) handleAdminPage(ctx context.Context, b *tgbot.Bot, u *models.Update) {
	chatID, msgID, cbID, body, presser, ok := h.owned(ctx, u)
	if !ok {
		return
	}
	h.answerCb(ctx, cbID, "")
	p, _ := strconv.Atoi(strings.TrimPrefix(body, i18n.CbPAdminPage))
	h.renderAdminChapters(ctx, chatID, msgID, p, false, presser)
}

func (h *Handler) handleAdminToggleChapter(ctx context.Context, b *tgbot.Bot, u *models.Update) {
	chatID, msgID, cbID, body, presser, ok := h.owned(ctx, u)
	if !ok {
		return
	}
	idx, ok := parseIndex(strings.TrimPrefix(body, i18n.CbPAdminVis))
	if !ok {
		h.answerCb(ctx, cbID, "Ошибка данных")
		return
	}
	now, err := h.store.ToggleChapterVisible(idx)
	if err != nil {
		h.answerCb(ctx, cbID, "Ошибка: "+err.Error())
		return
	}
	h.answerCb(ctx, cbID, i18n.VisibilityAck(now))
	h.renderAdminChapters(ctx, chatID, msgID, pageOf(idx, i18n.AdminChaptersPerPage), false, presser)
}

func (h *Handler) handleAdminAskDeleteChapter(ctx context.Context, b *tgbot.Bot, u *models.Update) {
	chatID, msgID, cbID, body, presser, ok := h.owned(ctx, u)
	if !ok {
		return
	}
	h.answerCb(ctx, cbID, "")
	idx, ok := parseIndex(strings.TrimPrefix(body, i18n.CbPAdminDel))
	if !ok {
		return
	}
	h.renderConfirmChapterDelete(ctx, chatID, msgID, idx, presser)
}

func (h *Handler) handleAdminConfirmDeleteChapter(ctx context.Context, b *tgbot.Bot, u *models.Update) {
	chatID, msgID, cbID, body, presser, ok := h.owned(ctx, u)
	if !ok {
		return
	}
	idx, ok := parseIndex(strings.TrimPrefix(body, i18n.CbPAdminDelOK))
	if !ok {
		h.answerCb(ctx, cbID, "Ошибка данных")
		return
	}
	if err := h.store.DeleteChapter(idx); err != nil {
		h.answerCb(ctx, cbID, "Ошибка: "+err.Error())
		return
	}
	h.answerCb(ctx, cbID, "Глава удалена")
	h.renderAdminChapters(ctx, chatID, msgID, 0, false, presser)
}

func (h *Handler) handleAdminItemsPage(ctx context.Context, b *tgbot.Bot, u *models.Update) {
	chatID, msgID, cbID, body, presser, ok := h.owned(ctx, u)
	if !ok {
		return
	}
	parts := strings.Split(strings.TrimPrefix(body, i18n.CbPAdminItems), ":")
	if len(parts) != 2 {
		h.answerCb(ctx, cbID, "Ошибка данных")
		return
	}
	ch, ok := parseIndex(parts[0])
	pg, _ := strconv.Atoi(parts[1])
	if !ok {
		h.answerCb(ctx, cbID, "Ошибка данных")
		return
	}
	h.answerCb(ctx, cbID, "")
	h.renderAdminItems(ctx, chatID, msgID, ch, pg, false, presser)
}

func (h *Handler) handleAdminEditItem(ctx context.Context, b *tgbot.Bot, u *models.Update) {
	chatID, _, cbID, body, presser, ok := h.owned(ctx, u)
	if !ok {
		return
	}
	parts := strings.Split(strings.TrimPrefix(body, i18n.CbPAdminEditItem), ":")
	if len(parts) != 2 {
		h.answerCb(ctx, cbID, "Ошибка данных")
		return
	}
	ch, ok1 := parseIndex(parts[0])
	it, ok2 := parseIndex(parts[1])
	if !ok1 || !ok2 {
		h.answerCb(ctx, cbID, "Ошибка данных")
		return
	}
	h.sessions.Set(presser, Pending{Kind: PendingItem, Chapter: ch, Item: it, ChatID: chatID})
	h.answerCb(ctx, cbID, "Жду новый текст")
	m := i18n.EditItemPrompt(ch+1, it+1)
	h.sendText(ctx, chatID, m.Text)
}

func (h *Handler) handleAdminToggleItem(ctx context.Context, b *tgbot.Bot, u *models.Update) {
	chatID, msgID, cbID, body, presser, ok := h.owned(ctx, u)
	if !ok {
		return
	}
	parts := strings.Split(strings.TrimPrefix(body, i18n.CbPAdminVisItem), ":")
	if len(parts) != 2 {
		h.answerCb(ctx, cbID, "Ошибка данных")
		return
	}
	ch, ok1 := parseIndex(parts[0])
	it, ok2 := parseIndex(parts[1])
	if !ok1 || !ok2 {
		h.answerCb(ctx, cbID, "Ошибка данных")
		return
	}
	now, err := h.store.ToggleItemVisible(ch, it)
	if err != nil {
		h.answerCb(ctx, cbID, "Ошибка: "+err.Error())
		return
	}
	h.answerCb(ctx, cbID, i18n.VisibilityAck(now))
	h.renderAdminItems(ctx, chatID, msgID, ch, pageOf(it, i18n.ItemsPerPage), false, presser)
}

func (h *Handler) handleAdminAskDeleteItem(ctx context.Context, b *tgbot.Bot, u *models.Update) {
	chatID, msgID, cbID, body, presser, ok := h.owned(ctx, u)
	if !ok {
		return
	}
	parts := strings.Split(strings.TrimPrefix(body, i18n.CbPAdminDelItem), ":")
	if len(parts) != 2 {
		h.answerCb(ctx, cbID, "Ошибка данных")
		return
	}
	ch, ok1 := parseIndex(parts[0])
	it, ok2 := parseIndex(parts[1])
	if !ok1 || !ok2 {
		h.answerCb(ctx, cbID, "Ошибка данных")
		return
	}
	h.answerCb(ctx, cbID, "")
	h.renderConfirmItemDelete(ctx, chatID, msgID, ch, it, presser)
}

func (h *Handler) handleAdminConfirmDeleteItem(ctx context.Context, b *tgbot.Bot, u *models.Update) {
	chatID, msgID, cbID, body, presser, ok := h.owned(ctx, u)
	if !ok {
		return
	}
	parts := strings.Split(strings.TrimPrefix(body, i18n.CbPAdminDelItemOK), ":")
	if len(parts) != 2 {
		h.answerCb(ctx, cbID, "Ошибка данных")
		return
	}
	ch, ok1 := parseIndex(parts[0])
	it, ok2 := parseIndex(parts[1])
	if !ok1 || !ok2 {
		h.answerCb(ctx, cbID, "Ошибка данных")
		return
	}
	if err := h.store.DeleteItem(ch, it); err != nil {
		h.answerCb(ctx, cbID, "Ошибка: "+err.Error())
		return
	}
	h.answerCb(ctx, cbID, "Пункт удалён")
	h.renderAdminItems(ctx, chatID, msgID, ch, 0, false, presser)
}

func pageOf(idx, perPage int) int {
	if perPage <= 0 || idx < 0 {
		return 0
	}
	return idx / perPage
}

// renderAdminChapters shows chapters with title row + ✏️ 🗑 👁 row.
func (h *Handler) renderAdminChapters(ctx context.Context, chatID int64, msgIDOrPage int, page int, sendNew bool, owner int64) {
	var msgID int
	if !sendNew {
		msgID = msgIDOrPage
	}
	r := h.store.Snapshot()
	pages := pageCount(len(r.Chapters), i18n.AdminChaptersPerPage)
	page = clampPage(page, pages)
	lo := page * i18n.AdminChaptersPerPage
	hi := lo + i18n.AdminChaptersPerPage
	if hi > len(r.Chapters) {
		hi = len(r.Chapters)
	}
	chapters := make([]i18n.AdminChapter, 0, hi-lo)
	for ci := lo; ci < hi; ci++ {
		chapters = append(chapters, i18n.AdminChapter{Index: ci, Title: r.Chapters[ci].Title, Visible: r.Chapters[ci].Visible})
	}
	m := i18n.AdminChapterList(r.Visible, chapters, page, pages, owner)
	h.sendOrEdit(ctx, chatID, msgID, sendNew, m.Text, m.Markup)
}

// renderAdminItems shows items of one chapter with pagination.
func (h *Handler) renderAdminItems(ctx context.Context, chatID int64, msgID, chIdx, page int, sendNew bool, owner int64) {
	r := h.store.Snapshot()
	if chIdx < 0 || chIdx >= len(r.Chapters) {
		m := i18n.ChapterNotFound()
		h.sendOrEdit(ctx, chatID, msgID, sendNew, m.Text, m.Markup)
		return
	}
	ch := r.Chapters[chIdx]
	pages := pageCount(len(ch.Items), i18n.ItemsPerPage)
	page = clampPage(page, pages)
	lo := page * i18n.ItemsPerPage
	hi := lo + i18n.ItemsPerPage
	if hi > len(ch.Items) {
		hi = len(ch.Items)
	}
	items := make([]i18n.AdminItem, 0, hi-lo)
	for i := lo; i < hi; i++ {
		items = append(items, i18n.AdminItem{Index: i, Text: ch.Items[i].Text, Visible: ch.Items[i].Visible})
	}
	m := i18n.AdminItemList(chIdx, ch.Title, ch.Visible, items, page, pages, owner)
	h.sendOrEdit(ctx, chatID, msgID, sendNew, m.Text, m.Markup)
}

func (h *Handler) renderConfirmChapterDelete(ctx context.Context, chatID int64, msgID, idx int, owner int64) {
	r := h.store.Snapshot()
	title := ""
	if idx >= 0 && idx < len(r.Chapters) {
		title = r.Chapters[idx].Title
	}
	m := i18n.ConfirmDeleteChapter(idx, title, owner)
	h.editMsg(ctx, chatID, msgID, m.Text, m.Markup)
}

func (h *Handler) renderConfirmItemDelete(ctx context.Context, chatID int64, msgID, chIdx, itemIdx int, owner int64) {
	m := i18n.ConfirmDeleteItem(chIdx, itemIdx, owner)
	h.editMsg(ctx, chatID, msgID, m.Text, m.Markup)
}
