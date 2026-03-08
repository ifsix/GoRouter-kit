package security

import (
	"fmt"
	"sync"
	"time"
)

type limitState struct {
	Count int
	Reset time.Time
}

type Limiter struct {
	mu   sync.Mutex
	data map[string]limitState
}

func NewLimiter() *Limiter {
	return &Limiter{data: map[string]limitState{}}
}

func (l *Limiter) Allow(key string, limit int, window time.Duration) (bool, time.Duration) {
	if limit <= 0 {
		return false, window
	}
	if window <= 0 {
		window = time.Minute
	}

	now := time.Now()

	l.mu.Lock()
	defer l.mu.Unlock()

	st, ok := l.data[key]
	if !ok || now.After(st.Reset) {
		l.data[key] = limitState{Count: 1, Reset: now.Add(window)}
		return true, 0
	}

	if st.Count >= limit {
		return false, st.Reset.Sub(now)
	}

	st.Count++
	l.data[key] = st
	return true, 0
}

func (l *Limiter) key(userID, tool string) string {
	if userID == "" {
		userID = "anonymous"
	}
	return fmt.Sprintf("%s|%s", userID, tool)
}

func (l *Limiter) Reset() {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.data = map[string]limitState{}
}

func (l *Limiter) ResetUser(userID string) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if userID == "" {
		return
	}
	prefix := userID + "|"
	for key := range l.data {
		if len(key) >= len(prefix) && key[:len(prefix)] == prefix {
			delete(l.data, key)
		}
	}
}
