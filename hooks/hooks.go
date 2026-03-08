package hooks

import (
	"context"

	"github.com/bycmd/GoRouter-kit/schema"
)

type Hook interface {
	BeforeChat(ctx context.Context, req *schema.ChatRequest) error
	AfterChat(ctx context.Context, req *schema.ChatRequest, resp *schema.ChatResponse) error
	OnError(ctx context.Context, req *schema.ChatRequest, err error)
}

type Func struct {
	Before func(ctx context.Context, req *schema.ChatRequest) error
	After  func(ctx context.Context, req *schema.ChatRequest, resp *schema.ChatResponse) error
	Fail   func(ctx context.Context, req *schema.ChatRequest, err error)
}

func (h Func) BeforeChat(ctx context.Context, req *schema.ChatRequest) error {
	if h.Before == nil {
		return nil
	}
	return h.Before(ctx, req)
}

func (h Func) AfterChat(ctx context.Context, req *schema.ChatRequest, resp *schema.ChatResponse) error {
	if h.After == nil {
		return nil
	}
	return h.After(ctx, req, resp)
}

func (h Func) OnError(ctx context.Context, req *schema.ChatRequest, err error) {
	if h.Fail != nil {
		h.Fail(ctx, req, err)
	}
}
