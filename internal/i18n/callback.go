package i18n

import (
	"fmt"
	"strconv"
	"strings"
)

// Callback data patterns. Every interactive payload ends with ":<ownerID>" —
// the Telegram user who opened the menu. Only the owner may press its buttons.
// (Telegram limit: 64 bytes; even with a 10-digit ID payloads stay ~25.)
const (
	CbNoop = "noop" // label buttons only; exact match, no owner needed

	CbPUserBack  = "u:back"
	CbPAdminBack = "a:back"

	CbPUserChapter    = "u:ch:"
	CbPUserPage       = "u:pg:"
	CbPAdminOpen      = "a:open:"
	CbPAdminPage      = "a:pg:"
	CbPAdminVis       = "a:vis:"
	CbPAdminDel       = "a:del:"
	CbPAdminDelOK     = "a:delok:"
	CbPAdminItems     = "a:i:"
	CbPAdminEditItem  = "a:ie:"
	CbPAdminVisItem   = "a:iv:"
	CbPAdminDelItem   = "a:id:"
	CbPAdminDelItemOK = "a:idok:"
)

// CallbackPrefixes lists every MatchTypePrefix pattern registered for
// callbacks. They are disjoint by construction — no payload matches two.
var CallbackPrefixes = []string{
	CbPUserBack, CbPAdminBack,
	CbPUserChapter, CbPUserPage,
	CbPAdminOpen, CbPAdminPage, CbPAdminVis, CbPAdminDel, CbPAdminDelOK,
	CbPAdminItems, CbPAdminEditItem, CbPAdminVisItem, CbPAdminDelItem, CbPAdminDelItemOK,
}

// OwnerSuffix appends ":<owner>" to a base payload.
func OwnerSuffix(data string, owner int64) string {
	return data + ":" + strconv.FormatInt(owner, 10)
}

// SplitOwner splits "base:<ownerID>" at the last colon.
func SplitOwner(data string) (body string, owner int64, ok bool) {
	idx := strings.LastIndex(data, ":")
	if idx < 0 {
		return "", 0, false
	}
	id, err := strconv.ParseInt(data[idx+1:], 10, 64)
	if err != nil || id <= 0 {
		return "", 0, false
	}
	return data[:idx], id, true
}

func CbUserBack(owner int64) string  { return OwnerSuffix(CbPUserBack, owner) }
func CbAdminBack(owner int64) string { return OwnerSuffix(CbPAdminBack, owner) }

func CbUserChapter(idx int, owner int64) string {
	return OwnerSuffix(fmt.Sprintf("%s%d", CbPUserChapter, idx), owner)
}
func CbUserPage(page int, owner int64) string {
	return OwnerSuffix(fmt.Sprintf("%s%d", CbPUserPage, page), owner)
}
func CbAdminOpen(idx int, owner int64) string {
	return OwnerSuffix(fmt.Sprintf("%s%d", CbPAdminOpen, idx), owner)
}
func CbAdminPage(page int, owner int64) string {
	return OwnerSuffix(fmt.Sprintf("%s%d", CbPAdminPage, page), owner)
}
func CbAdminToggleChapter(idx int, owner int64) string {
	return OwnerSuffix(fmt.Sprintf("%s%d", CbPAdminVis, idx), owner)
}
func CbAdminDeleteChapter(idx int, owner int64) string {
	return OwnerSuffix(fmt.Sprintf("%s%d", CbPAdminDel, idx), owner)
}
func CbAdminDeleteChapterOK(idx int, owner int64) string {
	return OwnerSuffix(fmt.Sprintf("%s%d", CbPAdminDelOK, idx), owner)
}
func CbAdminItemsPage(ch, page int, owner int64) string {
	return OwnerSuffix(fmt.Sprintf("%s%d:%d", CbPAdminItems, ch, page), owner)
}
func CbAdminEditItem(ch, item int, owner int64) string {
	return OwnerSuffix(fmt.Sprintf("%s%d:%d", CbPAdminEditItem, ch, item), owner)
}
func CbAdminToggleItem(ch, item int, owner int64) string {
	return OwnerSuffix(fmt.Sprintf("%s%d:%d", CbPAdminVisItem, ch, item), owner)
}
func CbAdminDeleteItem(ch, item int, owner int64) string {
	return OwnerSuffix(fmt.Sprintf("%s%d:%d", CbPAdminDelItem, ch, item), owner)
}
func CbAdminDeleteItemOK(ch, item int, owner int64) string {
	return OwnerSuffix(fmt.Sprintf("%s%d:%d", CbPAdminDelItemOK, ch, item), owner)
}
