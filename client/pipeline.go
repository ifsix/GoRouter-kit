package client

import (
	"context"

	"github.com/bycmd/GoRouter-kit/schema"
)

type Plugin interface {
	Init(client *Client) error
	Destroy() error
}

type MiddlewareContext struct {
	Request schema.ChatRequest
	Result  *schema.ChatResult
	Meta    map[string]any
}

type Next func(ctx context.Context) error

type Middleware func(ctx context.Context, m *MiddlewareContext, next Next) error

func (c *Client) runMiddlewares(ctx context.Context, m *MiddlewareContext, core func(ctx context.Context) error) error {
	if len(c.middlewares) == 0 {
		return core(ctx)
	}

	var walk func(i int, ctx context.Context) error
	walk = func(i int, ctx context.Context) error {
		if i >= len(c.middlewares) {
			return core(ctx)
		}
		return c.middlewares[i](ctx, m, func(ctx context.Context) error {
			return walk(i+1, ctx)
		})
	}

	return walk(0, ctx)
}
