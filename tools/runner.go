package tools

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/ifsix/GoRouter-kit/schema"
	"github.com/ifsix/GoRouter-kit/security"
)

type RunConfig struct {
	Parallel      bool
	IncludeResult bool
	Guard         security.Guard
	User          *security.User
}

func Run(ctx context.Context, calls []schema.ToolCall, available []schema.Tool, cfg RunConfig) ([]schema.ToolCallOutcome, error) {
	if len(calls) == 0 {
		return nil, nil
	}

	if cfg.Parallel {
		return runParallel(ctx, calls, available, cfg)
	}
	return runSequential(ctx, calls, available, cfg)
}

func runSequential(ctx context.Context, calls []schema.ToolCall, available []schema.Tool, cfg RunConfig) ([]schema.ToolCallOutcome, error) {
	out := make([]schema.ToolCallOutcome, 0, len(calls))
	for _, call := range calls {
		item := runOne(ctx, call, available, cfg)
		out = append(out, item)
	}
	return out, nil
}

func runParallel(ctx context.Context, calls []schema.ToolCall, available []schema.Tool, cfg RunConfig) ([]schema.ToolCallOutcome, error) {
	type row struct {
		idx int
		val schema.ToolCallOutcome
	}

	ch := make(chan row, len(calls))
	for i, call := range calls {
		i, call := i, call
		go func() {
			ch <- row{idx: i, val: runOne(ctx, call, available, cfg)}
		}()
	}

	out := make([]schema.ToolCallOutcome, len(calls))
	for range calls {
		item := <-ch
		out[item.idx] = item.val
	}
	return out, nil
}

func runOne(ctx context.Context, call schema.ToolCall, available []schema.Tool, cfg RunConfig) schema.ToolCallOutcome {
	started := time.Now()

	toolName := call.Function.Name
	if toolName == "" {
		out := errorOutcome(call, "unknown_tool", schema.ToolCallErrorUnknown, errors.New("tool name is missing"), started, nil)
		auditCall(cfg, "unknown_tool", call, nil, nil, out.Detail.Error)
		return out
	}

	tool, ok := findTool(available, toolName)
	if !ok {
		out := errorOutcome(call, toolName, schema.ToolCallErrorNotFound, fmt.Errorf("tool %q not found", toolName), started, nil)
		auditCall(cfg, toolName, call, nil, nil, out.Detail.Error)
		return out
	}
	if tool.Execute == nil {
		out := errorOutcome(call, toolName, schema.ToolCallErrorNotFound, fmt.Errorf("tool %q has no executor", toolName), started, nil)
		auditCall(cfg, toolName, call, nil, nil, out.Detail.Error)
		return out
	}

	var args any
	if call.Function.Arguments != "" {
		if err := json.Unmarshal([]byte(call.Function.Arguments), &args); err != nil {
			out := errorOutcome(call, toolName, schema.ToolCallErrorParsing, err, started, nil)
			auditCall(cfg, toolName, call, nil, nil, out.Detail.Error)
			return out
		}
	}

	if cfg.Guard != nil {
		if err := cfg.Guard.Check(ctx, security.CheckInput{
			Tool: tool,
			Call: call,
			Args: args,
			User: cfg.User,
		}); err != nil {
			out := errorOutcome(call, toolName, schema.ToolCallErrorSecurity, err, started, args)
			auditCall(cfg, toolName, call, args, nil, out.Detail.Error)
			return out
		}
	}

	res, err := tool.Execute(ctx, args, call)
	if err != nil {
		out := errorOutcome(call, toolName, schema.ToolCallErrorExecution, err, started, args)
		auditCall(cfg, toolName, call, args, nil, out.Detail.Error)
		return out
	}

	raw, err := json.Marshal(res)
	if err != nil {
		out := errorOutcome(call, toolName, schema.ToolCallErrorExecution, err, started, args)
		auditCall(cfg, toolName, call, args, nil, out.Detail.Error)
		return out
	}

	detail := schema.ToolCallDetail{
		ToolCallID:     call.ID,
		ToolName:       toolName,
		RequestArgsRaw: call.Function.Arguments,
		ParsedArgs:     args,
		Status:         schema.ToolCallSuccess,
		ResultString:   string(raw),
		DurationMs:     time.Since(started).Milliseconds(),
	}
	if cfg.IncludeResult {
		detail.Result = res
	}

	msg := schema.Message{
		Role:     schema.RoleTool,
		Name:     toolName,
		ToolCall: call.ID,
		Content:  string(raw),
	}
	out := schema.ToolCallOutcome{
		Message: msg,
		Detail:  detail,
	}
	auditCall(cfg, toolName, call, args, res, nil)
	return out
}

func errorOutcome(call schema.ToolCall, toolName string, status schema.ToolCallStatus, err error, started time.Time, args any) schema.ToolCallOutcome {
	payload := map[string]any{
		"errorType":    string(status),
		"errorMessage": err.Error(),
	}
	raw, _ := json.Marshal(payload)

	return schema.ToolCallOutcome{
		Message: schema.Message{
			Role:     schema.RoleTool,
			Name:     toolName,
			ToolCall: call.ID,
			Content:  string(raw),
		},
		Detail: schema.ToolCallDetail{
			ToolCallID:     call.ID,
			ToolName:       toolName,
			RequestArgsRaw: call.Function.Arguments,
			ParsedArgs:     args,
			Status:         status,
			Error: &schema.ToolCallError{
				Type:    string(status),
				Message: err.Error(),
			},
			ResultString: string(raw),
			DurationMs:   time.Since(started).Milliseconds(),
		},
	}
}

func findTool(items []schema.Tool, name string) (schema.Tool, bool) {
	for _, item := range items {
		if item.Function.Name == name || item.Name == name {
			return item, true
		}
	}
	return schema.Tool{}, false
}

func auditCall(cfg RunConfig, toolName string, call schema.ToolCall, args any, result any, detailErr *schema.ToolCallError) {
	auditor, ok := cfg.Guard.(security.ToolCallAuditor)
	if !ok {
		return
	}
	userID := ""
	if cfg.User != nil {
		userID = cfg.User.ID
	}

	var errObj error
	if detailErr != nil {
		errObj = errors.New(detailErr.Message)
	}

	auditor.LogToolCall(security.ToolCallEvent{
		ToolName: toolName,
		UserID:   userID,
		Args:     args,
		Result:   result,
		Success:  detailErr == nil,
		Error:    errObj,
		TimeUnix: time.Now().Unix(),
	})
}
