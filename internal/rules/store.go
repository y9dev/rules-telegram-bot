// Package rules owns the rules.json data model and thread-safe file persistence.
package rules

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// Item is a single numbered rule inside a chapter.
type Item struct {
	Text    string `json:"text"`
	Visible bool   `json:"visible"`
}

// Chapter groups items under one inline button.
type Chapter struct {
	Title   string `json:"title"`
	Visible bool   `json:"visible"`
	Items   []Item `json:"items"`
}

// Rules is the whole document: header text + chapters + global visibility.
type Rules struct {
	Title    string    `json:"title"`
	Visible  bool      `json:"visible"`
	Chapters []Chapter `json:"chapters"`
}

// UnmarshalJSON defaults missing visible flags to true for backward compatibility.
func (r *Rules) UnmarshalJSON(data []byte) error {
	type chapterRaw struct {
		Title   string `json:"title"`
		Visible *bool  `json:"visible"`
		Items   []struct {
			Text    string `json:"text"`
			Visible *bool  `json:"visible"`
		} `json:"items"`
	}
	var raw struct {
		Title    string       `json:"title"`
		Visible  *bool        `json:"visible"`
		Chapters []chapterRaw `json:"chapters"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	out := Rules{Title: raw.Title, Visible: true}
	if raw.Visible != nil {
		out.Visible = *raw.Visible
	}
	for _, c := range raw.Chapters {
		ch := Chapter{Title: c.Title, Visible: true}
		if c.Visible != nil {
			ch.Visible = *c.Visible
		}
		for _, it := range c.Items {
			item := Item{Text: it.Text, Visible: true}
			if it.Visible != nil {
				item.Visible = *it.Visible
			}
			ch.Items = append(ch.Items, item)
		}
		if ch.Items == nil {
			ch.Items = []Item{}
		}
		out.Chapters = append(out.Chapters, ch)
	}
	if out.Chapters == nil {
		out.Chapters = []Chapter{}
	}
	*r = out
	return nil
}

// Store is a mutex-guarded Rules document backed by a JSON file.
type Store struct {
	mu   sync.RWMutex
	path string
	data Rules
}

// Load reads path into memory. Missing file yields an empty visible document.
func Load(path string) (*Store, error) {
	s := &Store{path: path}
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			s.data = Rules{Visible: true, Chapters: []Chapter{}}
			return s, nil
		}
		return nil, fmt.Errorf("read rules file: %w", err)
	}
	if len(raw) == 0 {
		s.data = Rules{Visible: true, Chapters: []Chapter{}}
		return s, nil
	}
	var r Rules
	if err := json.Unmarshal(raw, &r); err != nil {
		return nil, fmt.Errorf("parse rules file: %w", err)
	}
	s.data = r
	return s, nil
}

// Snapshot returns a deep copy for safe rendering.
func (s *Store) Snapshot() Rules {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := s.data
	out.Chapters = append([]Chapter(nil), s.data.Chapters...)
	for i := range out.Chapters {
		out.Chapters[i].Items = append([]Item(nil), s.data.Chapters[i].Items...)
	}
	return out
}

// Update mutates the document under a write lock and persists it atomically.
func (s *Store) Update(fn func(*Rules) error) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := fn(&s.data); err != nil {
		return err
	}
	return s.saveLocked()
}

func (s *Store) saveLocked() error {
	raw, err := json.MarshalIndent(s.data, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal rules: %w", err)
	}
	raw = append(raw, '\n')
	dir := filepath.Dir(s.path)
	tmp, err := os.CreateTemp(dir, ".rules-*.tmp")
	if err != nil {
		return fmt.Errorf("create temp rules file: %w", err)
	}
	tmpName := tmp.Name()
	if _, err := tmp.Write(raw); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpName)
		return fmt.Errorf("write temp rules file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpName)
		return fmt.Errorf("close temp rules file: %w", err)
	}
	if err := os.Rename(tmpName, s.path); err != nil {
		_ = os.Remove(tmpName)
		return fmt.Errorf("replace rules file: %w", err)
	}
	return nil
}

// ToggleGlobalVisible flips Rules.Visible.
func (s *Store) ToggleGlobalVisible() (bool, error) {
	var now bool
	err := s.Update(func(r *Rules) error {
		r.Visible = !r.Visible
		now = r.Visible
		return nil
	})
	return now, err
}

// ToggleChapterVisible flips Chapters[idx].Visible.
func (s *Store) ToggleChapterVisible(idx int) (bool, error) {
	var now bool
	err := s.Update(func(r *Rules) error {
		if idx < 0 || idx >= len(r.Chapters) {
			return fmt.Errorf("chapter %d out of range", idx)
		}
		r.Chapters[idx].Visible = !r.Chapters[idx].Visible
		now = r.Chapters[idx].Visible
		return nil
	})
	return now, err
}

// ToggleItemVisible flips Chapters[c].Items[i].Visible.
func (s *Store) ToggleItemVisible(chIdx, itemIdx int) (bool, error) {
	var now bool
	err := s.Update(func(r *Rules) error {
		if chIdx < 0 || chIdx >= len(r.Chapters) {
			return fmt.Errorf("chapter %d out of range", chIdx)
		}
		items := r.Chapters[chIdx].Items
		if itemIdx < 0 || itemIdx >= len(items) {
			return fmt.Errorf("item %d out of range", itemIdx)
		}
		r.Chapters[chIdx].Items[itemIdx].Visible = !items[itemIdx].Visible
		now = r.Chapters[chIdx].Items[itemIdx].Visible
		return nil
	})
	return now, err
}

// DeleteChapter removes Chapters[idx].
func (s *Store) DeleteChapter(idx int) error {
	return s.Update(func(r *Rules) error {
		if idx < 0 || idx >= len(r.Chapters) {
			return fmt.Errorf("chapter %d out of range", idx)
		}
		r.Chapters = append(r.Chapters[:idx], r.Chapters[idx+1:]...)
		return nil
	})
}

// DeleteItem removes Chapters[c].Items[i].
func (s *Store) DeleteItem(chIdx, itemIdx int) error {
	return s.Update(func(r *Rules) error {
		if chIdx < 0 || chIdx >= len(r.Chapters) {
			return fmt.Errorf("chapter %d out of range", chIdx)
		}
		items := r.Chapters[chIdx].Items
		if itemIdx < 0 || itemIdx >= len(items) {
			return fmt.Errorf("item %d out of range", itemIdx)
		}
		r.Chapters[chIdx].Items = append(items[:itemIdx], items[itemIdx+1:]...)
		return nil
	})
}

// SetTitle replaces the top-level rules title (admin /admin_title).
func (s *Store) SetTitle(title string) error {
	return s.Update(func(r *Rules) error {
		r.Title = title
		return nil
	})
}

// SetItemText replaces the text of one item (admin edit).
func (s *Store) SetItemText(chIdx, itemIdx int, text string) error {
	return s.Update(func(r *Rules) error {
		if chIdx < 0 || chIdx >= len(r.Chapters) {
			return fmt.Errorf("chapter %d out of range", chIdx)
		}
		items := r.Chapters[chIdx].Items
		if itemIdx < 0 || itemIdx >= len(items) {
			return fmt.Errorf("item %d out of range", itemIdx)
		}
		r.Chapters[chIdx].Items[itemIdx].Text = text
		return nil
	})
}
