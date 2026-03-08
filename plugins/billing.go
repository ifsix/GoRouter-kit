package plugins

import (
	"context"
	"errors"
	"time"

	"github.com/bycmd/GoRouter-kit/client"
	"github.com/bycmd/GoRouter-kit/schema"
)

type BillingReport struct {
	Model           string
	Usage           schema.Usage
	Cost            float64
	HasPrice        bool
	DurationMs      int64
	TimestampUnixMs int64
}

type BillingReporter func(ctx context.Context, report BillingReport) error

type BillingPluginOptions struct {
	Reporter BillingReporter
	Async    bool
	OnError  func(error)
}

type BillingPlugin struct {
	opts BillingPluginOptions
}

func NewBillingPlugin(opts BillingPluginOptions) *BillingPlugin {
	return &BillingPlugin{opts: opts}
}

func (p *BillingPlugin) Init(c *client.Client) error {
	if p.opts.Reporter == nil {
		return errors.New("billing reporter is required")
	}

	c.Use(func(ctx context.Context, m *client.MiddlewareContext, next client.Next) error {
		err := next(ctx)
		if err != nil {
			return err
		}
		if m.Result == nil {
			return nil
		}

		resp := m.Result.Response
		model := resp.Model
		if model == "" {
			model = m.Request.Model
		}

		report := BillingReport{
			Model:           model,
			Usage:           resp.Usage,
			DurationMs:      m.Result.DurationMs,
			TimestampUnixMs: time.Now().UnixMilli(),
		}

		if model != "" {
			if price, ok := c.ModelPrice(model); ok {
				report.HasPrice = true
				report.Cost = (float64(resp.Usage.PromptTokens)/1000.0)*price.InputPer1K +
					(float64(resp.Usage.CompletionTokens)/1000.0)*price.OutputPer1K
			}
		}

		send := func(callCtx context.Context) {
			if callCtx == nil {
				callCtx = context.Background()
			}
			if reportErr := p.opts.Reporter(callCtx, report); reportErr != nil && p.opts.OnError != nil {
				p.opts.OnError(reportErr)
			}
		}

		if p.opts.Async {
			go send(context.Background())
			return nil
		}

		send(ctx)
		return nil
	})

	return nil
}

func (p *BillingPlugin) Destroy() error {
	return nil
}
