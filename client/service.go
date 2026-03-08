package client

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/ifsix/GoRouter-kit/errs"
	"github.com/ifsix/GoRouter-kit/schema"
)

func (c *Client) SetModel(model string) error {
	model = strings.TrimSpace(model)
	if model == "" {
		return fmt.Errorf("model is required")
	}
	c.cfg.Model = model
	return nil
}

func (c *Client) Model() string {
	return c.cfg.Model
}

func (c *Client) SetAPIKey(apiKey string) error {
	apiKey = strings.TrimSpace(apiKey)
	if apiKey == "" {
		return fmt.Errorf("api key is required")
	}
	c.cfg.APIKey = apiKey
	return nil
}

func (c *Client) APIKey() string {
	return c.cfg.APIKey
}

func (c *Client) GetCreditBalance(ctx context.Context) (schema.CreditBalance, error) {
	var out schema.CreditBalance
	if err := c.getJSON(ctx, "/credits", &out); err != nil {
		return schema.CreditBalance{}, err
	}
	return out, nil
}

func (c *Client) GetAPIKeyInfo(ctx context.Context) (schema.APIKeyInfo, error) {
	var out schema.APIKeyInfo
	if err := c.getJSON(ctx, "/auth/key", &out); err != nil {
		return schema.APIKeyInfo{}, err
	}
	return out, nil
}

func (c *Client) GetModels(ctx context.Context) ([]schema.ModelInfo, error) {
	var out schema.ModelsEnvelope
	if err := c.getJSON(ctx, "/models", &out); err != nil {
		return nil, err
	}
	return out.Data, nil
}

func (c *Client) getJSON(ctx context.Context, path string, out any) error {
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, c.url(path), nil)
	if err != nil {
		return err
	}
	c.setHeaders(httpReq)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 64*1024))
		return &errs.APIError{
			StatusCode: resp.StatusCode,
			Message:    fmt.Sprintf("openrouter %d: %s", resp.StatusCode, strings.TrimSpace(string(body))),
		}
	}

	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf("%w: %v", errs.ErrDecode, err)
	}
	return nil
}

func (c *Client) url(path string) string {
	if strings.HasPrefix(path, "/") {
		return c.cfg.BaseURL + path
	}
	return c.cfg.BaseURL + "/" + path
}
