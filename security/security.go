package security

import (
	"context"

	"github.com/ifsix/GoRouter-kit/schema"
)

type User struct {
	ID        string
	Role      string
	Roles     []string
	Scopes    []string
	Username  string
	ExpiresAt int64
	Metadata  map[string]any
}

type CheckInput struct {
	Tool schema.Tool
	Call schema.ToolCall
	Args any
	User *User
}

type Guard interface {
	Check(ctx context.Context, input CheckInput) error
}

type Authenticator interface {
	Authenticate(ctx context.Context, token string) (*User, error)
}

type ToolCallEvent struct {
	ToolName string
	UserID   string
	Args     any
	Result   any
	Success  bool
	Error    error
	TimeUnix int64
}

type ToolCallAuditor interface {
	LogToolCall(event ToolCallEvent)
}
