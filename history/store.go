package history

import (
	"context"

	"github.com/bycmd/GoRouter-kit/schema"
)

type Store interface {
	Load(ctx context.Context, sessionID string) ([]schema.Message, error)
	Save(ctx context.Context, sessionID string, messages []schema.Message) error
	Append(ctx context.Context, sessionID string, message schema.Message) error
	Clear(ctx context.Context, sessionID string) error
}
