package plugins

import (
	"errors"

	"github.com/bycmd/GoRouter-kit/client"
	"github.com/bycmd/GoRouter-kit/history"
	redisstore "github.com/bycmd/GoRouter-kit/history/redis"
)

type RedisHistoryPluginOptions struct {
	Prefix  string
	CloseFn func() error
}

type RedisHistoryPlugin struct {
	store history.Store
}

func NewRedisHistoryPlugin(store history.Store) *RedisHistoryPlugin {
	return &RedisHistoryPlugin{store: store}
}

func NewRedisHistoryPluginFromClient(client redisstore.Client, opts RedisHistoryPluginOptions) *RedisHistoryPlugin {
	store := redisstore.New(client, redisstore.Options{
		Prefix:  opts.Prefix,
		CloseFn: opts.CloseFn,
	})
	return NewRedisHistoryPlugin(store)
}

func (p *RedisHistoryPlugin) Init(c *client.Client) error {
	if p.store == nil {
		return errors.New("redis history store is required")
	}
	c.SetHistoryStore(p.store)
	return nil
}

func (p *RedisHistoryPlugin) Destroy() error {
	return nil
}
