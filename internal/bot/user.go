package bot

import (
	"context"
	"log"
	"strconv"
	"strings"

	tgbot "github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"

	"rules-telegram-bot/internal/i18n"
)

// handleUserChapter opens one chapter from the "u:ch:<idx>:<owner>" button.
func (h *Handler) handleUserChapter(ctx context.Context, b *tgbot.Bot, u *models.Update) {
	chatID, msgID, cbID, data := h.callbackTarget(u)
	presser := h.updateUserID(u)
	body, ok := checkOwner(data, presser)
	if !ok {
		h.answerCb(ctx, cbID, i18n.NotYourMenuAck())
		return
	}
	h.answerCb(ctx, cbID, "")
	idx, ok := parseIndex(strings.TrimPrefix(body, i18n.CbPUserChapter))
	if !ok {
		return
	}
	h.renderUserChapter(ctx, chatID, msgID, idx, presser)
}

// handleUserPage flips the chapter list page from the "u:pg:<n>:<owner>" button.
func (h *Handler) handleUserPage(ctx context.Context, b *tgbot.Bot, u *models.Update) {
	chatID, msgID, cbID, data := h.callbackTarget(u)
	presser := h.updateUserID(u)
	body, ok := checkOwner(data, presser)
	if !ok {
		h.answerCb(ctx, cbID, i18n.NotYourMenuAck())
		return
	}
	h.answerCb(ctx, cbID, "")
	p, _ := strconv.Atoi(strings.TrimPrefix(body, i18n.CbPUserPage))
	h.renderUserChaptersEdit(ctx, chatID, msgID, p, presser)
}

// renderUserChapters sends (sendNew) the rules header with chapter buttons.
func (h *Handler) renderUserChapters(ctx context.Context, chatID int64, page int, sendNew bool, owner int64) {
	r := h.store.Snapshot()
	if !r.Visible {
		m := i18n.RulesHidden()
		h.sendOrEdit(ctx, chatID, 0, sendNew, m.Text, m.Markup)
		return
	}
	visible := visibleChapterIdx(r.Chapters)
	pages := pageCount(len(visible), i18n.UserChaptersPerPage)
	page = clampPage(page, pages)
	text, markup := rulesListText(r.Title, r.Chapters, visible, page, pages, owner)
	h.sendOrEdit(ctx, chatID, 0, sendNew, text, markup)
}

// renderUserChapter shows numbered items of one chapter.
func (h *Handler) renderUserChapter(ctx context.Context, chatID int64, msgID int, chIdx int, owner int64) {
	r := h.store.Snapshot()
	if !r.Visible {
		m := i18n.RulesHidden()
		h.editMsg(ctx, chatID, msgID, m.Text, m.Markup)
		return
	}
	if chIdx < 0 || chIdx >= len(r.Chapters) || !r.Chapters[chIdx].Visible {
		m := i18n.ChapterUnavailable()
		h.editMsg(ctx, chatID, msgID, m.Text, m.Markup)
		return
	}
	ch := r.Chapters[chIdx]
	var lines []i18n.RuleLine
	for i, it := range ch.Items {
		if !it.Visible {
			continue
		}
		lines = append(lines, i18n.RuleLine{ChapterNum: chIdx + 1, ItemNum: i + 1, Text: it.Text})
	}
	m := i18n.ChapterDetail(chIdx+1, ch.Title, lines, owner)
	h.editMsg(ctx, chatID, msgID, m.Text, m.Markup)
}

// renderUserChaptersEdit edits the existing message with a chapter list page.
func (h *Handler) renderUserChaptersEdit(ctx context.Context, chatID int64, msgID, page int, owner int64) {
	r := h.store.Snapshot()
	if !r.Visible {
		m := i18n.RulesHidden()
		h.editMsg(ctx, chatID, msgID, m.Text, m.Markup)
		return
	}
	visible := visibleChapterIdx(r.Chapters)
	pages := pageCount(len(visible), i18n.UserChaptersPerPage)
	page = clampPage(page, pages)
	text, markup := rulesListText(r.Title, r.Chapters, visible, page, pages, owner)
	h.editMsg(ctx, chatID, msgID, text, markup)
}

func (h *Handler) sendOrEdit(ctx context.Context, chatID int64, msgID int, sendNew bool, text string, markup *models.InlineKeyboardMarkup) {
	if sendNew {
		params := &tgbot.SendMessageParams{ChatID: chatID, Text: text}
		if markup != nil {
			params.ReplyMarkup = markup
		}
		if _, err := h.bot.SendMessage(ctx, params); err != nil {
			log.Printf("send message: %v", err)
		}
		return
	}
	h.editMsg(ctx, chatID, msgID, text, markup)
}

func (h *Handler) editMsg(ctx context.Context, chatID int64, msgID int, text string, markup *models.InlineKeyboardMarkup) {
	params := &tgbot.EditMessageTextParams{ChatID: chatID, MessageID: msgID, Text: text}
	if markup != nil {
		params.ReplyMarkup = markup
	}
	if _, err := h.bot.EditMessageText(ctx, params); err != nil {
		log.Printf("edit message: %v", err)
	}
}
