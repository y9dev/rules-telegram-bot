package bot

import "sync"

// PendingKind distinguishes what the admin's next message means.
type PendingKind int

const (
	// PendingItem is new text for Chapters[Chapter].Items[Item].
	PendingItem PendingKind = iota
	// PendingTitle is the new top-level rules title.
	PendingTitle
)

// Pending tracks one admin waiting to send new text.
type Pending struct {
	Kind    PendingKind
	Chapter int
	Item    int
	ChatID  int64
}

// Sessions is an in-memory registry of userID -> pending input.
type Sessions struct {
	mu      sync.Mutex
	pending map[int64]Pending
}

// NewSessions creates an empty session registry.
func NewSessions() *Sessions {
	return &Sessions{pending: make(map[int64]Pending)}
}

// Set records that userID must send new text next.
func (s *Sessions) Set(userID int64, p Pending) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.pending[userID] = p
}

// Has reports whether userID has a pending edit without consuming it.
func (s *Sessions) Has(userID int64) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, ok := s.pending[userID]
	return ok
}

// Take returns and clears the pending edit for userID.
func (s *Sessions) Take(userID int64) (Pending, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	p, ok := s.pending[userID]
	if ok {
		delete(s.pending, userID)
	}
	return p, ok
}

// Clear drops any pending edit for userID.
func (s *Sessions) Clear(userID int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.pending, userID)
}
