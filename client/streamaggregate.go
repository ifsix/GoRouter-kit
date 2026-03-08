package client

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/bycmd/GoRouter-kit/schema"
	"github.com/bycmd/GoRouter-kit/security"
	"github.com/bycmd/GoRouter-kit/tools"
)

func (c *Client) ChatStreamResult(ctx context.Context, req schema.ChatRequest, cb *schema.StreamCallbacks) (*schema.ChatStreamResult, error) {
	started := time.Now()

	if req.MaxToolCalls <= 0 {
		req.MaxToolCalls = c.cfg.MaxToolCalls
	}

	runUser, err := c.resolveRunUser(ctx, req.AccessToken)
	if err != nil {
		if cb != nil && cb.OnError != nil {
			cb.OnError(err)
		}
		return nil, err
	}

	var result schema.ChatStreamResult
	for step := 0; ; step++ {
		item, err := c.streamOnce(ctx, req, cb)
		if err != nil {
			if cb != nil && cb.OnError != nil {
				cb.OnError(err)
			}
			return nil, err
		}
		result = item

		if len(item.ToolCalls) == 0 || item.FinishReason != "tool_calls" || len(req.Tools) == 0 || step >= req.MaxToolCalls {
			break
		}

		for _, call := range item.ToolCalls {
			if cb != nil && cb.OnToolCallExecuting != nil {
				var args any
				_ = json.Unmarshal([]byte(call.Function.Arguments), &args)
				cb.OnToolCallExecuting(call.Function.Name, args)
			}
		}

		parallel := req.ParallelToolCalls || c.cfg.ParallelToolCalls
		toolOutcomes, err := tools.Run(ctx, item.ToolCalls, req.Tools, tools.RunConfig{
			Parallel:      parallel,
			IncludeResult: req.IncludeToolResultInReport,
			Guard:         c.cfg.Security,
			User:          runUser,
		})
		if err != nil {
			if cb != nil && cb.OnError != nil {
				cb.OnError(err)
			}
			return nil, err
		}

		req.Messages = append(req.Messages, schema.Message{
			Role:      schema.RoleAssistant,
			Content:   item.Content,
			ToolCalls: item.ToolCalls,
		})
		for _, out := range toolOutcomes {
			req.Messages = append(req.Messages, out.Message)
			if cb != nil && cb.OnToolCallResult != nil {
				value := out.Detail.Result
				if value == nil {
					value = out.Message.Content
				}
				cb.OnToolCallResult(out.Detail.ToolName, value)
			}
		}
	}

	result.DurationMs = time.Since(started).Milliseconds()
	if cb != nil && cb.OnComplete != nil {
		cb.OnComplete(result)
	}

	return &result, nil
}

func (c *Client) streamOnce(ctx context.Context, req schema.ChatRequest, cb *schema.StreamCallbacks) (schema.ChatStreamResult, error) {
	stream, err := c.ChatStream(ctx, req)
	if err != nil {
		return schema.ChatStreamResult{}, err
	}

	var (
		content strings.Builder
		result  schema.ChatStreamResult
	)

	for item := range stream {
		if item.Err != nil {
			return schema.ChatStreamResult{}, item.Err
		}

		if item.Done {
			break
		}

		chunk := item.Chunk
		if cb != nil && cb.OnChunk != nil {
			cb.OnChunk(chunk)
		}

		if chunk.ID != "" {
			result.ID = chunk.ID
		}
		if chunk.Model != "" {
			result.Model = chunk.Model
		}
		if chunk.Usage != nil {
			usage := *chunk.Usage
			result.Usage = &usage
		}

		for _, choice := range chunk.Choices {
			if choice.Delta.Content != "" {
				content.WriteString(choice.Delta.Content)
				if cb != nil && cb.OnContent != nil {
					cb.OnContent(choice.Delta.Content)
				}
			}
			if choice.FinishType != "" {
				result.FinishReason = choice.FinishType
			}
			if len(choice.ToolCalls) > 0 {
				result.ToolCalls = append(result.ToolCalls, choice.ToolCalls...)
			}
			if len(choice.Delta.ToolCalls) > 0 {
				result.ToolCalls = append(result.ToolCalls, choice.Delta.ToolCalls...)
			}
		}
	}

	result.Content = content.String()
	if result.FinishReason == "" {
		result.FinishReason = "stop"
	}

	return result, nil
}

func (c *Client) resolveRunUser(ctx context.Context, accessToken string) (*security.User, error) {
	if accessToken == "" {
		return nil, nil
	}
	auth, ok := c.cfg.Security.(security.Authenticator)
	if !ok {
		return nil, nil
	}
	return auth.Authenticate(ctx, accessToken)
}
