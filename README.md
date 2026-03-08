# GoRouter Kit

[![Go Version](https://img.shields.io/badge/Go-1.22-00ADD8?logo=go)](https://go.dev/)
[![Go Reference](https://pkg.go.dev/badge/github.com/bycmd/GoRouter-kit.svg)](https://pkg.go.dev/github.com/bycmd/GoRouter-kit)
[![Module](https://img.shields.io/badge/module-github.com%2Fbycmd%2FGoRouter--kit-24292f)](https://github.com/bycmd/GoRouter-kit)
[![License: MIT](https://img.shields.io/badge/License-MIT-2ea44f.svg)](./LICENSE)
[![Fork Base](https://img.shields.io/badge/fork%20base-openrouter--kit-2b6cb0)](https://github.com/bycmd/openrouter-kit)

[Русский](./README.ru.md) | **English**

GoRouter Kit is a production-focused Go SDK for the OpenRouter API.

The project is a Go fork inspired by `openrouter-kit`, rewritten for idiomatic Go and backend workloads.

## Installation

```bash
go get github.com/bycmd/GoRouter-kit
```

```go
import gorouter "github.com/bycmd/GoRouter-kit"
```

## Table of Contents

- [Quick Start](#quick-start)
- [Client Configuration](#client-configuration)
- [Examples](#examples)
- [Error Handling](#error-handling)
- [License](#license)

## Quick Start

```go
package main

import (
    "context"
    "fmt"
    "log"
    "os"

    gorouter "github.com/bycmd/GoRouter-kit"
)

func main() {
    cli, err := gorouter.New(gorouter.Config{
        APIKey: os.Getenv("OPENROUTER_API_KEY"),
        Model:  "openai/gpt-4o-mini",
    })
    if err != nil {
        log.Fatal(err)
    }
    defer cli.Destroy()

    resp, err := cli.Chat(context.Background(), gorouter.ChatRequest{
        Messages: []gorouter.Message{
            {Role: gorouter.RoleUser, Content: "Reply in one short sentence."},
        },
    })
    if err != nil {
        log.Fatal(err)
    }

    fmt.Println(resp.Choices[0].Message.Content)
}
```

## Client Configuration

Required:
- `APIKey`

Common options:
- `Model` default model for requests
- `BaseURL` custom API endpoint (default is OpenRouter v1)
- `Timeout`, `MaxRetries`, `RetryBaseDelay`, `RetryMaxDelay` HTTP behavior
- `History` storage backend for session persistence
- `ModelPrices`, `RefreshPricesOnUp`, `PriceRefreshEvery` pricing control
- `Security` tool-call guard
- `MaxToolCalls`, `ParallelToolCalls` tool execution limits

```go
cli, err := gorouter.New(gorouter.Config{
    APIKey:            os.Getenv("OPENROUTER_API_KEY"),
    Model:             "openai/gpt-4o-mini",
    Timeout:           90 * time.Second,
    MaxRetries:        3,
    RetryBaseDelay:    300 * time.Millisecond,
    RetryMaxDelay:     4 * time.Second,
    History:           gorouter.NewMemoryHistoryStore(),
    MaxToolCalls:      5,
    ParallelToolCalls: true,
})
```

## Examples

### 1. Basic Chat

```go
resp, err := cli.Chat(context.Background(), gorouter.ChatRequest{
    SessionID: "demo",
    Messages: []gorouter.Message{
        {Role: gorouter.RoleUser, Content: "Summarize this in one sentence."},
    },
})
if err != nil {
    log.Fatal(err)
}
fmt.Println(resp.Choices[0].Message.Content)
```

### 2. Ask API with Automatic Session Keys

`Ask` composes messages from prompt options and can build session keys from `User` and `Group`.

```go
resp, err := cli.Ask(context.Background(), gorouter.ChatOptions{
    User:         "u-42",
    Group:        "team-a",
    SystemPrompt: "You are concise.",
    Prompt:       "Explain what retry with exponential backoff means.",
    Request: gorouter.ChatRequest{
        Model: "openai/gpt-4o-mini",
    },
})
if err != nil {
    log.Fatal(err)
}
fmt.Println(resp.Choices[0].Message.Content)
```

### 3. Streaming Output

```go
result, err := cli.ChatStreamResult(context.Background(), gorouter.ChatRequest{
    Messages: []gorouter.Message{
        {Role: gorouter.RoleUser, Content: "Stream a short answer."},
    },
}, &gorouter.StreamCallbacks{
    OnContent: func(part string) {
        fmt.Print(part)
    },
})
if err != nil {
    log.Fatal(err)
}

fmt.Printf("\nmodel=%s finish=%s duration_ms=%d\n", result.Model, result.FinishReason, result.DurationMs)
```

### 4. Structured Output with `json_object`

```go
resp, err := cli.Chat(context.Background(), gorouter.ChatRequest{
    Messages: []gorouter.Message{
        {Role: gorouter.RoleUser, Content: "Return JSON with fields title and score."},
    },
    ResponseFormat: &gorouter.ResponseFormat{
        Type: "json_object",
    },
})
if err != nil {
    log.Fatal(err)
}

var out map[string]any
if err := json.Unmarshal([]byte(resp.Choices[0].Message.Content), &out); err != nil {
    log.Fatal(err)
}
fmt.Println(out)
```

### 5. Structured Output with `json_schema`

```go
resp, err := cli.Chat(context.Background(), gorouter.ChatRequest{
    Messages: []gorouter.Message{
        {Role: gorouter.RoleUser, Content: "Return JSON summary for user Alice with age and city."},
    },
    ResponseFormat: &gorouter.ResponseFormat{
        Type: "json_schema",
        JSONSchema: &gorouter.ResponseJSONSchema{
            Name:   "user_summary",
            Strict: true,
            Schema: map[string]any{
                "type": "object",
                "properties": map[string]any{
                    "name": map[string]any{"type": "string"},
                    "age":  map[string]any{"type": "integer"},
                    "city": map[string]any{"type": "string"},
                },
                "required": []string{"name", "age", "city"},
            },
        },
    },
})
if err != nil {
    log.Fatal(err)
}

var out map[string]any
if err := json.Unmarshal([]byte(resp.Choices[0].Message.Content), &out); err != nil {
    log.Fatal(err)
}
fmt.Println(out)
```

### 6. Tool Calling with Execution Report

```go
params, _ := json.Marshal(map[string]any{
    "type": "object",
    "properties": map[string]any{
        "a": map[string]any{"type": "number"},
        "b": map[string]any{"type": "number"},
    },
    "required": []string{"a", "b"},
})

report, err := cli.ChatWithReport(context.Background(), gorouter.ChatRequest{
    Messages: []gorouter.Message{
        {Role: gorouter.RoleUser, Content: "Use tool sum for 7 and 5, then answer."},
    },
    Tools: []gorouter.Tool{
        {
            Type: "function",
            Function: gorouter.ToolFunction{
                Name:       "sum",
                Description: "Adds two numbers",
                Parameters: params,
            },
            Execute: func(ctx context.Context, args any, call gorouter.ToolCall) (any, error) {
                m, _ := args.(map[string]any)
                a, _ := m["a"].(float64)
                b, _ := m["b"].(float64)
                return map[string]any{"value": a + b}, nil
            },
        },
    },
    IncludeToolResultInReport: true,
})
if err != nil {
    log.Fatal(err)
}

for _, c := range report.ToolCalls {
    fmt.Printf("tool=%s status=%s duration_ms=%d\n", c.ToolName, c.Status, c.DurationMs)
}
```

### 7. Service API Methods

```go
models, err := cli.GetModels(context.Background())
if err != nil {
    log.Fatal(err)
}

keyInfo, err := cli.GetAPIKeyInfo(context.Background())
if err != nil {
    log.Fatal(err)
}

balance, err := cli.GetCreditBalance(context.Background())
if err != nil {
    log.Fatal(err)
}

fmt.Println(len(models), keyInfo.Data.RateLimit.Interval, balance.TotalCredits)
```

### 8. Pricing Refresh and Cost Snapshot

```go
if err := cli.RefreshModelPrices(context.Background()); err != nil {
    log.Fatal(err)
}

cli.StartPriceAutoRefresh(context.Background(), 6*time.Hour)
defer cli.StopPriceAutoRefresh()

snap := cli.CostSnapshot()
fmt.Println(snap.TotalCost)

price, ok := cli.ModelPrice("openai/gpt-4o-mini")
fmt.Println(ok, price)
```

### 9. Plugins

```go
if err := cli.RegisterPlugin(gorouter.NewLoggingPlugin()); err != nil {
    log.Fatal(err)
}

metrics := gorouter.NewMetricsPlugin()
if err := cli.RegisterPlugin(metrics); err != nil {
    log.Fatal(err)
}

billing := gorouter.NewBillingPlugin(gorouter.BillingPluginOptions{
    Reporter: func(ctx context.Context, report gorouter.BillingReport) error {
        fmt.Printf("model=%s tokens=%d cost=%.6f\n", report.Model, report.Usage.TotalTokens, report.Cost)
        return nil
    },
})
if err := cli.RegisterPlugin(billing); err != nil {
    log.Fatal(err)
}

fmt.Printf("metrics=%+v\n", metrics.Snapshot())
```

### 10. History Adapters and Analyzer

```go
store := gorouter.NewMemoryHistoryStore()
cli.SetHistoryStore(store)

disk := gorouter.NewDiskHistoryStore(".gorouter-history")
cli.SetHistoryStore(disk)

manager := gorouter.NewHistoryManager(gorouter.NewMemoryHistoryStore(), gorouter.HistoryManagerOptions{
    TTL:             30 * time.Minute,
    CleanupInterval: 5 * time.Minute,
})
defer manager.Destroy()

analyzer := gorouter.NewHistoryAnalyzer(manager)
stats, err := analyzer.GetStats(context.Background(), "user:42", gorouter.HistoryQueryOptions{})
if err != nil {
    log.Fatal(err)
}
fmt.Println(stats.TotalAPICalls, stats.TotalCost)
```

### 11. Security Guard for Tool Calls

```go
allow := true

guard := gorouter.NewSecurityManager(gorouter.SecurityConfig{
    DefaultPolicy:         "deny-all",
    RequireAuthentication: true,
    Auth: gorouter.SecurityAuthConfig{
        Type: gorouter.SecurityAuthTypeAPIKey,
        APIKeys: map[string]gorouter.SecurityUser{
            "demo-token": {
                ID:     "u-1",
                Roles:  []string{"user"},
                Scopes: []string{"tools:run"},
            },
        },
    },
    Tools: map[string]gorouter.SecurityToolPolicy{
        "sum": {
            Allow:  &allow,
            Scopes: []string{"tools:run"},
        },
    },
})

cli.SetSecurityGuard(guard)
```

## Error Handling

```go
resp, err := cli.Chat(context.Background(), req)
if err != nil {
    var apiErr *gorouter.APIError
    if errors.As(err, &apiErr) {
        log.Printf("api error: status=%d message=%s", apiErr.StatusCode, apiErr.Message)
        return
    }
    if errors.Is(err, gorouter.ErrBadRequest) {
        log.Printf("bad request: %v", err)
        return
    }
    log.Printf("unexpected error: %v", err)
    return
}

_ = resp
```

## License

MIT
