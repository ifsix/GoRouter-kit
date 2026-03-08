package client

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

func (c *Client) doRequest(ctx context.Context, payload []byte) (*http.Response, error) {
	tries := c.cfg.MaxRetries + 1
	if tries < 1 {
		tries = 1
	}

	var lastErr error
	for attempt := 0; attempt < tries; attempt++ {
		resp, err := c.requestOnce(ctx, payload)
		if err != nil {
			lastErr = err
			if attempt >= tries-1 || !canRetryErr(err) {
				return nil, err
			}
			if err := waitRetry(ctx, c.retryDelay(attempt, "")); err != nil {
				return nil, err
			}
			continue
		}

		if !canRetryStatus(resp.StatusCode) || attempt >= tries-1 {
			return resp, nil
		}

		delay := c.retryDelay(attempt, resp.Header.Get("Retry-After"))
		_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 64*1024))
		_ = resp.Body.Close()

		if err := waitRetry(ctx, delay); err != nil {
			return nil, err
		}
	}

	if lastErr != nil {
		return nil, lastErr
	}
	return nil, errors.New("request failed")
}

func (c *Client) requestOnce(ctx context.Context, payload []byte) (*http.Response, error) {
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.cfg.BaseURL+"/chat/completions", bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	c.setHeaders(httpReq)
	return c.httpClient.Do(httpReq)
}

func (c *Client) retryDelay(attempt int, retryAfter string) time.Duration {
	if d := parseRetryAfter(retryAfter); d > 0 {
		return d
	}

	delay := c.cfg.RetryBaseDelay
	for i := 0; i < attempt; i++ {
		delay *= 2
		if delay >= c.cfg.RetryMaxDelay {
			return c.cfg.RetryMaxDelay
		}
	}
	if delay > c.cfg.RetryMaxDelay {
		return c.cfg.RetryMaxDelay
	}
	return delay
}

func parseRetryAfter(raw string) time.Duration {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0
	}

	if secs, err := strconv.Atoi(raw); err == nil && secs > 0 {
		return time.Duration(secs) * time.Second
	}

	when, err := http.ParseTime(raw)
	if err != nil {
		return 0
	}
	wait := time.Until(when)
	if wait < 0 {
		return 0
	}
	return wait
}

func canRetryStatus(code int) bool {
	switch code {
	case http.StatusRequestTimeout,
		http.StatusConflict,
		http.StatusTooEarly,
		http.StatusTooManyRequests,
		http.StatusInternalServerError,
		http.StatusBadGateway,
		http.StatusServiceUnavailable,
		http.StatusGatewayTimeout:
		return true
	default:
		return false
	}
}

func canRetryErr(err error) bool {
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}

	var uerr *url.Error
	if errors.As(err, &uerr) {
		if errors.Is(uerr.Err, context.Canceled) || errors.Is(uerr.Err, context.DeadlineExceeded) {
			return false
		}
	}
	return true
}

func waitRetry(ctx context.Context, delay time.Duration) error {
	if delay <= 0 {
		return nil
	}

	timer := time.NewTimer(delay)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
