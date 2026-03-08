package gorouter

import (
	"log"

	"github.com/ifsix/GoRouter-kit/client"
	"github.com/ifsix/GoRouter-kit/config"
	"github.com/ifsix/GoRouter-kit/cost"
	"github.com/ifsix/GoRouter-kit/errs"
	"github.com/ifsix/GoRouter-kit/history"
	historydisk "github.com/ifsix/GoRouter-kit/history/disk"
	historymemory "github.com/ifsix/GoRouter-kit/history/memory"
	historyredis "github.com/ifsix/GoRouter-kit/history/redis"
	"github.com/ifsix/GoRouter-kit/hooks"
	pluginpack "github.com/ifsix/GoRouter-kit/plugins"
	"github.com/ifsix/GoRouter-kit/schema"
	"github.com/ifsix/GoRouter-kit/security"
)

type Client = client.Client
type Config = config.Config

type HistoryStore = history.Store
type HistoryEntryStore = history.EntryStore
type HistoryEntry = history.HistoryEntry
type ApiCallMetadata = history.ApiCallMetadata
type HistoryManager = history.Manager
type HistoryManagerOptions = history.ManagerOptions
type HistoryAnalyzer = history.Analyzer
type HistoryQueryOptions = history.QueryOptions
type HistoryStats = history.Stats
type HistoryTimeSeriesPoint = history.TimeSeriesPoint
type MemoryHistoryStore = historymemory.Store
type DiskHistoryStore = historydisk.Store
type RedisHistoryStoreClient = historyredis.Client
type RedisHistoryStoreOptions = historyredis.Options
type RedisHistoryStore = historyredis.Store

type Hook = hooks.Hook
type HookFunc = hooks.Func
type Middleware = client.Middleware
type MiddlewareContext = client.MiddlewareContext
type Plugin = client.Plugin
type ClientEventHandler = client.EventHandler
type RedisHistoryPlugin = pluginpack.RedisHistoryPlugin
type RedisHistoryPluginOptions = pluginpack.RedisHistoryPluginOptions
type ToolRegistryPlugin = pluginpack.ToolRegistryPlugin
type ToolRegistryMode = pluginpack.ToolRegistryMode
type ToolRegistryOptions = pluginpack.ToolRegistryOptions
type ToolProvider = pluginpack.ToolProvider
type BillingPlugin = pluginpack.BillingPlugin
type BillingPluginOptions = pluginpack.BillingPluginOptions
type BillingReporter = pluginpack.BillingReporter
type BillingReport = pluginpack.BillingReport
type ExternalSecurityPlugin = pluginpack.ExternalSecurityPlugin
type GuardFactory = pluginpack.GuardFactory
type LoggingPlugin = pluginpack.LoggingPlugin
type MetricsPlugin = pluginpack.MetricsPlugin
type Metrics = pluginpack.Metrics

type Role = schema.Role

const (
	RoleSystem    = schema.RoleSystem
	RoleUser      = schema.RoleUser
	RoleAssistant = schema.RoleAssistant
	RoleTool      = schema.RoleTool
)

const (
	ToolRegistryOverride = pluginpack.ToolRegistryOverride
	ToolRegistryMerge    = pluginpack.ToolRegistryMerge
	ToolRegistryIfEmpty  = pluginpack.ToolRegistryIfEmpty
)

type Message = schema.Message
type Tool = schema.Tool
type ToolFunction = schema.ToolFunction
type ToolCall = schema.ToolCall
type ToolCallFunction = schema.ToolCallFunction
type Choice = schema.Choice
type Usage = schema.Usage
type ChatRequest = schema.ChatRequest
type ChatOptions = schema.ChatOptions
type Provider = schema.Provider
type ResponseFormat = schema.ResponseFormat
type ResponseJSONSchema = schema.ResponseJSONSchema
type Reasoning = schema.Reasoning
type ChatResponse = schema.ChatResponse
type ChatResult = schema.ChatResult
type CreditBalance = schema.CreditBalance
type APIKeyInfo = schema.APIKeyInfo
type ModelInfo = schema.ModelInfo
type ChatStreamChunk = schema.ChatStreamChunk
type StreamChoiceChunk = schema.StreamChoiceChunk
type MessageDelta = schema.MessageDelta
type StreamEvent = schema.StreamEvent
type StreamCallbacks = schema.StreamCallbacks
type ChatStreamResult = schema.ChatStreamResult
type ToolCallDetail = schema.ToolCallDetail
type ToolCallOutcome = schema.ToolCallOutcome
type ToolCallStatus = schema.ToolCallStatus
type ToolCallError = schema.ToolCallError
type ToolSecurity = schema.ToolSecurity

type Price = cost.Price
type PriceTable = cost.PriceTable
type ModelCost = cost.ModelCost
type CostSnapshot = cost.Snapshot
type CostTracker = cost.Tracker

type SecurityUser = security.User
type SecurityManager = security.Manager
type SecurityGuard = security.Guard
type SecurityAuthenticator = security.Authenticator
type SecurityConfig = security.Config
type SecurityToolPolicy = security.ToolPolicy
type SecurityRolePolicy = security.RolePolicy
type SecurityRateLimit = security.RateLimit
type SecurityAuthConfig = security.AuthConfig
type SecurityAuthType = security.AuthType
type SecurityDangerousArgumentsConfig = security.DangerousArgumentsConfig
type SecurityEventHandler = security.EventHandler
type SecurityToolCallEvent = security.ToolCallEvent

const (
	SecurityAuthTypeAPIKey = security.AuthTypeAPIKey
	SecurityAuthTypeJWT    = security.AuthTypeJWT
	SecurityAuthTypeCustom = security.AuthTypeCustom
)

var ErrBadRequest = errs.ErrBadRequest
var ErrDecode = errs.ErrDecode
var ErrAccessDenied = security.ErrAccessDenied
var ErrAuthRequired = security.ErrAuthRequired
var ErrRateLimited = security.ErrRateLimited
var ErrInvalidToken = security.ErrInvalidToken
var ErrDangerousArgument = security.ErrDangerousArgument

type APIError = errs.APIError

func New(cfg Config) (*Client, error) {
	return client.New(cfg)
}

func NewCostTracker(prices PriceTable) *CostTracker {
	return cost.NewTracker(prices)
}

func NewSimpleGuard() *security.SimpleGuard {
	return security.NewSimpleGuard()
}

func NewSecurityManager(cfg SecurityConfig) *security.Manager {
	return security.NewManager(cfg)
}

func NewHistoryManager(store HistoryEntryStore, opts HistoryManagerOptions) *HistoryManager {
	return history.NewManager(store, opts)
}

func NewHistoryAnalyzer(manager *HistoryManager) *HistoryAnalyzer {
	return history.NewAnalyzer(manager)
}

func NewMemoryHistoryStore() *MemoryHistoryStore {
	return historymemory.New()
}

func NewDiskHistoryStore(dir string) *DiskHistoryStore {
	return historydisk.New(dir)
}

func NewRedisHistoryStore(redisClient RedisHistoryStoreClient, opts RedisHistoryStoreOptions) *RedisHistoryStore {
	return historyredis.New(redisClient, opts)
}

func NewRedisHistoryPlugin(store HistoryStore) *RedisHistoryPlugin {
	return pluginpack.NewRedisHistoryPlugin(store)
}

func NewRedisHistoryPluginFromClient(redisClient RedisHistoryStoreClient, opts RedisHistoryPluginOptions) *RedisHistoryPlugin {
	return pluginpack.NewRedisHistoryPluginFromClient(redisClient, opts)
}

func NewToolRegistryPlugin(tools []Tool, opts ToolRegistryOptions) *ToolRegistryPlugin {
	return pluginpack.NewToolRegistryPlugin(tools, opts)
}

func NewDynamicToolRegistryPlugin(provider ToolProvider, opts ToolRegistryOptions) *ToolRegistryPlugin {
	return pluginpack.NewDynamicToolRegistryPlugin(provider, opts)
}

func NewBillingPlugin(opts BillingPluginOptions) *BillingPlugin {
	return pluginpack.NewBillingPlugin(opts)
}

func NewExternalSecurityPlugin(guard SecurityGuard) *ExternalSecurityPlugin {
	return pluginpack.NewExternalSecurityPlugin(guard)
}

func NewExternalSecurityPluginWithFactory(factory GuardFactory) *ExternalSecurityPlugin {
	return pluginpack.NewExternalSecurityPluginWithFactory(factory)
}

func NewLoggingPlugin() *LoggingPlugin {
	return pluginpack.NewLoggingPlugin(nil)
}

func NewLoggingPluginWithLogger(logger *log.Logger) *LoggingPlugin {
	return pluginpack.NewLoggingPlugin(logger)
}

func NewMetricsPlugin() *MetricsPlugin {
	return pluginpack.NewMetricsPlugin()
}
