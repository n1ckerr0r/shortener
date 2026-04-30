package link

import (
	"context"
	"time"
)

type Repository interface {
	Save(ctx context.Context, shortLink *ShortLink) error
	Find(ctx context.Context, code ShortCode) (*ShortLink, error)
	Exists(ctx context.Context, code ShortCode) (bool, error)
}

type ResolveCache interface {
	Get(ctx context.Context, code ShortCode) (OriginalURL, bool, error)
	Set(ctx context.Context, code ShortCode, originalURL OriginalURL, ttl time.Duration) error
}

type CodeGenerator interface {
	Generate() (ShortCode, error)
}

type Clock interface {
	Now() time.Time
}
