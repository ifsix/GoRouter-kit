package plugins

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/bycmd/GoRouter-kit/client"
)

type LoggingPlugin struct {
	logger *log.Logger
}

func NewLoggingPlugin(logger *log.Logger) *LoggingPlugin {
	if logger == nil {
		logger = log.New(os.Stdout, "[gorouter] ", log.LstdFlags)
	}
	return &LoggingPlugin{logger: logger}
}

func (p *LoggingPlugin) Init(c *client.Client) error {
	c.Use(func(ctx context.Context, m *client.MiddlewareContext, next client.Next) error {
		started := time.Now()
		err := next(ctx)
		dur := time.Since(started)

		if err != nil {
			p.logger.Printf("chat failed model=%s messages=%d duration=%s err=%v", m.Request.Model, len(m.Request.Messages), dur, err)
			return err
		}

		toolCalls := 0
		if m.Result != nil {
			toolCalls = len(m.Result.ToolCalls)
		}
		p.logger.Printf("chat ok model=%s messages=%d duration=%s tool_calls=%d", m.Request.Model, len(m.Request.Messages), dur, toolCalls)
		return nil
	})
	return nil
}

func (p *LoggingPlugin) Destroy() error {
	return nil
}
