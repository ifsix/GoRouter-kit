package schema

import (
	"context"
	"encoding/json"
	"time"
)

type Role string

const (
	RoleSystem    Role = "system"
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
	RoleTool      Role = "tool"
)

type Message struct {
	Role      Role       `json:"role"`
	Content   string     `json:"content,omitempty"`
	Name      string     `json:"name,omitempty"`
	ToolCall  string     `json:"tool_call_id,omitempty"`
	ToolCalls []ToolCall `json:"tool_calls,omitempty"`
	Refusal   string     `json:"refusal,omitempty"`
	AudioHint string     `json:"audio,omitempty"`
}

type Tool struct {
	Type     string        `json:"type"`
	Function ToolFunction  `json:"function"`
	Name     string        `json:"name,omitempty"`
	Execute  ToolExec      `json:"-"`
	Security *ToolSecurity `json:"-"`
}

type ToolExec func(ctx context.Context, args any, call ToolCall) (any, error)

type ToolFunction struct {
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	Parameters  json.RawMessage `json:"parameters,omitempty"`
}

type ToolCall struct {
	ID       string           `json:"id"`
	Type     string           `json:"type"`
	Function ToolCallFunction `json:"function"`
}

type ToolCallFunction struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type Choice struct {
	Index      int     `json:"index"`
	Message    Message `json:"message"`
	FinishType string  `json:"finish_reason,omitempty"`
}

type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

type ChatRequest struct {
	Model                     string             `json:"model,omitempty"`
	Messages                  []Message          `json:"messages"`
	Tools                     []Tool             `json:"tools,omitempty"`
	ToolChoice                interface{}        `json:"tool_choice,omitempty"`
	Provider                  *Provider          `json:"provider,omitempty"`
	Temperature               float64            `json:"temperature,omitempty"`
	MaxTokens                 int                `json:"max_tokens,omitempty"`
	TopP                      float64            `json:"top_p,omitempty"`
	PresencePenalty           float64            `json:"presence_penalty,omitempty"`
	FrequencyPenalty          float64            `json:"frequency_penalty,omitempty"`
	Stop                      any                `json:"stop,omitempty"`
	Seed                      int64              `json:"seed,omitempty"`
	LogitBias                 map[string]float64 `json:"logit_bias,omitempty"`
	ResponseFormat            *ResponseFormat    `json:"response_format,omitempty"`
	Route                     string             `json:"route,omitempty"`
	Transforms                []string           `json:"transforms,omitempty"`
	Models                    []string           `json:"models,omitempty"`
	Plugins                   []map[string]any   `json:"plugins,omitempty"`
	Reasoning                 *Reasoning         `json:"reasoning,omitempty"`
	Stream                    bool               `json:"stream,omitempty"`
	User                      string             `json:"user,omitempty"`
	AccessToken               string             `json:"-"`
	ParallelToolCalls         bool               `json:"-"`
	MaxToolCalls              int                `json:"-"`
	IncludeToolResultInReport bool               `json:"-"`
	SessionID                 string             `json:"-"`
}

type ChatOptions struct {
	Request        ChatRequest
	Prompt         string
	CustomMessages []Message
	SystemPrompt   string
	User           string
	Group          string
}

type Provider struct {
	Order             []string `json:"order,omitempty"`
	Only              []string `json:"only,omitempty"`
	Ignore            []string `json:"ignore,omitempty"`
	Quantizations     []string `json:"quantizations,omitempty"`
	Sort              string   `json:"sort,omitempty"`
	DataCollection    string   `json:"data_collection,omitempty"`
	AllowFallbacks    *bool    `json:"allow_fallbacks,omitempty"`
	RequireParameters *bool    `json:"require_parameters,omitempty"`
}

type ResponseFormat struct {
	Type       string              `json:"type"`
	JSONSchema *ResponseJSONSchema `json:"json_schema,omitempty"`
}

type ResponseJSONSchema struct {
	Name        string         `json:"name"`
	Strict      bool           `json:"strict,omitempty"`
	Schema      map[string]any `json:"schema"`
	Description string         `json:"description,omitempty"`
}

type Reasoning struct {
	Effort    string `json:"effort,omitempty"`
	MaxTokens int    `json:"max_tokens,omitempty"`
	Exclude   bool   `json:"exclude,omitempty"`
}

type ChatResponse struct {
	ID      string   `json:"id"`
	Model   string   `json:"model"`
	Choices []Choice `json:"choices"`
	Usage   Usage    `json:"usage"`
}

type ChatStreamChunk struct {
	ID      string              `json:"id"`
	Model   string              `json:"model"`
	Choices []StreamChoiceChunk `json:"choices"`
	Usage   *Usage              `json:"usage,omitempty"`
}

type StreamChoiceChunk struct {
	Index      int          `json:"index"`
	Delta      MessageDelta `json:"delta"`
	FinishType string       `json:"finish_reason,omitempty"`
	ToolCalls  []ToolCall   `json:"tool_calls,omitempty"`
}

type MessageDelta struct {
	Role      Role       `json:"role,omitempty"`
	Content   string     `json:"content,omitempty"`
	ToolCalls []ToolCall `json:"tool_calls,omitempty"`
}

type StreamEvent struct {
	Chunk ChatStreamChunk
	Done  bool
	Err   error
}

type StreamCallbacks struct {
	OnChunk             func(chunk ChatStreamChunk)
	OnContent           func(content string)
	OnToolCallExecuting func(toolName string, args any)
	OnToolCallResult    func(toolName string, result any)
	OnComplete          func(result ChatStreamResult)
	OnError             func(err error)
}

type ChatStreamResult struct {
	Content      string     `json:"content"`
	Usage        *Usage     `json:"usage,omitempty"`
	Model        string     `json:"model,omitempty"`
	FinishReason string     `json:"finish_reason,omitempty"`
	ID           string     `json:"id,omitempty"`
	ToolCalls    []ToolCall `json:"tool_calls,omitempty"`
	DurationMs   int64      `json:"duration_ms"`
}

type ToolSecurity struct {
	RequiredRole    string         `json:"required_role,omitempty"`
	RequiredAnyRole []string       `json:"required_any_role,omitempty"`
	RequiredScopes  []string       `json:"required_scopes,omitempty"`
	RateLimit       *ToolRateLimit `json:"rate_limit,omitempty"`
}

type ToolRateLimit struct {
	Limit  int           `json:"limit"`
	Window time.Duration `json:"window"`
}

type ToolCallStatus string

const (
	ToolCallSuccess         ToolCallStatus = "success"
	ToolCallErrorParsing    ToolCallStatus = "error_parsing"
	ToolCallErrorValidation ToolCallStatus = "error_validation"
	ToolCallErrorSecurity   ToolCallStatus = "error_security"
	ToolCallErrorExecution  ToolCallStatus = "error_execution"
	ToolCallErrorUnknown    ToolCallStatus = "error_unknown"
	ToolCallErrorNotFound   ToolCallStatus = "error_not_found"
)

type ToolCallError struct {
	Type    string `json:"type"`
	Message string `json:"message"`
	Details any    `json:"details,omitempty"`
}

type ToolCallDetail struct {
	ToolCallID     string         `json:"tool_call_id"`
	ToolName       string         `json:"tool_name"`
	RequestArgsRaw string         `json:"request_args_raw"`
	ParsedArgs     any            `json:"parsed_args,omitempty"`
	Status         ToolCallStatus `json:"status"`
	Result         any            `json:"result,omitempty"`
	Error          *ToolCallError `json:"error,omitempty"`
	ResultString   string         `json:"result_string"`
	DurationMs     int64          `json:"duration_ms"`
}

type ToolCallOutcome struct {
	Message Message        `json:"message"`
	Detail  ToolCallDetail `json:"detail"`
}

type ChatResult struct {
	Response   ChatResponse     `json:"response"`
	ToolCalls  []ToolCallDetail `json:"tool_calls,omitempty"`
	DurationMs int64            `json:"duration_ms"`
}

type CreditBalance struct {
	TotalCredits float64 `json:"total_credits"`
	TotalUsage   float64 `json:"total_usage"`
}

type APIKeyInfo struct {
	Data APIKeyData `json:"data"`
}

type APIKeyData struct {
	Limit      float64         `json:"limit"`
	Usage      float64         `json:"usage"`
	IsFreeTier bool            `json:"is_free_tier"`
	RateLimit  APIKeyRateLimit `json:"rate_limit"`
}

type APIKeyRateLimit struct {
	Requests int    `json:"requests"`
	Interval string `json:"interval"`
}

type ModelsEnvelope struct {
	Data []ModelInfo `json:"data"`
}

type ModelInfo struct {
	ID            string       `json:"id"`
	Name          string       `json:"name,omitempty"`
	ContextLength int          `json:"context_length,omitempty"`
	Pricing       ModelPricing `json:"pricing,omitempty"`
}

type ModelPricing struct {
	Prompt     string `json:"prompt,omitempty"`
	Completion string `json:"completion,omitempty"`
	Request    string `json:"request,omitempty"`
}
