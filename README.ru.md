# GoRouter Kit

Язык: [English](./README.md) | **Русский**

[![Go Version](https://img.shields.io/badge/Go-1.22-00ADD8?style=for-the-badge&logo=go&logoColor=white)](https://go.dev/)
[![Module](https://img.shields.io/badge/module-github.com%2Fifsix%2FGoRouter--kit-111827?style=for-the-badge)](https://github.com/ifsix/GoRouter-kit)
[![Go Reference](https://img.shields.io/badge/docs-pkg.go.dev-0A66C2?style=for-the-badge)](https://pkg.go.dev/github.com/ifsix/GoRouter-kit)
[![Stars](https://img.shields.io/github/stars/ifsix/GoRouter-kit?style=for-the-badge)](https://github.com/ifsix/GoRouter-kit/stargazers)
[![Last Commit](https://img.shields.io/github/last-commit/ifsix/GoRouter-kit?style=for-the-badge)](https://github.com/ifsix/GoRouter-kit/commits/main)
[![License](https://img.shields.io/github/license/ifsix/GoRouter-kit?style=for-the-badge)](./LICENSE)

GoRouter Kit — Go SDK для OpenRouter API, ориентированный на продакшн-нагрузку.

Проект является Go-форком, вдохновленным `openrouter-kit`, и переработан под идиоматичный Go и backend-сценарии.

## Установка

```bash
go get github.com/ifsix/GoRouter-kit
```

```go
import gorouter "github.com/ifsix/GoRouter-kit"
```

## Содержание

- [Быстрый старт](#быстрый-старт)
- [Конфигурация клиента](#конфигурация-клиента)
- [Примеры](#примеры)
- [Обработка ошибок](#обработка-ошибок)
- [Лицензия](#лицензия)

## Быстрый старт

```go
package main

import (
    "context"
    "fmt"
    "log"
    "os"

    gorouter "github.com/ifsix/GoRouter-kit"
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
            {Role: gorouter.RoleUser, Content: "Ответь одной короткой фразой."},
        },
    })
    if err != nil {
        log.Fatal(err)
    }

    fmt.Println(resp.Choices[0].Message.Content)
}
```

## Конфигурация клиента

Обязательное поле:
- `APIKey`

Часто используемые параметры:
- `Model` модель по умолчанию
- `BaseURL` кастомный API endpoint (по умолчанию OpenRouter v1)
- `Timeout`, `MaxRetries`, `RetryBaseDelay`, `RetryMaxDelay` настройки HTTP
- `History` backend для хранения истории
- `ModelPrices`, `RefreshPricesOnUp`, `PriceRefreshEvery` управление ценами
- `Security` guard для вызовов инструментов
- `MaxToolCalls`, `ParallelToolCalls` лимиты выполнения инструментов

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

## Примеры

### 1. Базовый Chat

```go
resp, err := cli.Chat(context.Background(), gorouter.ChatRequest{
    SessionID: "demo",
    Messages: []gorouter.Message{
        {Role: gorouter.RoleUser, Content: "Суммируй это одним предложением."},
    },
})
if err != nil {
    log.Fatal(err)
}
fmt.Println(resp.Choices[0].Message.Content)
```

### 2. Ask API с автоматическим ключом сессии

`Ask` собирает сообщения из параметров и может формировать ключ истории из `User` и `Group`.

```go
resp, err := cli.Ask(context.Background(), gorouter.ChatOptions{
    User:         "u-42",
    Group:        "team-a",
    SystemPrompt: "Отвечай кратко.",
    Prompt:       "Объясни retry с exponential backoff простыми словами.",
    Request: gorouter.ChatRequest{
        Model: "openai/gpt-4o-mini",
    },
})
if err != nil {
    log.Fatal(err)
}
fmt.Println(resp.Choices[0].Message.Content)
```

### 3. Стриминг ответа

```go
result, err := cli.ChatStreamResult(context.Background(), gorouter.ChatRequest{
    Messages: []gorouter.Message{
        {Role: gorouter.RoleUser, Content: "Дай короткий ответ в стриме."},
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

### 4. Структурированный ответ через `json_object`

```go
resp, err := cli.Chat(context.Background(), gorouter.ChatRequest{
    Messages: []gorouter.Message{
        {Role: gorouter.RoleUser, Content: "Верни JSON с полями title и score."},
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

### 5. Структурированный ответ через `json_schema`

```go
resp, err := cli.Chat(context.Background(), gorouter.ChatRequest{
    Messages: []gorouter.Message{
        {Role: gorouter.RoleUser, Content: "Верни JSON-сводку по пользователю Alice с возрастом и городом."},
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

### 6. Tool Calling с отчетом по вызовам

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
        {Role: gorouter.RoleUser, Content: "Используй tool sum для 7 и 5, затем ответь."},
    },
    Tools: []gorouter.Tool{
        {
            Type: "function",
            Function: gorouter.ToolFunction{
                Name:        "sum",
                Description: "Складывает два числа",
                Parameters:  params,
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

### 7. Service API методы

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

### 8. Обновление цен и снимок стоимости

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

### 9. Плагины

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

### 10. История и аналитика

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

### 11. Security guard для tool-вызовов

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

## Обработка ошибок

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

## Лицензия

MIT
