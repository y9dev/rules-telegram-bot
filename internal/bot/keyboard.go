package bot

import (
	"strconv"
	"strings"

	"github.com/go-telegram/bot/models"

	"rules-telegram-bot/internal/i18n"
	"rules-telegram-bot/internal/rules"
)

// parseIndex parses a non-negative array index from callback data.
func parseIndex(s string) (int, bool) {
	n, err := strconv.Atoi(strings.TrimSpace(s))
	if err != nil || n < 0 {
		return 0, false
	}
	return n, true
}

// visibleChapterIdx returns indexes of visible chapters.
func visibleChapterIdx(chapters []rules.Chapter) []int {
	var out []int
	for i := range chapters {
		if chapters[i].Visible {
			out = append(out, i)
		}
	}
	return out
}

// pageCount returns the number of pages (at least 1).
func pageCount(total, perPage int) int {
	if total <= 0 {
		return 1
	}
	return (total + perPage - 1) / perPage
}

// clampPage keeps page inside [0, pages).
func clampPage(page, pages int) int {
	if page < 0 {
		return 0
	}
	if page >= pages {
		return pages - 1
	}
	return page
}

// rulesListText slices the visible chapters for page and builds the message.
func rulesListText(title string, chapters []rules.Chapter, visible []int, page, pages int, owner int64) (string, *models.InlineKeyboardMarkup) {
	lo := page * i18n.UserChaptersPerPage
	hi := lo + i18n.UserChaptersPerPage
	if hi > len(visible) {
		hi = len(visible)
	}
	if lo > hi {
		lo = hi
	}
	btns := make([]i18n.ChapterButton, 0, hi-lo)
	for _, ci := range visible[lo:hi] {
		btns = append(btns, i18n.ChapterButton{Index: ci, Title: chapters[ci].Title})
	}
	m := i18n.RulesList(title, btns, page, pages, owner)
	return m.Text, m.Markup
}

// checkOwner verifies the ":<ownerID>" suffix against the pressing user.
// Pure function (no bot calls) so it is unit-testable; callers answer the
// callback themselves when it fails.
func checkOwner(data string, presser int64) (body string, ok bool) {
	body, owner, ok := i18n.SplitOwner(data)
	if !ok || presser == 0 || owner != presser {
		return "", false
	}
	return body, true
}
