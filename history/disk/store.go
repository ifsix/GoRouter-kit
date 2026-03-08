package disk

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"

	"github.com/ifsix/GoRouter-kit/history"
	"github.com/ifsix/GoRouter-kit/schema"
)

type Store struct {
	dir string
}

var keySanitizer = regexp.MustCompile(`[^a-zA-Z0-9_.\-]`)

func New(dir string) *Store {
	if dir == "" {
		dir = ".gorouter-history"
	}
	return &Store{dir: dir}
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

func (s *Store) LoadEntries(_ context.Context, sessionID string) ([]history.HistoryEntry, error) {
	file := s.filePath(sessionID)
	data, err := os.ReadFile(file)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	if len(data) == 0 {
		return nil, nil
	}

	var entries []history.HistoryEntry
	if err := json.Unmarshal(data, &entries); err == nil {
		if len(entries) == 0 || entries[0].Message.Role != "" || entries[0].Message.Content != "" || entries[0].ApiCallMetadata != nil {
			return entries, nil
		}
	}

	var legacy []schema.Message
	if err := json.Unmarshal(data, &legacy); err != nil {
		return nil, err
	}
	out := make([]history.HistoryEntry, 0, len(legacy))
	for _, item := range legacy {
		out = append(out, history.HistoryEntry{Message: item})
	}
	return out, nil
}

func (s *Store) SaveEntries(_ context.Context, sessionID string, entries []history.HistoryEntry) error {
	if err := os.MkdirAll(s.dir, 0o755); err != nil {
		return err
	}
	file := s.filePath(sessionID)
	data, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(file, data, 0o644)
}

func (s *Store) Append(ctx context.Context, sessionID string, message schema.Message) error {
	items, err := s.LoadEntries(ctx, sessionID)
	if err != nil {
		return err
	}
	items = append(items, history.HistoryEntry{Message: message})
	return s.SaveEntries(ctx, sessionID, items)
}

func (s *Store) Clear(_ context.Context, sessionID string) error {
	return s.DeleteEntries(context.Background(), sessionID)
}

func (s *Store) DeleteEntries(_ context.Context, sessionID string) error {
	file := s.filePath(sessionID)
	err := os.Remove(file)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func (s *Store) ListKeys() ([]string, error) {
	return s.ListEntryKeys(context.Background())
}

func (s *Store) ListEntryKeys(_ context.Context) ([]string, error) {
	items, err := os.ReadDir(s.dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	out := make([]string, 0, len(items))
	for _, item := range items {
		if item.IsDir() {
			continue
		}
		name := item.Name()
		if filepath.Ext(name) != ".json" {
			continue
		}
		base := name[:len(name)-len(filepath.Ext(name))]
		out = append(out, base)
	}
	return out, nil
}

func (s *Store) filePath(sessionID string) string {
	safe := keySanitizer.ReplaceAllString(sessionID, "_")
	if safe == "" {
		safe = "default"
	}
	return filepath.Join(s.dir, safe+".json")
}
