package client

import (
	"context"
	"strconv"
	"strings"
	"time"

	"github.com/ifsix/GoRouter-kit/cost"
)

func (c *Client) RefreshModelPrices(ctx context.Context) error {
	models, err := c.GetModels(ctx)
	if err != nil {
		return err
	}

	out := make(cost.PriceTable)
	for _, item := range models {
		in, okIn := parsePricing(item.Pricing.Prompt)
		outPrice, okOut := parsePricing(item.Pricing.Completion)
		if !okIn || !okOut {
			continue
		}
		out[item.ID] = cost.Price{
			InputPer1K:  in * 1000.0,
			OutputPer1K: outPrice * 1000.0,
		}
	}

	if len(out) > 0 {
		c.costs.SetPrices(out)
	}
	return nil
}

func (c *Client) SetModelPrices(prices cost.PriceTable) {
	c.costs.SetPrices(prices)
}

func (c *Client) ModelPrices() cost.PriceTable {
	return c.costs.Prices()
}

func (c *Client) ModelPrice(model string) (cost.Price, bool) {
	return c.costs.Price(model)
}

func (c *Client) StartPriceAutoRefresh(ctx context.Context, interval time.Duration) {
	if interval <= 0 {
		return
	}

	c.priceRefreshMu.Lock()
	if c.priceRefreshCancel != nil {
		c.priceRefreshCancel()
	}

	runCtx, cancel := context.WithCancel(ctx)
	c.priceRefreshCancel = cancel
	c.priceRefreshMu.Unlock()

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-runCtx.Done():
				return
			case <-ticker.C:
				_ = c.RefreshModelPrices(runCtx)
			}
		}
	}()
}

func (c *Client) StopPriceAutoRefresh() {
	c.priceRefreshMu.Lock()
	cancel := c.priceRefreshCancel
	c.priceRefreshCancel = nil
	c.priceRefreshMu.Unlock()

	if cancel != nil {
		cancel()
	}
}

func parsePricing(raw string) (float64, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0, false
	}
	v, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return 0, false
	}
	return v, true
}
