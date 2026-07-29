package memory

import (
	"context"
	"sync"

	"github.com/n1ckerr0r/shortener/internal/link"
)

type Repository struct {
	mu    sync.RWMutex
	links map[string]*link.ShortLink
}

func NewRepository() *Repository {
	return &Repository{
		links: make(map[string]*link.ShortLink),
	}
}

func (r *Repository) Save(ctx context.Context, shortLink *link.ShortLink) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.links[shortLink.ShortCode().Value()]; exists {
		return link.ErrShortCodeAlreadyExists
	}

	r.links[shortLink.ShortCode().Value()] = shortLink
	return nil
}

func (r *Repository) Find(ctx context.Context, code link.ShortCode) (*link.ShortLink, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	shortLink, ok := r.links[code.Value()]
	if !ok {
		return nil, link.ErrNotFound
	}

	return shortLink, nil
}

func (r *Repository) Exists(ctx context.Context, code link.ShortCode) (bool, error) {
	if err := ctx.Err(); err != nil {
		return false, err
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	_, ok := r.links[code.Value()]
	return ok, nil
}
