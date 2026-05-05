package rediscache

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/n1ckerr0r/shortener/internal/link"
)

const defaultPrefix = "shortener:link:"

type Options struct {
	Addr     string
	Username string
	Password string
	DB       int
	Prefix   string
}

type Cache struct {
	client *redis.Client
	prefix string
}

func Open(ctx context.Context, opts Options) (*Cache, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     opts.Addr,
		Username: opts.Username,
		Password: opts.Password,
		DB:       opts.DB,
	})

	cache := NewCache(client, opts.Prefix)
	if err := cache.Ping(ctx); err != nil {
		_ = client.Close()
		return nil, err
	}

	return cache, nil
}

func NewCache(client *redis.Client, prefix string) *Cache {
	if prefix == "" {
		prefix = defaultPrefix
	}

	return &Cache{
		client: client,
		prefix: prefix,
	}
}

func (c *Cache) Ping(ctx context.Context) error {
	return c.client.Ping(ctx).Err()
}

func (c *Cache) Close() error {
	return c.client.Close()
}

func (c *Cache) Get(ctx context.Context, code link.ShortCode) (link.OriginalURL, bool, error) {
	value, err := c.client.Get(ctx, c.key(code)).Result()
	if errors.Is(err, redis.Nil) {
		return link.OriginalURL{}, false, nil
	}
	if err != nil {
		return link.OriginalURL{}, false, err
	}

	originalURL, err := link.NewOriginalURL(value)
	if err != nil {
		_ = c.client.Del(ctx, c.key(code)).Err()
		return link.OriginalURL{}, false, nil
	}

	return originalURL, true, nil
}

func (c *Cache) Set(ctx context.Context, code link.ShortCode, originalURL link.OriginalURL, ttl time.Duration) error {
	if ttl <= 0 {
		return nil
	}

	return c.client.Set(ctx, c.key(code), originalURL.Value(), ttl).Err()
}

func (c *Cache) key(code link.ShortCode) string {
	return fmt.Sprintf("%s%s", c.prefix, code.Value())
}
