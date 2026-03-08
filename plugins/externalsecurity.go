package plugins

import (
	"errors"

	"github.com/bycmd/GoRouter-kit/client"
	"github.com/bycmd/GoRouter-kit/security"
)

type GuardFactory func(current security.Guard) security.Guard

type ExternalSecurityPlugin struct {
	guard   security.Guard
	factory GuardFactory
	prev    security.Guard
	client  *client.Client
}

func NewExternalSecurityPlugin(guard security.Guard) *ExternalSecurityPlugin {
	return &ExternalSecurityPlugin{guard: guard}
}

func NewExternalSecurityPluginWithFactory(factory GuardFactory) *ExternalSecurityPlugin {
	return &ExternalSecurityPlugin{factory: factory}
}

func (p *ExternalSecurityPlugin) Init(c *client.Client) error {
	next := p.guard
	if p.factory != nil {
		next = p.factory(c.SecurityGuard())
	}
	if next == nil {
		return errors.New("external security guard is required")
	}

	p.prev = c.SecurityGuard()
	p.client = c
	p.guard = next
	c.SetSecurityGuard(next)
	return nil
}

func (p *ExternalSecurityPlugin) Destroy() error {
	if p.guard != nil {
		if d, ok := p.guard.(interface{ Destroy() }); ok {
			d.Destroy()
		}
	}
	if p.client != nil {
		p.client.SetSecurityGuard(p.prev)
	}
	p.client = nil
	p.prev = nil
	p.guard = nil
	return nil
}
