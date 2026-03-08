package client

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/bycmd/GoRouter-kit/errs"
	"github.com/bycmd/GoRouter-kit/schema"
)

func (c *Client) ChatStream(ctx context.Context, req schema.ChatRequest) (<-chan schema.StreamEvent, error) {
	if len(req.Messages) == 0 {
		return nil, errs.ErrBadRequest
	}
	if strings.TrimSpace(req.Model) == "" {
		req.Model = c.cfg.Model
	}
	req.Stream = true

	if err := c.applyHistoryLoad(ctx, &req); err != nil {
		return nil, err
	}
	if err := c.beforeHooks(ctx, &req); err != nil {
		return nil, err
	}

	payload, err := json.Marshal(req)
	if err != nil {
		c.failHooks(ctx, &req, err)
		return nil, err
	}

	resp, err := c.doRequest(ctx, payload)
	if err != nil {
		c.failHooks(ctx, &req, err)
		return nil, err
	}
	if resp.StatusCode >= 300 {
		defer resp.Body.Close()
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 64*1024))
		apiErr := &errs.APIError{StatusCode: resp.StatusCode, Message: fmt.Sprintf("openrouter %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))}
		c.failHooks(ctx, &req, apiErr)
		return nil, apiErr
	}

	out := make(chan schema.StreamEvent)
	go func() {
		defer close(out)
		defer resp.Body.Close()

		scanner := bufio.NewScanner(resp.Body)
		scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

		var (
			lastID    string
			lastModel string
			lastUsage schema.Usage
			hasUsage  bool
			content   strings.Builder
			toolCalls []schema.ToolCall
		)

		sendErr := func(err error) {
			c.failHooks(ctx, &req, err)
			out <- schema.StreamEvent{Err: err}
		}

		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line == "" || !strings.HasPrefix(line, "data:") {
				continue
			}

			raw := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
			if raw == "[DONE]" {
				break
			}

			var chunk schema.ChatStreamChunk
			if err := json.Unmarshal([]byte(raw), &chunk); err != nil {
				sendErr(err)
				return
			}

			if chunk.ID != "" {
				lastID = chunk.ID
			}
			if chunk.Model != "" {
				lastModel = chunk.Model
			}
			if chunk.Usage != nil {
				lastUsage = *chunk.Usage
				hasUsage = true
			}
			for _, choice := range chunk.Choices {
				if choice.Delta.Content != "" {
					content.WriteString(choice.Delta.Content)
				}
				if len(choice.ToolCalls) > 0 {
					toolCalls = append(toolCalls, choice.ToolCalls...)
				}
				if len(choice.Delta.ToolCalls) > 0 {
					toolCalls = append(toolCalls, choice.Delta.ToolCalls...)
				}
			}

			out <- schema.StreamEvent{Chunk: chunk}
		}

		if err := scanner.Err(); err != nil {
			sendErr(err)
			return
		}

		finalModel := lastModel
		if strings.TrimSpace(finalModel) == "" {
			finalModel = req.Model
		}

		msg := schema.Message{
			Role:    schema.RoleAssistant,
			Content: content.String(),
		}
		if len(toolCalls) > 0 {
			msg.ToolCalls = toolCalls
		}

		final := schema.ChatResponse{
			ID:    lastID,
			Model: finalModel,
		}
		if msg.Content != "" || len(msg.ToolCalls) > 0 {
			final.Choices = []schema.Choice{
				{
					Index:      0,
					Message:    msg,
					FinishType: "stop",
				},
			}
		}
		if hasUsage {
			final.Usage = lastUsage
			c.costs.Add(finalModel, lastUsage)
		}

		if err := c.applyHistorySave(ctx, req, &final); err != nil {
			sendErr(err)
			return
		}
		if err := c.afterHooks(ctx, &req, &final); err != nil {
			sendErr(err)
			return
		}

		out <- schema.StreamEvent{Done: true}
	}()

	return out, nil
}
