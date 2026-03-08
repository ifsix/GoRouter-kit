package config

import (
	"errors"
	"strings"
	"time"

	"github.com/bycmd/GoRouter-kit/cost"
	"github.com/bycmd/GoRouter-kit/history"
	"github.com/bycmd/GoRouter-kit/security"
)

type Config struct {
	APIKey            string
	BaseURL           string
	Model             string
	AppName           string
	HTTPReferer       string
	Timeout           time.Duration
	MaxRetries        int
	RetryBaseDelay    time.Duration
	RetryMaxDelay     time.Duration
	ModelPrices       cost.PriceTable
	PriceRefreshEvery time.Duration
	RefreshPricesOnUp bool
	History           history.Store
	Security          security.Guard
	MaxToolCalls      int
	ParallelToolCalls bool
}

func (c Config) Prepare() Config {
	out := c
	if strings.TrimSpace(out.BaseURL) == "" {
		out.BaseURL = "https://openrouter.ai/api/v1"
	}
	out.BaseURL = strings.TrimRight(out.BaseURL, "/")
	if strings.TrimSpace(out.Model) == "" {
		out.Model = "openai/gpt-4o-mini"
	}
	if out.Timeout <= 0 {
		out.Timeout = 90 * time.Second
	}
	if out.MaxRetries < 0 {
		out.MaxRetries = 0
	}
	if out.RetryBaseDelay <= 0 {
		out.RetryBaseDelay = 300 * time.Millisecond
	}
	if out.RetryMaxDelay <= 0 {
		out.RetryMaxDelay = 4 * time.Second
	}
	if out.RetryMaxDelay < out.RetryBaseDelay {
		out.RetryMaxDelay = out.RetryBaseDelay
	}
	if out.MaxToolCalls <= 0 {
		out.MaxToolCalls = 5
	}
	return out
}

func (c Config) Validate() error {
	if strings.TrimSpace(c.APIKey) == "" {
		return errors.New("api key is required")
	}
	return nil
}
