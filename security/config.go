package security

import (
	"context"
	"time"
)

type AuthType string

const (
	AuthTypeAPIKey AuthType = "api-key"
	AuthTypeJWT    AuthType = "jwt"
	AuthTypeCustom AuthType = "custom"
)

type RateLimit struct {
	Limit  int
	Window time.Duration
}

type ToolPolicy struct {
	Allow     *bool
	Roles     []string
	Scopes    []string
	RateLimit *RateLimit
}

type RolePolicy struct {
	AllowedTools []string
	RateLimits   map[string]RateLimit
}

type CustomAuthenticator func(ctx context.Context, token string) (*User, error)

type AuthConfig struct {
	Type                AuthType
	APIKeys             map[string]User
	JWTSecret           string
	TokenTTL            time.Duration
	CustomAuthenticator CustomAuthenticator
}

type DangerousArgumentsConfig struct {
	GlobalPatterns []string
	ToolPatterns   map[string][]string
	BlockedValues  []string
	AuditOnly      bool
}

type Config struct {
	DefaultPolicy              string
	RequireAuthentication      bool
	AllowUnauthenticatedAccess bool
	Debug                      bool
	TokenCacheTTL              time.Duration
	DangerousArguments         DangerousArgumentsConfig
	Auth                       AuthConfig
	Tools                      map[string]ToolPolicy
	Roles                      map[string]RolePolicy
}

func (c Config) prepare() Config {
	out := c
	if out.DefaultPolicy == "" {
		out.DefaultPolicy = "deny-all"
	}
	if out.TokenCacheTTL <= 0 {
		out.TokenCacheTTL = 5 * time.Minute
	}
	if out.Tools == nil {
		out.Tools = map[string]ToolPolicy{}
	}
	if out.Roles == nil {
		out.Roles = map[string]RolePolicy{}
	}
	if out.DangerousArguments.ToolPatterns == nil {
		out.DangerousArguments.ToolPatterns = map[string][]string{}
	}
	if out.Auth.APIKeys == nil {
		out.Auth.APIKeys = map[string]User{}
	}
	if out.Auth.Type == "" {
		out.Auth.Type = AuthTypeAPIKey
	}
	if out.Auth.TokenTTL <= 0 {
		out.Auth.TokenTTL = 24 * time.Hour
	}
	return out
}
