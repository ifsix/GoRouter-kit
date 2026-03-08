package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/bycmd/GoRouter-kit"
)

func main() {
	store := gorouter.NewMemoryHistoryStore()
	manager := gorouter.NewHistoryManager(store, gorouter.HistoryManagerOptions{
		TTL:             10 * time.Minute,
		CleanupInterval: time.Minute,
	})
	defer manager.Destroy()

	analyzer := gorouter.NewHistoryAnalyzer(manager)
	now := time.Now().UnixMilli()
	cost := 0.031

	err := manager.AddHistoryEntries(context.Background(), "user:42", []gorouter.HistoryEntry{
		{
			Message: gorouter.Message{
				Role:    gorouter.RoleUser,
				Content: "hello",
			},
		},
		{
			Message: gorouter.Message{
				Role:    gorouter.RoleAssistant,
				Content: "hi",
			},
			ApiCallMetadata: &gorouter.ApiCallMetadata{
				CallID:       "call-1",
				ModelUsed:    "openai/gpt-4o-mini",
				Usage:        &gorouter.Usage{PromptTokens: 20, CompletionTokens: 40, TotalTokens: 60},
				Cost:         &cost,
				Timestamp:    now,
				FinishReason: "stop",
			},
		},
	})
	if err != nil {
		log.Fatal(err)
	}

	stats, err := analyzer.GetStats(context.Background(), "user:42", gorouter.HistoryQueryOptions{})
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("api_calls=%d total_cost=%.6f total_tokens=%d\n", stats.TotalAPICalls, stats.TotalCost, stats.TotalUsage.TotalTokens)
}
