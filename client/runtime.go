package client

import (
	"github.com/bycmd/GoRouter-kit/history"
	"github.com/bycmd/GoRouter-kit/security"
)

type destroyable interface {
	Destroy()
}

func (c *Client) On(event string, fn EventHandler) int {
	return c.events.on(event, fn)
}

func (c *Client) Off(event string, id int) {
	c.events.off(event, id)
}

func (c *Client) SetSecurityGuard(guard security.Guard) {
	c.cfg.Security = guard
}

func (c *Client) SecurityGuard() security.Guard {
	return c.cfg.Security
}

func (c *Client) SetHistoryStore(store history.Store) {
	c.cfg.History = store
}

func (c *Client) HistoryStore() history.Store {
	return c.cfg.History
}

func (c *Client) Destroy() error {
	c.StopPriceAutoRefresh()

	pluginErr := c.DestroyPlugins()

	if d, ok := c.cfg.History.(destroyable); ok {
		d.Destroy()
	}
	if d, ok := c.cfg.Security.(destroyable); ok {
		d.Destroy()
	}

	return pluginErr
}
