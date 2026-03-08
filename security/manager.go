package security

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"regexp"
	"slices"
	"strings"
	"sync"
	"time"
)

type cachedAuth struct {
	user   *User
	cached time.Time
}

type jwtHeader struct {
	Alg string `json:"alg"`
	Typ string `json:"typ"`
}

type jwtPayload struct {
	Sub      string         `json:"sub"`
	Role     string         `json:"role,omitempty"`
	Roles    []string       `json:"roles,omitempty"`
	Scopes   []string       `json:"scopes,omitempty"`
	Username string         `json:"username,omitempty"`
	Metadata map[string]any `json:"metadata,omitempty"`
	Iat      int64          `json:"iat,omitempty"`
	Exp      int64          `json:"exp,omitempty"`
}

type Manager struct {
	mu         sync.RWMutex
	cfg        Config
	limiter    *Limiter
	tokenCache map[string]cachedAuth
	events     *eventBus
}

func NewManager(cfg Config) *Manager {
	return &Manager{
		cfg:        cfg.prepare(),
		limiter:    NewLimiter(),
		tokenCache: map[string]cachedAuth{},
		events:     newEventBus(),
	}
}

func (m *Manager) GetConfig() Config {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.cfg
}

func (m *Manager) UpdateConfig(cfg Config) {
	m.mu.Lock()
	m.cfg = cfg.prepare()
	m.mu.Unlock()
	m.events.emit("config:updated", m.GetConfig())
}

func (m *Manager) On(event string, fn EventHandler) int {
	return m.events.on(event, fn)
}

func (m *Manager) Off(event string, id int) {
	m.events.off(event, id)
}

func (m *Manager) ClearTokenCache() {
	m.mu.Lock()
	m.tokenCache = map[string]cachedAuth{}
	m.mu.Unlock()
	m.events.emit("auth:cache:cleared", nil)
}

func (m *Manager) ClearRateLimitCounters(userID string) {
	if strings.TrimSpace(userID) == "" {
		m.limiter.Reset()
	} else {
		m.limiter.ResetUser(userID)
	}
	m.events.emit("ratelimit:cleared", map[string]any{"user_id": userID})
}

func (m *Manager) Destroy() {
	m.ClearTokenCache()
	m.ClearRateLimitCounters("")
}

func (m *Manager) Authenticate(ctx context.Context, token string) (*User, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return nil, nil
	}

	if user, ok := m.cachedUser(token); ok {
		m.events.emit("auth:success", map[string]any{"user": user, "cached": true})
		return user, nil
	}

	cfg := m.GetConfig()
	user, err := m.authenticateByType(ctx, cfg, token)
	if err != nil {
		m.events.emit("auth:failed", map[string]any{"error": err.Error()})
		return nil, err
	}
	if user == nil {
		m.events.emit("auth:failed", map[string]any{"error": ErrInvalidToken.Error()})
		return nil, ErrInvalidToken
	}

	m.cacheUser(token, user)
	m.events.emit("auth:success", map[string]any{"user": user, "cached": false})
	return user, nil
}

func (m *Manager) CreateAccessToken(user User, expiresIn time.Duration) (string, error) {
	cfg := m.GetConfig()
	if cfg.Auth.Type != AuthTypeJWT {
		return "", fmt.Errorf("%w: auth type %q does not support token creation", ErrTokenCreate, cfg.Auth.Type)
	}
	secret := strings.TrimSpace(cfg.Auth.JWTSecret)
	if secret == "" {
		return "", fmt.Errorf("%w: jwt secret is empty", ErrAuthConfig)
	}

	if strings.TrimSpace(user.ID) == "" {
		return "", fmt.Errorf("%w: user id is required", ErrTokenCreate)
	}
	if expiresIn <= 0 {
		expiresIn = cfg.Auth.TokenTTL
	}
	if expiresIn <= 0 {
		expiresIn = 24 * time.Hour
	}

	now := time.Now().Unix()
	payload := jwtPayload{
		Sub:      user.ID,
		Role:     user.Role,
		Roles:    user.Roles,
		Scopes:   user.Scopes,
		Username: user.Username,
		Metadata: user.Metadata,
		Iat:      now,
		Exp:      now + int64(expiresIn.Seconds()),
	}

	headerRaw, _ := json.Marshal(jwtHeader{Alg: "HS256", Typ: "JWT"})
	payloadRaw, _ := json.Marshal(payload)

	enc := base64.RawURLEncoding
	headerPart := enc.EncodeToString(headerRaw)
	payloadPart := enc.EncodeToString(payloadRaw)
	signed := headerPart + "." + payloadPart
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(signed))
	sigPart := enc.EncodeToString(mac.Sum(nil))

	return signed + "." + sigPart, nil
}

func (m *Manager) Check(_ context.Context, in CheckInput) error {
	cfg := m.GetConfig()

	toolName := in.Tool.Function.Name
	if toolName == "" {
		toolName = in.Tool.Name
	}
	if toolName == "" {
		toolName = "unknown_tool"
	}

	if in.User == nil && cfg.RequireAuthentication && !cfg.AllowUnauthenticatedAccess {
		err := ErrAuthRequired
		m.events.emit("access:denied", map[string]any{"tool": toolName, "error": err.Error()})
		return err
	}

	if err := checkToolSecurity(toolName, in); err != nil {
		m.events.emit("access:denied", map[string]any{"tool": toolName, "error": err.Error()})
		return err
	}
	if err := m.checkConfigPolicy(cfg, toolName, in.User); err != nil {
		m.events.emit("access:denied", map[string]any{"tool": toolName, "error": err.Error()})
		return err
	}
	if err := m.checkDangerousArgs(cfg, toolName, in.Args); err != nil {
		m.events.emit("security:error", map[string]any{"tool": toolName, "error": err.Error()})
		return err
	}
	if err := m.checkRateLimit(cfg, toolName, in); err != nil {
		m.events.emit("ratelimit:hit", map[string]any{"tool": toolName, "error": err.Error()})
		return err
	}

	m.events.emit("access:granted", map[string]any{"tool": toolName, "user": in.User})
	return nil
}

func (m *Manager) LogToolCall(event ToolCallEvent) {
	m.events.emit("tool:call", event)
}

func (m *Manager) authenticateByType(ctx context.Context, cfg Config, token string) (*User, error) {
	switch cfg.Auth.Type {
	case AuthTypeCustom:
		if cfg.Auth.CustomAuthenticator == nil {
			return nil, fmt.Errorf("%w: custom authenticator is not configured", ErrAuthConfig)
		}
		return cfg.Auth.CustomAuthenticator(ctx, token)
	case AuthTypeJWT:
		return parseJWT(token, cfg.Auth.JWTSecret)
	case AuthTypeAPIKey, "":
		user, ok := cfg.Auth.APIKeys[token]
		if !ok {
			return nil, ErrInvalidToken
		}
		u := user
		return &u, nil
	default:
		return nil, fmt.Errorf("%w: unsupported auth type %q", ErrAuthConfig, cfg.Auth.Type)
	}
}

func (m *Manager) cachedUser(token string) (*User, bool) {
	m.mu.RLock()
	cached, ok := m.tokenCache[token]
	cfg := m.cfg
	m.mu.RUnlock()
	if !ok {
		return nil, false
	}
	if cfg.TokenCacheTTL > 0 && time.Since(cached.cached) > cfg.TokenCacheTTL {
		m.mu.Lock()
		delete(m.tokenCache, token)
		m.mu.Unlock()
		return nil, false
	}
	if cached.user != nil && cached.user.ExpiresAt > 0 && time.Now().Unix() >= cached.user.ExpiresAt {
		m.mu.Lock()
		delete(m.tokenCache, token)
		m.mu.Unlock()
		return nil, false
	}
	u := *cached.user
	return &u, true
}

func (m *Manager) cacheUser(token string, user *User) {
	if token == "" || user == nil {
		return
	}
	u := *user
	m.mu.Lock()
	m.tokenCache[token] = cachedAuth{user: &u, cached: time.Now()}
	m.mu.Unlock()
}

func checkToolSecurity(toolName string, in CheckInput) error {
	sec := in.Tool.Security
	if sec == nil {
		return nil
	}

	if sec.RequiredRole != "" {
		if in.User == nil {
			return fmt.Errorf("%w: role %q required for %s", ErrAccessDenied, sec.RequiredRole, toolName)
		}
		if in.User.Role != sec.RequiredRole && !slices.Contains(in.User.Roles, sec.RequiredRole) {
			return fmt.Errorf("%w: role %q required for %s", ErrAccessDenied, sec.RequiredRole, toolName)
		}
	}

	if len(sec.RequiredAnyRole) > 0 {
		if in.User == nil {
			return fmt.Errorf("%w: role is required for %s", ErrAccessDenied, toolName)
		}
		matched := false
		for _, role := range sec.RequiredAnyRole {
			if in.User.Role == role || slices.Contains(in.User.Roles, role) {
				matched = true
				break
			}
		}
		if !matched {
			return fmt.Errorf("%w: none of required roles matched for %s", ErrAccessDenied, toolName)
		}
	}

	if len(sec.RequiredScopes) > 0 {
		if in.User == nil {
			return fmt.Errorf("%w: scopes required for %s", ErrAccessDenied, toolName)
		}
		for _, scope := range sec.RequiredScopes {
			if !slices.Contains(in.User.Scopes, scope) {
				return fmt.Errorf("%w: missing scope %q for %s", ErrAccessDenied, scope, toolName)
			}
		}
	}

	return nil
}

func (m *Manager) checkConfigPolicy(cfg Config, toolName string, user *User) error {
	policy, hasToolPolicy := cfg.Tools[toolName]
	globalPolicy, hasGlobalPolicy := cfg.Tools["*"]

	if hasToolPolicy && policy.Allow != nil && !*policy.Allow {
		return fmt.Errorf("%w: tool %s is blocked by policy", ErrAccessDenied, toolName)
	}

	if hasToolPolicy && len(policy.Roles) > 0 {
		if user == nil {
			return fmt.Errorf("%w: tool %s requires role", ErrAccessDenied, toolName)
		}
		if !hasAnyRole(user, policy.Roles) {
			return fmt.Errorf("%w: role is not allowed for %s", ErrAccessDenied, toolName)
		}
	}

	if hasToolPolicy && len(policy.Scopes) > 0 {
		if user == nil {
			return fmt.Errorf("%w: tool %s requires scope", ErrAccessDenied, toolName)
		}
		for _, scope := range policy.Scopes {
			if !slices.Contains(user.Scopes, scope) {
				return fmt.Errorf("%w: missing scope %q for %s", ErrAccessDenied, scope, toolName)
			}
		}
	}

	if hasToolPolicy && policy.Allow != nil && *policy.Allow {
		return nil
	}

	if user != nil {
		if roleCfg, ok := cfg.Roles[user.Role]; ok {
			if len(roleCfg.AllowedTools) == 0 || slices.Contains(roleCfg.AllowedTools, "*") || slices.Contains(roleCfg.AllowedTools, toolName) {
				return nil
			}
		}
	}

	if hasGlobalPolicy && globalPolicy.Allow != nil {
		if *globalPolicy.Allow {
			return nil
		}
		return fmt.Errorf("%w: global tool policy denies access", ErrAccessDenied)
	}

	if cfg.DefaultPolicy == "allow-all" {
		return nil
	}
	return fmt.Errorf("%w: denied by default policy", ErrAccessDenied)
}

func (m *Manager) checkDangerousArgs(cfg Config, toolName string, args any) error {
	if args == nil {
		return nil
	}

	raw, err := json.Marshal(args)
	if err != nil {
		return nil
	}
	text := string(raw)
	lower := strings.ToLower(text)

	for _, v := range cfg.DangerousArguments.BlockedValues {
		if v == "" {
			continue
		}
		if strings.Contains(lower, strings.ToLower(v)) {
			if cfg.DangerousArguments.AuditOnly {
				m.events.emit("security:audit", map[string]any{"tool": toolName, "reason": "blocked_value", "value": v})
				return nil
			}
			return fmt.Errorf("%w: blocked value %q", ErrDangerousArgument, v)
		}
	}

	patterns := make([]string, 0, len(cfg.DangerousArguments.GlobalPatterns)+len(cfg.DangerousArguments.ToolPatterns[toolName]))
	patterns = append(patterns, cfg.DangerousArguments.GlobalPatterns...)
	patterns = append(patterns, cfg.DangerousArguments.ToolPatterns[toolName]...)

	for _, p := range patterns {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}

		re, err := regexp.Compile(p)
		if err != nil {
			if strings.Contains(lower, strings.ToLower(p)) {
				if cfg.DangerousArguments.AuditOnly {
					m.events.emit("security:audit", map[string]any{"tool": toolName, "reason": "pattern", "pattern": p})
					return nil
				}
				return fmt.Errorf("%w: matched pattern %q", ErrDangerousArgument, p)
			}
			continue
		}

		if re.MatchString(text) {
			if cfg.DangerousArguments.AuditOnly {
				m.events.emit("security:audit", map[string]any{"tool": toolName, "reason": "pattern", "pattern": p})
				return nil
			}
			return fmt.Errorf("%w: matched pattern %q", ErrDangerousArgument, p)
		}
	}

	return nil
}

func (m *Manager) checkRateLimit(cfg Config, toolName string, in CheckInput) error {
	limit, ok := resolveRateLimit(cfg, toolName, in)
	if !ok {
		return nil
	}

	userID := ""
	if in.User != nil {
		userID = in.User.ID
	}

	key := m.limiter.key(userID, toolName)
	allowed, wait := m.limiter.Allow(key, limit.Limit, limit.Window)
	if allowed {
		return nil
	}

	return fmt.Errorf("%w: retry after %s", ErrRateLimited, wait.Round(time.Second))
}

func resolveRateLimit(cfg Config, toolName string, in CheckInput) (RateLimit, bool) {
	if in.Tool.Security != nil && in.Tool.Security.RateLimit != nil {
		v := in.Tool.Security.RateLimit
		return RateLimit{Limit: v.Limit, Window: v.Window}, true
	}

	if policy, ok := cfg.Tools[toolName]; ok && policy.RateLimit != nil {
		return *policy.RateLimit, true
	}
	if policy, ok := cfg.Tools["*"]; ok && policy.RateLimit != nil {
		return *policy.RateLimit, true
	}

	if in.User != nil {
		if roleCfg, ok := cfg.Roles[in.User.Role]; ok {
			if limit, ok := roleCfg.RateLimits[toolName]; ok {
				return limit, true
			}
			if limit, ok := roleCfg.RateLimits["*"]; ok {
				return limit, true
			}
		}
	}

	return RateLimit{}, false
}

func hasAnyRole(user *User, required []string) bool {
	for _, role := range required {
		if user.Role == role || slices.Contains(user.Roles, role) {
			return true
		}
	}
	return false
}

func parseJWT(token, secret string) (*User, error) {
	if strings.TrimSpace(secret) == "" {
		return nil, fmt.Errorf("%w: jwt secret is empty", ErrAuthConfig)
	}

	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, ErrInvalidToken
	}

	enc := base64.RawURLEncoding
	signed := parts[0] + "." + parts[1]

	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(signed))
	expected := mac.Sum(nil)
	received, err := enc.DecodeString(parts[2])
	if err != nil {
		return nil, ErrInvalidToken
	}
	if !hmac.Equal(received, expected) {
		return nil, ErrInvalidToken
	}

	payloadRaw, err := enc.DecodeString(parts[1])
	if err != nil {
		return nil, ErrInvalidToken
	}

	var payload jwtPayload
	if err := json.Unmarshal(payloadRaw, &payload); err != nil {
		return nil, ErrInvalidToken
	}
	if payload.Sub == "" {
		return nil, ErrInvalidToken
	}
	if payload.Exp > 0 && time.Now().Unix() >= payload.Exp {
		return nil, ErrInvalidToken
	}

	return &User{
		ID:        payload.Sub,
		Role:      payload.Role,
		Roles:     payload.Roles,
		Scopes:    payload.Scopes,
		Username:  payload.Username,
		ExpiresAt: payload.Exp,
		Metadata:  payload.Metadata,
	}, nil
}
