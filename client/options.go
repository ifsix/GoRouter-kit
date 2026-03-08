package client

import (
	"context"
	"regexp"
	"strings"

	"github.com/ifsix/GoRouter-kit/errs"
	"github.com/ifsix/GoRouter-kit/schema"
)

var historyKeySanitizer = regexp.MustCompile(`[:/\\?#%]`)

func (c *Client) Ask(ctx context.Context, opt schema.ChatOptions) (*schema.ChatResponse, error) {
	out, err := c.AskWithReport(ctx, opt)
	if err != nil {
		return nil, err
	}
	return &out.Response, nil
}

func (c *Client) AskWithReport(ctx context.Context, opt schema.ChatOptions) (*schema.ChatResult, error) {
	req, err := composeChatRequest(opt)
	if err != nil {
		return nil, err
	}
	return c.ChatWithReport(ctx, req)
}

func composeChatRequest(opt schema.ChatOptions) (schema.ChatRequest, error) {
	req := opt.Request

	if strings.TrimSpace(req.SessionID) == "" {
		req.SessionID = historyKey(opt.User, opt.Group)
	}

	if len(opt.CustomMessages) > 0 {
		req.Messages = cloneMessages(opt.CustomMessages)
		if prompt := strings.TrimSpace(opt.SystemPrompt); prompt != "" && !hasSystem(req.Messages) {
			req.Messages = append([]schema.Message{{Role: schema.RoleSystem, Content: opt.SystemPrompt}}, req.Messages...)
		}

		if strings.TrimSpace(opt.Request.SessionID) == "" {
			req.SessionID = ""
		}
		if len(req.Messages) == 0 {
			return schema.ChatRequest{}, errs.ErrBadRequest
		}
		return req, nil
	}

	if prompt := strings.TrimSpace(opt.SystemPrompt); prompt != "" && !hasSystem(req.Messages) {
		req.Messages = append([]schema.Message{{Role: schema.RoleSystem, Content: opt.SystemPrompt}}, req.Messages...)
	}
	if prompt := strings.TrimSpace(opt.Prompt); prompt != "" {
		req.Messages = append(req.Messages, schema.Message{Role: schema.RoleUser, Content: opt.Prompt})
	}

	if len(req.Messages) == 0 && strings.TrimSpace(req.SessionID) == "" {
		return schema.ChatRequest{}, errs.ErrBadRequest
	}

	return req, nil
}

func cloneMessages(in []schema.Message) []schema.Message {
	if len(in) == 0 {
		return nil
	}
	out := make([]schema.Message, len(in))
	copy(out, in)
	return out
}

func hasSystem(messages []schema.Message) bool {
	for _, item := range messages {
		if item.Role == schema.RoleSystem {
			return true
		}
	}
	return false
}

func historyKey(user, group string) string {
	safeUser := normalizeKeyPart(user)
	if safeUser == "" {
		return ""
	}

	key := "user:" + safeUser
	safeGroup := normalizeKeyPart(group)
	if safeGroup != "" {
		key = "group:" + safeGroup + "_" + key
	}

	return key
}

func normalizeKeyPart(v string) string {
	v = strings.TrimSpace(v)
	if v == "" {
		return ""
	}
	return historyKeySanitizer.ReplaceAllString(v, "_")
}
