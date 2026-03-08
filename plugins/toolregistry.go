package plugins

import (
	"context"
	"slices"

	"github.com/bycmd/GoRouter-kit/client"
	"github.com/bycmd/GoRouter-kit/schema"
)

type ToolRegistryMode string

const (
	ToolRegistryOverride ToolRegistryMode = "override"
	ToolRegistryMerge    ToolRegistryMode = "merge"
	ToolRegistryIfEmpty  ToolRegistryMode = "if-empty"
)

type ToolRegistryOptions struct {
	Mode ToolRegistryMode
}

type ToolProvider func(ctx context.Context) ([]schema.Tool, error)

type ToolRegistryPlugin struct {
	provider ToolProvider
	mode     ToolRegistryMode
}

func NewToolRegistryPlugin(tools []schema.Tool, opts ToolRegistryOptions) *ToolRegistryPlugin {
	staticTools := cloneTools(tools)
	return NewDynamicToolRegistryPlugin(func(ctx context.Context) ([]schema.Tool, error) {
		return cloneTools(staticTools), nil
	}, opts)
}

func NewDynamicToolRegistryPlugin(provider ToolProvider, opts ToolRegistryOptions) *ToolRegistryPlugin {
	mode := opts.Mode
	if mode == "" {
		mode = ToolRegistryOverride
	}

	return &ToolRegistryPlugin{
		provider: provider,
		mode:     mode,
	}
}

func (p *ToolRegistryPlugin) Init(c *client.Client) error {
	if p.provider == nil {
		return nil
	}

	c.Use(func(ctx context.Context, m *client.MiddlewareContext, next client.Next) error {
		injected, err := p.provider(ctx)
		if err != nil {
			return err
		}

		m.Request.Tools = p.apply(m.Request.Tools, injected)
		return next(ctx)
	})
	return nil
}

func (p *ToolRegistryPlugin) Destroy() error {
	return nil
}

func (p *ToolRegistryPlugin) apply(current, injected []schema.Tool) []schema.Tool {
	base := cloneTools(current)
	added := cloneTools(injected)

	switch p.mode {
	case ToolRegistryIfEmpty:
		if len(base) > 0 {
			return base
		}
		return added
	case ToolRegistryMerge:
		if len(added) == 0 {
			return base
		}

		names := make([]string, 0, len(base))
		for _, item := range base {
			names = append(names, toolName(item))
		}
		for _, item := range added {
			name := toolName(item)
			if name != "" && slices.Contains(names, name) {
				continue
			}
			base = append(base, item)
			names = append(names, name)
		}
		return base
	default:
		return added
	}
}

func toolName(item schema.Tool) string {
	if item.Function.Name != "" {
		return item.Function.Name
	}
	return item.Name
}

func cloneTools(in []schema.Tool) []schema.Tool {
	if len(in) == 0 {
		return nil
	}
	out := make([]schema.Tool, len(in))
	copy(out, in)
	return out
}
