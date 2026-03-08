package plugins

import (
	"context"
	"sync/atomic"

	"github.com/bycmd/GoRouter-kit/client"
)

type Metrics struct {
	Requests       int64
	Succeeded      int64
	Failed         int64
	ToolCalls      int64
	LastDurationMs int64
}

type MetricsPlugin struct {
	requests       atomic.Int64
	succeeded      atomic.Int64
	failed         atomic.Int64
	toolCalls      atomic.Int64
	lastDurationMs atomic.Int64
}

func NewMetricsPlugin() *MetricsPlugin {
	return &MetricsPlugin{}
}

func (p *MetricsPlugin) Init(c *client.Client) error {
	c.Use(func(ctx context.Context, m *client.MiddlewareContext, next client.Next) error {
		p.requests.Add(1)

		err := next(ctx)
		if err != nil {
			p.failed.Add(1)
			return err
		}

		p.succeeded.Add(1)
		if m.Result != nil {
			p.lastDurationMs.Store(m.Result.DurationMs)
			p.toolCalls.Add(int64(len(m.Result.ToolCalls)))
		}
		return nil
	})
	return nil
}

func (p *MetricsPlugin) Destroy() error {
	return nil
}

func (p *MetricsPlugin) Snapshot() Metrics {
	return Metrics{
		Requests:       p.requests.Load(),
		Succeeded:      p.succeeded.Load(),
		Failed:         p.failed.Load(),
		ToolCalls:      p.toolCalls.Load(),
		LastDurationMs: p.lastDurationMs.Load(),
	}
}
