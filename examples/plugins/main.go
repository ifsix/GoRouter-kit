package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/bycmd/GoRouter-kit"
)

func main() {
	key := os.Getenv("OPENROUTER_API_KEY")
	if key == "" {
		log.Fatal("OPENROUTER_API_KEY is required")
	}

	cli, err := gorouter.New(gorouter.Config{
		APIKey:  key,
		Model:   "openai/gpt-4o-mini",
		History: gorouter.NewMemoryHistoryStore(),
		ModelPrices: gorouter.PriceTable{
			"openai/gpt-4o-mini": {
				InputPer1K:  0.15,
				OutputPer1K: 0.60,
			},
		},
	})
	if err != nil {
		log.Fatal(err)
	}
	defer cli.Destroy()

	metrics := gorouter.NewMetricsPlugin()
	if err := cli.RegisterPlugin(metrics); err != nil {
		log.Fatal(err)
	}

	billing := gorouter.NewBillingPlugin(gorouter.BillingPluginOptions{
		Reporter: func(ctx context.Context, report gorouter.BillingReport) error {
			fmt.Printf("[billing] model=%s tokens=%d cost=%.6f has_price=%v\n", report.Model, report.Usage.TotalTokens, report.Cost, report.HasPrice)
			return nil
		},
	})
	if err := cli.RegisterPlugin(billing); err != nil {
		log.Fatal(err)
	}

	registry := gorouter.NewToolRegistryPlugin([]gorouter.Tool{
		{
			Type: "function",
			Function: gorouter.ToolFunction{
				Name:        "sum",
				Description: "sum two numbers",
			},
			Execute: func(ctx context.Context, args any, call gorouter.ToolCall) (any, error) {
				m, _ := args.(map[string]any)
				a, _ := m["a"].(float64)
				b, _ := m["b"].(float64)
				return map[string]any{"value": a + b}, nil
			},
		},
	}, gorouter.ToolRegistryOptions{Mode: gorouter.ToolRegistryIfEmpty})
	if err := cli.RegisterPlugin(registry); err != nil {
		log.Fatal(err)
	}

	resp, err := cli.Chat(context.Background(), gorouter.ChatRequest{
		SessionID: "plugins-demo",
		Messages: []gorouter.Message{
			{Role: gorouter.RoleUser, Content: "Say hi in one short sentence"},
		},
	})
	if err != nil {
		log.Fatal(err)
	}
	if len(resp.Choices) == 0 {
		log.Fatal("empty response")
	}

	fmt.Println(resp.Choices[0].Message.Content)
	fmt.Printf("metrics=%+v\n", metrics.Snapshot())
}
