package rules

import (
	"os"
	"path/filepath"
	"testing"
)

func writeTemp(t *testing.T, content string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "rules.json")
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestMissingVisibleDefaultsTrue(t *testing.T) {
	p := writeTemp(t, `{"title":"T","chapters":[{"title":"C","items":[{"text":"x"}]}]}`)
	s, err := Load(p)
	if err != nil {
		t.Fatal(err)
	}
	r := s.Snapshot()
	if !r.Visible || !r.Chapters[0].Visible || !r.Chapters[0].Items[0].Visible {
		t.Fatalf("expected all visible default true: %+v", r)
	}
}

func TestToggleDeleteSetItem(t *testing.T) {
	p := writeTemp(t, `{"title":"T","visible":true,"chapters":[{"title":"C","visible":true,"items":[{"text":"a","visible":true},{"text":"b","visible":true}]}]}`)
	s, err := Load(p)
	if err != nil {
		t.Fatal(err)
	}
	if now, err := s.ToggleChapterVisible(0); err != nil || now {
		t.Fatalf("toggle chapter: now=%v err=%v", now, err)
	}
	if now, err := s.ToggleItemVisible(0, 1); err != nil || now {
		t.Fatalf("toggle item: now=%v err=%v", now, err)
	}
	if err := s.SetItemText(0, 0, "new"); err != nil {
		t.Fatal(err)
	}
	if got := s.Snapshot().Chapters[0].Items[0].Text; got != "new" {
		t.Fatalf("got %q", got)
	}
	if err := s.DeleteItem(0, 1); err != nil {
		t.Fatal(err)
	}
	if n := len(s.Snapshot().Chapters[0].Items); n != 1 {
		t.Fatalf("expected 1 item, got %d", n)
	}
	if now, err := s.ToggleGlobalVisible(); err != nil || now {
		t.Fatalf("toggle global: now=%v err=%v", now, err)
	}
	if err := s.DeleteChapter(0); err != nil {
		t.Fatal(err)
	}
	if n := len(s.Snapshot().Chapters); n != 0 {
		t.Fatalf("expected 0 chapters, got %d", n)
	}
	// persisted?
	raw, _ := os.ReadFile(p)
	if len(raw) == 0 {
		t.Fatal("rules file empty after updates")
	}
}

func TestSetTitle(t *testing.T) {
	p := writeTemp(t, `{"title":"Old","visible":true,"chapters":[]}`)
	s, err := Load(p)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.SetTitle("📜 New title"); err != nil {
		t.Fatal(err)
	}
	if got := s.Snapshot().Title; got != "📜 New title" {
		t.Fatalf("got %q", got)
	}
	reloaded, err := Load(p)
	if err != nil {
		t.Fatal(err)
	}
	if got := reloaded.Snapshot().Title; got != "📜 New title" {
		t.Fatalf("not persisted, got %q", got)
	}
}

func TestOutOfRange(t *testing.T) {
	p := writeTemp(t, `{"title":"T","chapters":[]}`)
	s, err := Load(p)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.DeleteChapter(0); err == nil {
		t.Fatal("expected error")
	}
	if err := s.SetItemText(0, 0, "x"); err == nil {
		t.Fatal("expected error")
	}
}
