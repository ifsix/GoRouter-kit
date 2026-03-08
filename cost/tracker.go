package cost

import (
	"sync"

	"github.com/bycmd/GoRouter-kit/schema"
)

type Price struct {
	InputPer1K  float64
	OutputPer1K float64
}

type PriceTable map[string]Price

type ModelCost struct {
	PromptTokens     int64
	CompletionTokens int64
	TotalTokens      int64
	Cost             float64
}

type Snapshot struct {
	TotalTokens int64
	TotalCost   float64
	ByModel     map[string]ModelCost
}

type Tracker struct {
	mu      sync.Mutex
	prices  PriceTable
	byModel map[string]ModelCost
}

func NewTracker(prices PriceTable) *Tracker {
	if prices == nil {
		prices = PriceTable{}
	}
	return &Tracker{
		prices:  prices,
		byModel: map[string]ModelCost{},
	}
}

func (t *Tracker) Add(model string, usage schema.Usage) {
	t.mu.Lock()
	defer t.mu.Unlock()

	row := t.byModel[model]
	row.PromptTokens += int64(usage.PromptTokens)
	row.CompletionTokens += int64(usage.CompletionTokens)
	row.TotalTokens += int64(usage.TotalTokens)

	if price, ok := t.prices[model]; ok {
		row.Cost += (float64(usage.PromptTokens)/1000.0)*price.InputPer1K +
			(float64(usage.CompletionTokens)/1000.0)*price.OutputPer1K
	}

	t.byModel[model] = row
}

func (t *Tracker) Snapshot() Snapshot {
	t.mu.Lock()
	defer t.mu.Unlock()

	out := Snapshot{ByModel: map[string]ModelCost{}}
	for model, row := range t.byModel {
		out.ByModel[model] = row
		out.TotalTokens += row.TotalTokens
		out.TotalCost += row.Cost
	}
	return out
}

func (t *Tracker) SetPrices(prices PriceTable) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if prices == nil {
		t.prices = PriceTable{}
		return
	}

	out := make(PriceTable, len(prices))
	for model, price := range prices {
		out[model] = price
	}
	t.prices = out
}

func (t *Tracker) Prices() PriceTable {
	t.mu.Lock()
	defer t.mu.Unlock()

	out := make(PriceTable, len(t.prices))
	for model, price := range t.prices {
		out[model] = price
	}
	return out
}

func (t *Tracker) Price(model string) (Price, bool) {
	t.mu.Lock()
	defer t.mu.Unlock()

	v, ok := t.prices[model]
	return v, ok
}
