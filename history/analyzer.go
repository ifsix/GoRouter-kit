package history

import (
	"context"
	"slices"
	"sort"
	"time"

	"github.com/ifsix/GoRouter-kit/schema"
)

type QueryOptions struct {
	StartDate     *time.Time
	EndDate       *time.Time
	Models        []string
	Roles         []schema.Role
	MinCost       *float64
	FinishReasons []string
}

type Stats struct {
	TotalAPICalls      int
	TotalCost          float64
	TotalUsage         schema.Usage
	UsageByModel       map[string]schema.Usage
	CostByModel        map[string]float64
	FinishReasonCounts map[string]int
	FirstEntryTS       int64
	LastEntryTS        int64
	EntriesAnalyzed    int
	EntriesTotal       int
}

type TimeSeriesPoint struct {
	Timestamp int64
	Value     float64
}

type Analyzer struct {
	manager *Manager
}

func NewAnalyzer(manager *Manager) *Analyzer {
	return &Analyzer{manager: manager}
}

func (a *Analyzer) GetStats(ctx context.Context, key string, opt QueryOptions) (Stats, error) {
	entries, err := a.manager.GetHistoryEntries(ctx, key)
	if err != nil {
		return Stats{}, err
	}
	filtered := filterEntries(entries, opt)

	out := Stats{
		UsageByModel:       map[string]schema.Usage{},
		CostByModel:        map[string]float64{},
		FinishReasonCounts: map[string]int{},
		EntriesAnalyzed:    len(filtered),
		EntriesTotal:       len(entries),
	}

	for _, item := range filtered {
		ts := entryTimestamp(item)
		if ts > 0 {
			if out.FirstEntryTS == 0 || ts < out.FirstEntryTS {
				out.FirstEntryTS = ts
			}
			if ts > out.LastEntryTS {
				out.LastEntryTS = ts
			}
		}

		meta := item.ApiCallMetadata
		if meta == nil {
			continue
		}

		out.TotalAPICalls++
		if meta.Cost != nil {
			out.TotalCost += *meta.Cost
			out.CostByModel[meta.ModelUsed] += *meta.Cost
		}

		if meta.Usage != nil {
			out.TotalUsage.PromptTokens += meta.Usage.PromptTokens
			out.TotalUsage.CompletionTokens += meta.Usage.CompletionTokens
			out.TotalUsage.TotalTokens += meta.Usage.TotalTokens

			usage := out.UsageByModel[meta.ModelUsed]
			usage.PromptTokens += meta.Usage.PromptTokens
			usage.CompletionTokens += meta.Usage.CompletionTokens
			usage.TotalTokens += meta.Usage.TotalTokens
			out.UsageByModel[meta.ModelUsed] = usage
		}

		reason := meta.FinishReason
		if reason == "" {
			reason = "unknown"
		}
		out.FinishReasonCounts[reason]++
	}

	return out, nil
}

func (a *Analyzer) GetCostOverTime(ctx context.Context, key string, interval string, opt QueryOptions) ([]TimeSeriesPoint, error) {
	entries, err := a.manager.GetHistoryEntries(ctx, key)
	if err != nil {
		return nil, err
	}
	filtered := filterEntries(entries, opt)

	buckets := map[int64]float64{}
	for _, item := range filtered {
		meta := item.ApiCallMetadata
		if meta == nil || meta.Cost == nil || meta.Timestamp <= 0 {
			continue
		}

		slot := bucketStart(meta.Timestamp, interval)
		buckets[slot] += *meta.Cost
	}

	out := make([]TimeSeriesPoint, 0, len(buckets))
	for ts, v := range buckets {
		out = append(out, TimeSeriesPoint{
			Timestamp: ts,
			Value:     v,
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Timestamp < out[j].Timestamp })
	return out, nil
}

func (a *Analyzer) GetTokenUsageByModel(ctx context.Context, key string, opt QueryOptions) (map[string]schema.Usage, error) {
	entries, err := a.manager.GetHistoryEntries(ctx, key)
	if err != nil {
		return nil, err
	}
	filtered := filterEntries(entries, opt)

	out := map[string]schema.Usage{}
	for _, item := range filtered {
		meta := item.ApiCallMetadata
		if meta == nil || meta.Usage == nil {
			continue
		}
		usage := out[meta.ModelUsed]
		usage.PromptTokens += meta.Usage.PromptTokens
		usage.CompletionTokens += meta.Usage.CompletionTokens
		usage.TotalTokens += meta.Usage.TotalTokens
		out[meta.ModelUsed] = usage
	}
	return out, nil
}

func filterEntries(entries []HistoryEntry, opt QueryOptions) []HistoryEntry {
	if len(entries) == 0 {
		return nil
	}

	out := make([]HistoryEntry, 0, len(entries))
	for _, item := range entries {
		if !matchesDate(item, opt) {
			continue
		}
		if len(opt.Roles) > 0 && !slices.Contains(opt.Roles, item.Message.Role) {
			continue
		}

		meta := item.ApiCallMetadata
		if len(opt.Models) > 0 {
			if meta == nil || !slices.Contains(opt.Models, meta.ModelUsed) {
				continue
			}
		}
		if opt.MinCost != nil {
			if meta == nil || meta.Cost == nil || *meta.Cost < *opt.MinCost {
				continue
			}
		}
		if len(opt.FinishReasons) > 0 {
			if meta == nil || !slices.Contains(opt.FinishReasons, meta.FinishReason) {
				continue
			}
		}

		out = append(out, item)
	}
	return out
}

func matchesDate(item HistoryEntry, opt QueryOptions) bool {
	start := opt.StartDate
	end := opt.EndDate
	if start == nil && end == nil {
		return true
	}

	ts := entryTimestamp(item)
	if ts <= 0 {
		return false
	}

	t := time.UnixMilli(ts)
	if start != nil && t.Before(*start) {
		return false
	}
	if end != nil && t.After(*end) {
		return false
	}
	return true
}

func entryTimestamp(item HistoryEntry) int64 {
	if item.ApiCallMetadata != nil && item.ApiCallMetadata.Timestamp > 0 {
		return item.ApiCallMetadata.Timestamp
	}
	return 0
}

func bucketStart(ts int64, interval string) int64 {
	date := time.UnixMilli(ts).UTC()
	switch interval {
	case "minute":
		date = time.Date(date.Year(), date.Month(), date.Day(), date.Hour(), date.Minute(), 0, 0, time.UTC)
	case "hour":
		date = time.Date(date.Year(), date.Month(), date.Day(), date.Hour(), 0, 0, 0, time.UTC)
	default:
		date = time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, time.UTC)
	}
	return date.UnixMilli()
}
