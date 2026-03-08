package redis

import (
	"context"
	"encoding/json"
	"errors"
	"regexp"
	"sort"
	"strings"
	"sync"

	"github.com/ifsix/GoRouter-kit/history"
	"github.com/ifsix/GoRouter-kit/schema"
)

type Client interface {
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key, value string) error
	Del(ctx context.Context, key string) error
	Keys(ctx context.Context, pattern string) ([]string, error)
}

type Options struct {
	Prefix  string
	CloseFn func() error
}

type Store struct {
	client  Client
	prefix  string
	closeFn func() error
	once    sync.Once
}

var keyFilter = regexp.MustCompile(`[^a-zA-Z0-9_.\-:]`)
var errNilClient = errors.New("redis history client is nil")

func New(client Client, opts Options) *Store {
	prefix := strings.TrimSpace(opts.Prefix)
	if prefix == "" {
		prefix = "chat_history:"
	}

	return &Store{
		client:  client,
		prefix:  prefix,
		closeFn: opts.CloseFn,
	}
}

func (s *Store) Load(ctx context.Context, sessionID string) ([]schema.Message, error) {
	entries, err := s.LoadEntries(ctx, sessionID)
	if err != nil {
		return nil, err
	}

	out := make([]schema.Message, 0, len(entries))
	for _, item := range entries {
		out = append(out, item.Message)
	}
	return out, nil
}

func (s *Store) Save(ctx context.Context, sessionID string, messages []schema.Message) error {
	entries := make([]history.HistoryEntry, 0, len(messages))
	for _, item := range messages {
		entries = append(entries, history.HistoryEntry{Message: item})
	}
	return s.SaveEntries(ctx, sessionID, entries)
}

func (s *Store) Append(ctx context.Context, sessionID string, message schema.Message) error {
	items, err := s.LoadEntries(ctx, sessionID)
	if err != nil {
		return err
	}

	items = append(items, history.HistoryEntry{Message: message})
	return s.SaveEntries(ctx, sessionID, items)
}

func (s *Store) Clear(ctx context.Context, sessionID string) error {
	return s.DeleteEntries(ctx, sessionID)
}

func (s *Store) LoadEntries(ctx context.Context, sessionID string) ([]history.HistoryEntry, error) {
	if s.client == nil {
		return nil, errNilClient
	}

	raw, err := s.client.Get(ctx, s.key(sessionID))
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(raw) == "" {
		return nil, nil
	}

	var entries []history.HistoryEntry
	if err := json.Unmarshal([]byte(raw), &entries); err == nil {
		if len(entries) == 0 || entries[0].Message.Role != "" || entries[0].Message.Content != "" || entries[0].ApiCallMetadata != nil {
			return entries, nil
		}
	}

	var legacy []schema.Message
	if err := json.Unmarshal([]byte(raw), &legacy); err != nil {
		return nil, err
	}

	out := make([]history.HistoryEntry, 0, len(legacy))
	for _, item := range legacy {
		out = append(out, history.HistoryEntry{Message: item})
	}
	return out, nil
}

func (s *Store) SaveEntries(ctx context.Context, sessionID string, entries []history.HistoryEntry) error {
	if s.client == nil {
		return errNilClient
	}

	payload, err := json.Marshal(entries)
	if err != nil {
		return err
	}
	return s.client.Set(ctx, s.key(sessionID), string(payload))
}

func (s *Store) DeleteEntries(ctx context.Context, sessionID string) error {
	if s.client == nil {
		return errNilClient
	}
	return s.client.Del(ctx, s.key(sessionID))
}

func (s *Store) ListEntryKeys(ctx context.Context) ([]string, error) {
	if s.client == nil {
		return nil, errNilClient
	}

	keys, err := s.client.Keys(ctx, s.prefix+"*")
	if err != nil {
		return nil, err
	}

	out := make([]string, 0, len(keys))
	for _, item := range keys {
		if strings.HasPrefix(item, s.prefix) {
			out = append(out, item[len(s.prefix):])
		}
	}
	sort.Strings(out)
	return out, nil
}

func (s *Store) Destroy() {
	if s.closeFn == nil {
		return
	}
	s.once.Do(func() {
		_ = s.closeFn()
	})
}

func (s *Store) key(sessionID string) string {
	safe := keyFilter.ReplaceAllString(sessionID, "_")
	if safe == "" {
		safe = "default"
	}
	return s.prefix + safe
}
