package security

import "sync"

type EventHandler func(payload any)

type eventBus struct {
	mu      sync.RWMutex
	seq     int
	byEvent map[string]map[int]EventHandler
}

func newEventBus() *eventBus {
	return &eventBus{
		byEvent: map[string]map[int]EventHandler{},
	}
}

func (b *eventBus) on(event string, fn EventHandler) int {
	if fn == nil || event == "" {
		return 0
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	b.seq++
	id := b.seq
	if b.byEvent[event] == nil {
		b.byEvent[event] = map[int]EventHandler{}
	}
	b.byEvent[event][id] = fn
	return id
}

func (b *eventBus) off(event string, id int) {
	if event == "" || id <= 0 {
		return
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	handlers := b.byEvent[event]
	if handlers == nil {
		return
	}
	delete(handlers, id)
	if len(handlers) == 0 {
		delete(b.byEvent, event)
	}
}

func (b *eventBus) emit(event string, payload any) {
	if event == "" {
		return
	}

	b.mu.RLock()
	handlers := b.byEvent[event]
	copied := make([]EventHandler, 0, len(handlers))
	for _, fn := range handlers {
		copied = append(copied, fn)
	}
	b.mu.RUnlock()

	for _, fn := range copied {
		fn(payload)
	}
}
