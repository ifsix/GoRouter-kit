package history

import (
	"context"
	"sync"
	"time"

	"github.com/ifsix/GoRouter-kit/schema"
)

type ManagerOptions struct {
	TTL             time.Duration
	CleanupInterval time.Duration
}

type cachedHistory struct {
	entries    []HistoryEntry
	lastAccess time.Time
	created    time.Time
}

type Manager struct {
	store EntryStore
	opts  ManagerOptions

	mu    sync.RWMutex
	cache map[string]cachedHistory

	stopOnce sync.Once
	stopCh   chan struct{}
}

func NewManager(store EntryStore, opts ManagerOptions) *Manager {
	m := &Manager{
		store:  store,
		opts:   opts,
		cache:  map[string]cachedHistory{},
		stopCh: make(chan struct{}),
	}
	m.startCleanup()
	return m
}

func (m *Manager) GetHistoryEntries(ctx context.Context, key string) ([]HistoryEntry, error) {
	if key == "" {
		return nil, nil
	}

	now := time.Now()
	if entries, ok := m.getCached(key, now); ok {
		return entries, nil
	}

	entries, err := m.store.LoadEntries(ctx, key)
	if err != nil {
		return nil, err
	}

	m.mu.Lock()
	m.cache[key] = cachedHistory{
		entries:    copyEntries(entries),
		lastAccess: now,
		created:    now,
	}
	m.mu.Unlock()

	return copyEntries(entries), nil
}

func (m *Manager) GetHistoryMessages(ctx context.Context, key string) ([]schema.Message, error) {
	entries, err := m.GetHistoryEntries(ctx, key)
	if err != nil {
		return nil, err
	}
	out := make([]schema.Message, 0, len(entries))
	for _, item := range entries {
		out = append(out, item.Message)
	}
	return out, nil
}

func (m *Manager) AddHistoryEntries(ctx context.Context, key string, newEntries []HistoryEntry) error {
	if key == "" || len(newEntries) == 0 {
		return nil
	}

	current, err := m.GetHistoryEntries(ctx, key)
	if err != nil {
		return err
	}

	merged := make([]HistoryEntry, 0, len(current)+len(newEntries))
	merged = append(merged, current...)
	merged = append(merged, copyEntries(newEntries)...)

	if err := m.store.SaveEntries(ctx, key, merged); err != nil {
		return err
	}

	now := time.Now()
	m.mu.Lock()
	m.cache[key] = cachedHistory{
		entries:    copyEntries(merged),
		lastAccess: now,
		created:    now,
	}
	m.mu.Unlock()

	return nil
}

func (m *Manager) ClearHistory(ctx context.Context, key string) error {
	if key == "" {
		return nil
	}

	if err := m.store.SaveEntries(ctx, key, nil); err != nil {
		return err
	}

	now := time.Now()
	m.mu.Lock()
	m.cache[key] = cachedHistory{
		entries:    nil,
		lastAccess: now,
		created:    now,
	}
	m.mu.Unlock()
	return nil
}

func (m *Manager) DeleteHistory(ctx context.Context, key string) error {
	if key == "" {
		return nil
	}
	if err := m.store.DeleteEntries(ctx, key); err != nil {
		return err
	}
	m.mu.Lock()
	delete(m.cache, key)
	m.mu.Unlock()
	return nil
}

func (m *Manager) GetAllHistoryKeys(ctx context.Context) ([]string, error) {
	return m.store.ListEntryKeys(ctx)
}

func (m *Manager) Destroy() {
	m.stopOnce.Do(func() {
		close(m.stopCh)
		m.mu.Lock()
		m.cache = map[string]cachedHistory{}
		m.mu.Unlock()
	})
}

func (m *Manager) getCached(key string, now time.Time) ([]HistoryEntry, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	item, ok := m.cache[key]
	if !ok {
		return nil, false
	}

	if m.opts.TTL > 0 && now.Sub(item.lastAccess) > m.opts.TTL {
		delete(m.cache, key)
		return nil, false
	}

	item.lastAccess = now
	m.cache[key] = item
	return copyEntries(item.entries), true
}

func (m *Manager) startCleanup() {
	if m.opts.TTL <= 0 || m.opts.CleanupInterval <= 0 {
		return
	}

	go func() {
		ticker := time.NewTicker(m.opts.CleanupInterval)
		defer ticker.Stop()

		for {
			select {
			case <-m.stopCh:
				return
			case now := <-ticker.C:
				m.cleanupExpired(now)
			}
		}
	}()
}

func (m *Manager) cleanupExpired(now time.Time) {
	m.mu.Lock()
	defer m.mu.Unlock()

	for key, item := range m.cache {
		if now.Sub(item.lastAccess) > m.opts.TTL {
			delete(m.cache, key)
		}
	}
}

func copyEntries(in []HistoryEntry) []HistoryEntry {
	if len(in) == 0 {
		return nil
	}

	out := make([]HistoryEntry, len(in))
	for i, item := range in {
		out[i] = item
		if item.ApiCallMetadata != nil {
			meta := *item.ApiCallMetadata
			if item.ApiCallMetadata.Usage != nil {
				usage := *item.ApiCallMetadata.Usage
				meta.Usage = &usage
			}
			if item.ApiCallMetadata.Cost != nil {
				cost := *item.ApiCallMetadata.Cost
				meta.Cost = &cost
			}
			out[i].ApiCallMetadata = &meta
		}
	}
	return out
}
