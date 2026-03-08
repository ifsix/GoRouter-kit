package client

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/bycmd/GoRouter-kit/config"
	"github.com/bycmd/GoRouter-kit/cost"
	"github.com/bycmd/GoRouter-kit/errs"
	"github.com/bycmd/GoRouter-kit/history"
	"github.com/bycmd/GoRouter-kit/hooks"
	"github.com/bycmd/GoRouter-kit/schema"
	"github.com/bycmd/GoRouter-kit/security"
	"github.com/bycmd/GoRouter-kit/tools"
)

type Client struct {
	cfg         config.Config
	httpClient  *http.Client
	hooks       []hooks.Hook
	costs       *cost.Tracker
	middlewares []Middleware
	plugins     []Plugin
	events      *eventBus

	priceRefreshMu     sync.Mutex
	priceRefreshCancel context.CancelFunc
}

func New(cfg config.Config) (*Client, error) {
	cfg = cfg.Prepare()
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	cli := &Client{
		cfg: cfg,
		httpClient: &http.Client{
			Timeout: cfg.Timeout,
		},
		costs:  cost.NewTracker(cfg.ModelPrices),
		events: newEventBus(),
	}
	if cfg.RefreshPricesOnUp {
		_ = cli.RefreshModelPrices(context.Background())
	}
	if cfg.PriceRefreshEvery > 0 {
		cli.StartPriceAutoRefresh(context.Background(), cfg.PriceRefreshEvery)
	}
	return cli, nil
}

func (c *Client) SetCostTracker(tracker *cost.Tracker) {
	if tracker == nil {
		c.costs = cost.NewTracker(nil)
		return
	}
	c.costs = tracker
}

func (c *Client) Close() error {
	return c.Destroy()
}

func (c *Client) CostSnapshot() cost.Snapshot {
	return c.costs.Snapshot()
}

func (c *Client) AddHook(h hooks.Hook) {
	if h == nil {
		return
	}
	c.hooks = append(c.hooks, h)
}

func (c *Client) Use(m Middleware) {
	if m == nil {
		return
	}
	c.middlewares = append(c.middlewares, m)
}

func (c *Client) RegisterPlugin(plugin Plugin) error {
	if plugin == nil {
		return nil
	}
	if err := plugin.Init(c); err != nil {
		return err
	}
	c.plugins = append(c.plugins, plugin)
	return nil
}

func (c *Client) DestroyPlugins() error {
	var first error
	for i := len(c.plugins) - 1; i >= 0; i-- {
		if err := c.plugins[i].Destroy(); err != nil && first == nil {
			first = err
		}
	}
	c.plugins = nil
	return first
}

func (c *Client) Chat(ctx context.Context, req schema.ChatRequest) (*schema.ChatResponse, error) {
	out, err := c.ChatWithReport(ctx, req)
	if err != nil {
		return nil, err
	}
	return &out.Response, nil
}

func (c *Client) ChatWithReport(ctx context.Context, req schema.ChatRequest) (*schema.ChatResult, error) {
	ctxState := &MiddlewareContext{
		Request: req,
		Meta:    map[string]any{},
	}

	if err := c.runMiddlewares(ctx, ctxState, func(ctx context.Context) error {
		res, err := c.chatFlow(ctx, ctxState.Request)
		if err != nil {
			return err
		}
		ctxState.Result = res
		return nil
	}); err != nil {
		c.events.emit("error", err)
		return nil, err
	}

	if ctxState.Result == nil {
		return nil, errorsf("chat result is empty")
	}

	return ctxState.Result, nil
}

func (c *Client) chatFlow(ctx context.Context, req schema.ChatRequest) (*schema.ChatResult, error) {
	started := time.Now()
	if strings.TrimSpace(req.Model) == "" {
		req.Model = c.cfg.Model
	}
	if req.MaxToolCalls <= 0 {
		req.MaxToolCalls = c.cfg.MaxToolCalls
	}

	if err := c.applyHistoryLoad(ctx, &req); err != nil {
		return nil, err
	}
	if len(req.Messages) == 0 {
		return nil, errs.ErrBadRequest
	}
	if err := c.beforeHooks(ctx, &req); err != nil {
		return nil, err
	}

	toolDetails := make([]schema.ToolCallDetail, 0)
	var out schema.ChatResponse
	var runUser *security.User

	if req.AccessToken != "" {
		if auth, ok := c.cfg.Security.(security.Authenticator); ok {
			user, err := auth.Authenticate(ctx, req.AccessToken)
			if err != nil {
				c.failHooks(ctx, &req, err)
				return nil, err
			}
			runUser = user
		}
	}

	for step := 0; step <= req.MaxToolCalls; step++ {
		resp, err := c.doChatOnce(ctx, req)
		if err != nil {
			c.failHooks(ctx, &req, err)
			return nil, err
		}
		out = resp

		calls := toolCallsFrom(resp)
		if len(calls) == 0 || len(req.Tools) == 0 {
			break
		}
		if step == req.MaxToolCalls {
			break
		}

		parallel := req.ParallelToolCalls || c.cfg.ParallelToolCalls
		toolOutcomes, err := tools.Run(ctx, calls, req.Tools, tools.RunConfig{
			Parallel:      parallel,
			IncludeResult: req.IncludeToolResultInReport,
			Guard:         c.cfg.Security,
			User:          runUser,
		})
		if err != nil {
			c.failHooks(ctx, &req, err)
			return nil, err
		}

		for _, item := range toolOutcomes {
			toolDetails = append(toolDetails, item.Detail)
		}

		if len(resp.Choices) > 0 {
			req.Messages = append(req.Messages, resp.Choices[0].Message)
		}
		for _, item := range toolOutcomes {
			req.Messages = append(req.Messages, item.Message)
		}
	}

	if err := c.applyHistorySave(ctx, req, &out); err != nil {
		c.failHooks(ctx, &req, err)
		return nil, err
	}
	if err := c.afterHooks(ctx, &req, &out); err != nil {
		c.failHooks(ctx, &req, err)
		return nil, err
	}

	return &schema.ChatResult{
		Response:   out,
		ToolCalls:  toolDetails,
		DurationMs: time.Since(started).Milliseconds(),
	}, nil
}

func (c *Client) doChatOnce(ctx context.Context, req schema.ChatRequest) (schema.ChatResponse, error) {
	payload, err := json.Marshal(req)
	if err != nil {
		return schema.ChatResponse{}, err
	}

	resp, err := c.doRequest(ctx, payload)
	if err != nil {
		return schema.ChatResponse{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 64*1024))
		return schema.ChatResponse{}, &errs.APIError{StatusCode: resp.StatusCode, Message: fmt.Sprintf("openrouter %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))}
	}

	var out schema.ChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return schema.ChatResponse{}, fmt.Errorf("%w: %v", errs.ErrDecode, err)
	}

	c.costs.Add(out.Model, out.Usage)
	return out, nil
}

func (c *Client) setHeaders(req *http.Request) {
	req.Header.Set("Authorization", "Bearer "+c.cfg.APIKey)
	req.Header.Set("Content-Type", "application/json")
	if strings.TrimSpace(c.cfg.HTTPReferer) != "" {
		req.Header.Set("HTTP-Referer", c.cfg.HTTPReferer)
	}
	if strings.TrimSpace(c.cfg.AppName) != "" {
		req.Header.Set("X-Title", c.cfg.AppName)
	}
}

func (c *Client) beforeHooks(ctx context.Context, req *schema.ChatRequest) error {
	for _, hook := range c.hooks {
		if err := hook.BeforeChat(ctx, req); err != nil {
			return err
		}
	}
	return nil
}

func (c *Client) afterHooks(ctx context.Context, req *schema.ChatRequest, resp *schema.ChatResponse) error {
	for _, hook := range c.hooks {
		if err := hook.AfterChat(ctx, req, resp); err != nil {
			return err
		}
	}
	return nil
}

func (c *Client) failHooks(ctx context.Context, req *schema.ChatRequest, err error) {
	for _, hook := range c.hooks {
		hook.OnError(ctx, req, err)
	}
	c.events.emit("error", err)
}

func (c *Client) applyHistoryLoad(ctx context.Context, req *schema.ChatRequest) error {
	if c.cfg.History == nil || strings.TrimSpace(req.SessionID) == "" {
		return nil
	}

	if entryStore, ok := c.cfg.History.(history.EntryStore); ok {
		entries, err := entryStore.LoadEntries(ctx, req.SessionID)
		if err != nil {
			return err
		}
		if len(entries) == 0 {
			return nil
		}

		merged := make([]schema.Message, 0, len(entries)+len(req.Messages))
		for _, item := range entries {
			merged = append(merged, item.Message)
		}
		merged = append(merged, req.Messages...)
		req.Messages = merged
		return nil
	}

	items, err := c.cfg.History.Load(ctx, req.SessionID)
	if err != nil {
		return err
	}
	if len(items) == 0 {
		return nil
	}

	merged := make([]schema.Message, 0, len(items)+len(req.Messages))
	merged = append(merged, items...)
	merged = append(merged, req.Messages...)
	req.Messages = merged
	return nil
}

func (c *Client) applyHistorySave(ctx context.Context, req schema.ChatRequest, resp *schema.ChatResponse) error {
	if c.cfg.History == nil || strings.TrimSpace(req.SessionID) == "" {
		return nil
	}

	if entryStore, ok := c.cfg.History.(history.EntryStore); ok {
		oldEntries, err := entryStore.LoadEntries(ctx, req.SessionID)
		if err != nil {
			return err
		}

		start := len(oldEntries)
		if start > len(req.Messages) {
			start = len(req.Messages)
		}

		newEntries := make([]history.HistoryEntry, 0, len(req.Messages)-start+1)
		for i := start; i < len(req.Messages); i++ {
			newEntries = append(newEntries, history.HistoryEntry{
				Message: req.Messages[i],
			})
		}

		if len(resp.Choices) > 0 {
			finish := resp.Choices[0].FinishType
			usage := resp.Usage
			meta := &history.ApiCallMetadata{
				CallID:               resp.ID,
				ModelUsed:            resp.Model,
				Usage:                &usage,
				Timestamp:            time.Now().UnixMilli(),
				FinishReason:         finish,
				RequestMessagesCount: len(req.Messages),
			}
			newEntries = append(newEntries, history.HistoryEntry{
				Message:         resp.Choices[0].Message,
				ApiCallMetadata: meta,
			})
		}

		if len(newEntries) == 0 {
			return nil
		}

		merged := make([]history.HistoryEntry, 0, len(oldEntries)+len(newEntries))
		merged = append(merged, oldEntries...)
		merged = append(merged, newEntries...)
		return entryStore.SaveEntries(ctx, req.SessionID, merged)
	}

	merged := append([]schema.Message{}, req.Messages...)
	if len(resp.Choices) > 0 {
		merged = append(merged, resp.Choices[0].Message)
	}

	return c.cfg.History.Save(ctx, req.SessionID, merged)
}

func toolCallsFrom(resp schema.ChatResponse) []schema.ToolCall {
	if len(resp.Choices) == 0 {
		return nil
	}
	return resp.Choices[0].Message.ToolCalls
}

func errorsf(format string, args ...any) error {
	return fmt.Errorf(format, args...)
}
