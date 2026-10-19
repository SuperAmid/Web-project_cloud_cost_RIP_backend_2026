// Package redis — клиент Redis для blacklist JWT: после выхода токен
// хранится в Redis до конца срока действия и больше не принимается.
package redis

import (
	"context"
	"errors"
	"fmt"
	"time"

	goredis "github.com/redis/go-redis/v9"
)

// Префикс ключей: в одном Redis могут жить данные разных сервисов.
const jwtBlacklistPrefix = "provider_routers:jwt:blacklist:"

type Client struct {
	client *goredis.Client
}

func New(ctx context.Context, addr, password string) (*Client, error) {
	client := goredis.NewClient(&goredis.Options{Addr: addr, Password: password})
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("Redis %s недоступен: %w", addr, err)
	}
	return &Client{client: client}, nil
}

// WriteJWTToBlacklist кладёт токен в blacklist на ttl; значение — email
// пользователя, чтобы в redis-cli было видно, чей это выход.
func (c *Client) WriteJWTToBlacklist(ctx context.Context, token, email string, ttl time.Duration) error {
	return c.client.Set(ctx, jwtBlacklistPrefix+token, email, ttl).Err()
}

// IsJWTBlacklisted — true, если пользователь уже вышел с этим токеном.
func (c *Client) IsJWTBlacklisted(ctx context.Context, token string) (bool, error) {
	err := c.client.Get(ctx, jwtBlacklistPrefix+token).Err()
	if errors.Is(err, goredis.Nil) {
		return false, nil
	}
	return err == nil, err
}
