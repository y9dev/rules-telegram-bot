package i18n

import (
	"strings"
	"testing"
)

const testOwner int64 = 777

func countButtons(m Message) int {
	n := 0
	if m.Markup == nil {
		return 0
	}
	for _, r := range m.Markup.InlineKeyboard {
		n += len(r)
	}
	return n
}

func TestRulesList(t *testing.T) {
	m := RulesList("", []ChapterButton{{Index: 0, Title: "A"}, {Index: 1, Title: "B"}}, 0, 1, testOwner)
	if !strings.HasPrefix(m.Text, DefaultRulesTitle) {
		t.Fatalf("default title missing: %q", m.Text)
	}
	if countButtons(m) != 2 {
		t.Fatalf("want 2 buttons, got %d", countButtons(m))
	}
	if got := m.Markup.InlineKeyboard[0][0].CallbackData; got != "u:ch:0:777" {
		t.Fatalf("owner missing in callback: %q", got)
	}
}

func TestRulesListEmpty(t *testing.T) {
	m := RulesList("T", nil, 0, 1, testOwner)
	if !strings.Contains(m.Text, "Пока нет доступных глав") || m.Markup != nil {
		t.Fatalf("got %+v", m)
	}
}

func TestRulesListPaging(t *testing.T) {
	var ch []ChapterButton
	for i := 0; i < 3; i++ {
		ch = append(ch, ChapterButton{Index: i, Title: "C"})
	}
	m := RulesList("T", ch, 1, 3, testOwner)
	if !strings.Contains(m.Text, "Страница 2/3") {
		t.Fatalf("no page suffix: %q", m.Text)
	}
	// 3 chapter rows + 1 nav row (prev + label + next = 3 buttons).
	if countButtons(m) != 6 {
		t.Fatalf("want 6 buttons, got %d", countButtons(m))
	}
}

func TestChapterDetail(t *testing.T) {
	m := ChapterDetail(2, "Mod", []RuleLine{{ChapterNum: 2, ItemNum: 4, Text: "no spam"}}, testOwner)
	if !strings.Contains(m.Text, "📌 2. Mod") || !strings.Contains(m.Text, "2.4 no spam") {
		t.Fatalf("got %q", m.Text)
	}
	if got := m.Markup.InlineKeyboard[0][0].CallbackData; got != "u:back:777" {
		t.Fatalf("owner missing in back button: %q", got)
	}
	empty := ChapterDetail(1, "G", nil, testOwner)
	if !strings.Contains(empty.Text, "Пока нет доступных пунктов") {
		t.Fatalf("got %q", empty.Text)
	}
}

func TestAdminChapterList(t *testing.T) {
	m := AdminChapterList(false, []AdminChapter{{Index: 0, Title: "G", Visible: false}}, 0, 1, testOwner)
	if !strings.Contains(m.Text, "СКРЫТЫ глобально") {
		t.Fatalf("global warning missing: %q", m.Text)
	}
	// title row + action row.
	if countButtons(m) != 4 {
		t.Fatalf("want 4 buttons, got %d", countButtons(m))
	}
	row := m.Markup.InlineKeyboard[1]
	if row[0].Text != "✏️" || row[1].Text != "🗑" || row[2].Text != "🚫" {
		t.Fatalf("bad action row: %+v", row)
	}
	if got := row[1].CallbackData; got != "a:del:0:777" {
		t.Fatalf("owner missing in callback: %q", got)
	}
}

func TestAdminItemList(t *testing.T) {
	m := AdminItemList(0, "G", true, []AdminItem{{Index: 0, Text: "hello world", Visible: true}}, 0, 1, testOwner)
	if !strings.Contains(m.Text, "🛠 Глава 1. G") {
		t.Fatalf("got %q", m.Text)
	}
	last := m.Markup.InlineKeyboard[len(m.Markup.InlineKeyboard)-1][0]
	if last.Text != "⬅️ К главам" || last.CallbackData != CbAdminBack(testOwner) {
		t.Fatalf("bad back row: %+v", last)
	}
}

func TestConfirmDelete(t *testing.T) {
	m := ConfirmDeleteChapter(2, "G", testOwner)
	if !strings.Contains(m.Text, "главу 3. G") {
		t.Fatalf("got %q", m.Text)
	}
	mi := ConfirmDeleteItem(1, 9, testOwner)
	if !strings.Contains(mi.Text, "пункт 2.10") {
		t.Fatalf("got %q", mi.Text)
	}
	// item 9 with 10 per page goes back to page 0.
	if got := mi.Markup.InlineKeyboard[0][1].CallbackData; got != "a:i:1:0:777" {
		t.Fatalf("got %q", got)
	}
}

func TestCallbackConstructors(t *testing.T) {
	cases := map[string]string{
		CbUserChapter(3, testOwner):          "u:ch:3:777",
		CbUserPage(1, testOwner):             "u:pg:1:777",
		CbUserBack(testOwner):                "u:back:777",
		CbAdminOpen(0, testOwner):            "a:open:0:777",
		CbAdminItemsPage(1, 2, testOwner):    "a:i:1:2:777",
		CbAdminEditItem(1, 2, testOwner):     "a:ie:1:2:777",
		CbAdminDeleteItemOK(1, 2, testOwner): "a:idok:1:2:777",
	}
	for got, want := range cases {
		if got != want {
			t.Errorf("got %q want %q", got, want)
		}
	}
	for _, c := range []string{CbUserChapter(99, 1234567890), CbAdminDeleteItemOK(99, 99, 1234567890)} {
		if len(c) > 64 {
			t.Errorf("callback %q exceeds 64 bytes", c)
		}
	}
}

func TestOwnerSuffixRoundTrip(t *testing.T) {
	body, owner, ok := SplitOwner(CbAdminToggleItem(4, 5, 12345))
	if !ok || owner != 12345 || body != "a:iv:4:5" {
		t.Fatalf("got %q %d %v", body, owner, ok)
	}
	for _, bad := range []string{"noop", "a:del:1:abc", "a:del:1:0", "a:del:1:-5", "u:back:"} {
		if _, _, ok := SplitOwner(bad); ok {
			t.Errorf("SplitOwner(%q) must fail", bad)
		}
	}
}

func TestCallbackPrefixesDisjoint(t *testing.T) {
	samples := []string{
		CbUserChapter(3, testOwner), CbUserPage(1, testOwner), CbUserBack(testOwner),
		CbAdminBack(testOwner), CbAdminOpen(0, testOwner), CbAdminPage(2, testOwner),
		CbAdminToggleChapter(1, testOwner), CbAdminDeleteChapter(1, testOwner),
		CbAdminDeleteChapterOK(1, testOwner), CbAdminItemsPage(1, 2, testOwner),
		CbAdminEditItem(1, 2, testOwner), CbAdminToggleItem(1, 2, testOwner),
		CbAdminDeleteItem(1, 2, testOwner), CbAdminDeleteItemOK(1, 2, testOwner),
	}
	for _, s := range samples {
		hits := 0
		for _, p := range CallbackPrefixes {
			if strings.HasPrefix(s, p) {
				hits++
			}
		}
		if hits != 1 {
			t.Errorf("callback %q matches %d prefixes, want exactly 1", s, hits)
		}
	}
}

func TestPreviewEye(t *testing.T) {
	if preview("hello", 10) != "hello" || preview("hello world foo", 5) != "hell…" {
		t.Fatal("bad preview")
	}
	if eye(true) != "👁" || eye(false) != "🚫" {
		t.Fatal("bad eye")
	}
}

func TestTitleMessages(t *testing.T) {
	p := EditTitlePrompt("📜 Old")
	if !strings.Contains(p.Text, "новый заголовок") || !strings.Contains(p.Text, "📜 Old") {
		t.Fatalf("got %q", p.Text)
	}
	if TitleUpdated().Text == "" {
		t.Fatal("empty confirmation")
	}
}
