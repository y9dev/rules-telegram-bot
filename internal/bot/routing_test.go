package bot

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-telegram/bot/models"

	"rules-telegram-bot/internal/config"
	"rules-telegram-bot/internal/i18n"
	"rules-telegram-bot/internal/rules"
)

func newRoutingHandler(t *testing.T) *Handler {
	t.Helper()
	p := filepath.Join(t.TempDir(), "rules.json")
	if err := os.WriteFile(p, []byte(`{"title":"T","visible":true,"chapters":[]}`), 0o644); err != nil {
		t.Fatal(err)
	}
	store, err := rules.Load(p)
	if err != nil {
		t.Fatal(err)
	}
	return &Handler{
		cfg:      &config.Config{TargetChatID: -100, AdminIDs: map[int64]struct{}{7: {}}},
		store:    store,
		sessions: NewSessions(),
	}
}

func msgUpdate(text string, chatID, userID int64) *models.Update {
	return &models.Update{
		Message: &models.Message{
			Text: text,
			Chat: models.Chat{ID: chatID},
			From: &models.User{ID: userID},
		},
	}
}

func TestMatchRules(t *testing.T) {
	h := newRoutingHandler(t)
	cases := map[string]bool{
		"Правила": true, "правила": true, "ПРАВИЛА": true, "  Правила  ": true,
		"rules": true, "Rules": true, "RULES": true,
		"/rules": true, "/start": true, "/rules@mybot": true, "/start@mybot": true,
		"/rules_admin": false, "/rules_toggle": false,
		"правила676767": false, "привет": false, "": false, "правила!": false,
	}
	for text, want := range cases {
		if got := h.matchRules(msgUpdate(text, -100, 5)); got != want {
			t.Errorf("matchRules(%q) = %v, want %v", text, got, want)
		}
	}
}

func TestMatchAdminCodes(t *testing.T) {
	h := newRoutingHandler(t)
	if !h.matchAdminPanel(msgUpdate("/rules_admin", -100, 7)) {
		t.Error("admin panel must match")
	}
	if !h.matchAdminPanel(msgUpdate("/RULES_ADMIN", -100, 7)) {
		t.Error("admin panel must match case-insensitively")
	}
	if h.matchAdminPanel(msgUpdate("/rules_toggle", -100, 7)) {
		t.Error("toggle must not match panel")
	}
	if !h.matchGlobalToggle(msgUpdate("/rules_toggle", -100, 7)) {
		t.Error("global toggle must match")
	}
	if h.matchGlobalToggle(msgUpdate("правила", -100, 7)) {
		t.Error("rules must not match toggle")
	}
	if h.matchGlobalToggle(msgUpdate("/rules_admin", -100, 7)) {
		t.Error("panel must not match toggle")
	}
}

func TestMatchAdminTitle(t *testing.T) {
	h := newRoutingHandler(t)
	for _, text := range []string{"/admin_title", "/ADMIN_TITLE", "/admin_title New title here"} {
		if !h.matchAdminTitle(msgUpdate(text, -100, 7)) {
			t.Errorf("matchAdminTitle(%q) must match", text)
		}
	}
	for _, text := range []string{"/admin_titleX", "/admin", "правила", "/rules"} {
		if h.matchAdminTitle(msgUpdate(text, -100, 7)) {
			t.Errorf("matchAdminTitle(%q) must not match", text)
		}
	}
}

// Pending input is registered first: while an edit is pending it must win
// over command matchers so the next message becomes the new item text.
func TestMatchPendingInputPriority(t *testing.T) {
	h := newRoutingHandler(t)
	u := msgUpdate("Правила", -100, 7)
	if h.matchPendingInput(u) {
		t.Fatal("no session: must not match")
	}
	h.sessions.Set(7, Pending{Kind: PendingItem, Chapter: 0, Item: 0})
	if !h.matchPendingInput(u) {
		t.Fatal("with session: must match even for command text")
	}
	if h.matchPendingInput(msgUpdate("Правила", -100, 9)) {
		t.Fatal("other user: must not match")
	}
}

// Callback payloads built by i18n must hit exactly one registered prefix.
func TestCallbackPrefixesDisjoint(t *testing.T) {
	const owner int64 = 777
	samples := []string{
		i18n.CbUserChapter(3, owner), i18n.CbUserPage(1, owner), i18n.CbUserBack(owner),
		i18n.CbAdminBack(owner), i18n.CbAdminOpen(0, owner), i18n.CbAdminPage(2, owner),
		i18n.CbAdminToggleChapter(1, owner), i18n.CbAdminDeleteChapter(1, owner),
		i18n.CbAdminDeleteChapterOK(1, owner), i18n.CbAdminItemsPage(1, 2, owner),
		i18n.CbAdminEditItem(1, 2, owner), i18n.CbAdminToggleItem(1, 2, owner),
		i18n.CbAdminDeleteItem(1, 2, owner), i18n.CbAdminDeleteItemOK(1, 2, owner),
	}
	for _, s := range samples {
		hits := 0
		for _, p := range i18n.CallbackPrefixes {
			if strings.HasPrefix(s, p) {
				hits++
			}
		}
		if hits != 1 {
			t.Errorf("callback %q matches %d prefixes, want exactly 1", s, hits)
		}
	}
}

// Only the menu owner may press its buttons.
func TestCheckOwner(t *testing.T) {
	body, ok := checkOwner(i18n.CbAdminDeleteItemOK(1, 2, 777), 777)
	if !ok || body != "a:idok:1:2" {
		t.Fatalf("owner must pass: %q %v", body, ok)
	}
	if _, ok := checkOwner(i18n.CbAdminDeleteItemOK(1, 2, 777), 888); ok {
		t.Fatal("foreigner must be rejected")
	}
	for _, bad := range []string{"noop", "u:ch:3", "a:del:1:0"} {
		if _, ok := checkOwner(bad, 777); ok {
			t.Errorf("checkOwner(%q) must fail", bad)
		}
	}
	if _, ok := checkOwner(i18n.CbUserPage(0, 777), 0); ok {
		t.Fatal("zero presser must be rejected")
	}
}
