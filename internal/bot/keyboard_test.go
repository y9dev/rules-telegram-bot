package bot

import "testing"

func TestParseIndex(t *testing.T) {
	if n, ok := parseIndex("12"); !ok || n != 12 {
		t.Fatalf("got %d %v", n, ok)
	}
	if _, ok := parseIndex("-1"); ok {
		t.Fatal("negative must fail")
	}
	if _, ok := parseIndex("abc"); ok {
		t.Fatal("non-numeric must fail")
	}
}

func TestSessions(t *testing.T) {
	s := NewSessions()
	s.Set(1, Pending{Kind: PendingItem, Chapter: 2, Item: 3})
	if !s.Has(1) {
		t.Fatal("Has must report pending edit")
	}
	p, ok := s.Take(1)
	if !ok || p.Kind != PendingItem || p.Chapter != 2 || p.Item != 3 {
		t.Fatalf("got %+v %v", p, ok)
	}
	if _, ok := s.Take(1); ok {
		t.Fatal("must be cleared after take")
	}
	if s.Has(1) {
		t.Fatal("Has must be false after take")
	}
}

func TestPageOf(t *testing.T) {
	if pageOf(0, 10) != 0 || pageOf(10, 10) != 1 || pageOf(25, 10) != 2 {
		t.Fatal("bad page calc")
	}
}

func TestPageCountClamp(t *testing.T) {
	if pageCount(0, 10) != 1 || pageCount(25, 10) != 3 {
		t.Fatal("bad page count")
	}
	if clampPage(-1, 3) != 0 || clampPage(9, 3) != 2 || clampPage(1, 3) != 1 {
		t.Fatal("bad clamp")
	}
}
