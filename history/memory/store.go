package memory

import (
	"context"
	"sort"
	"sync"

	"github.com/bycmd/GoRouter-kit/history"
	"github.com/bycmd/GoRouter-kit/schema"
)

type Store struct {
	mu   sync.RWMutex
	data map[string][]history.HistoryEntry
}

func New() *Store {
	return &Store{data: map[string][]history.HistoryEntry{}}
}

func (s *Store) Load(_ context.Context, sessionID string) ([]schema.Message, error) {
	entries, err := s.LoadEntries(context.Background(), sessionID)
	if err != nil {
		return nil, err
	}

	out := make([]schema.Message, 0, len(entries))
	for _, item := range entries {
		out = append(out, item.Message)
	}
	return out, nil
}

func (s *Store) Save(_ context.Context, sessionID string, messages []schema.Message) error {
	entries := make([]history.HistoryEntry, 0, len(messages))
	for _, item := range messages {
		entries = append(entries, history.HistoryEntry{Message: item})
	}
	return s.SaveEntries(context.Background(), sessionID, entries)
}

func (s *Store) Append(ctx context.Context, sessionID string, message schema.Message) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.data[sessionID] = append(s.data[sessionID], history.HistoryEntry{Message: message})
	return nil
}

func (s *Store) Clear(_ context.Context, sessionID string) error {
	return s.DeleteEntries(context.Background(), sessionID)
}

func (s *Store) LoadEntries(_ context.Context, sessionID string) ([]history.HistoryEntry, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	items := s.data[sessionID]
	out := make([]history.HistoryEntry, len(items))
	copy(out, items)
	return out, nil
}

func (s *Store) SaveEntries(_ context.Context, sessionID string, entries []history.HistoryEntry) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	out := make([]history.HistoryEntry, len(entries))
	copy(out, entries)
	s.data[sessionID] = out
	return nil
}

func (s *Store) DeleteEntries(_ context.Context, sessionID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.data, sessionID)
	return nil
}

func (s *Store) ListEntryKeys(_ context.Context) ([]string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	keys := make([]string, 0, len(s.data))
	for key := range s.data {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys, nil
}
