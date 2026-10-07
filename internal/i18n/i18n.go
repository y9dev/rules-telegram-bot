// Package i18n centralizes every user-visible message object: text plus the
// optional inline keyboard. Handlers only pick a builder and pass the result
// to the send/edit glue in the bot package. Plain text is used (no ParseMode)
// so rule content with <>& never breaks rendering.
package i18n

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/go-telegram/bot/models"
)

// Pagination limits. Telegram caps a message at 100 buttons, so lists are
// paged: chapters 15/8 per page, items 10 per page.
const (
	UserChaptersPerPage  = 15
	AdminChaptersPerPage = 8
	ItemsPerPage         = 10
)

// Message is one sendable unit: text plus optional inline keyboard.
type Message struct {
	Text   string
	Markup *models.InlineKeyboardMarkup // nil = no keyboard
}

func button(text, data string) models.InlineKeyboardButton {
	return models.InlineKeyboardButton{Text: text, CallbackData: data}
}

func markup(rows ...[]models.InlineKeyboardButton) *models.InlineKeyboardMarkup {
	return &models.InlineKeyboardMarkup{InlineKeyboard: rows}
}

// pageNav computes prev/next callback payloads via at(); empty means no arrow.
func pageNav(page, pages int, at func(int) string) (prev, next string) {
	if page > 0 {
		prev = at(page - 1)
	}
	if page < pages-1 {
		next = at(page + 1)
	}
	return prev, next
}

// pager appends a "⬅️ x/y ➡️" row; empty prev/next hides that arrow.
func pager(rows [][]models.InlineKeyboardButton, page, pages int, prev, next string) [][]models.InlineKeyboardButton {
	if pages <= 1 {
		return rows
	}
	var nav []models.InlineKeyboardButton
	if prev != "" {
		nav = append(nav, button("⬅️", prev))
	}
	nav = append(nav, button(fmt.Sprintf("%d/%d", page+1, pages), CbNoop))
	if next != "" {
		nav = append(nav, button("➡️", next))
	}
	return append(rows, nav)
}

// preview shortens long item text for an inline button label.
func preview(s string, max int) string {
	s = strings.TrimSpace(strings.ReplaceAll(s, "\n", " "))
	if utf8.RuneCountInString(s) <= max {
		if s == "" {
			return "…"
		}
		return s
	}
	return string([]rune(s)[:max-1]) + "…"
}

// eye returns the visibility toggle icon.
func eye(visible bool) string {
	if visible {
		return "👁"
	}
	return "🚫"
}
