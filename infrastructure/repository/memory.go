package repository

import (
	"errors"
	"sync"

	"github.com/n1ckerr0r/shortener/core/domain/link"
)

// In-memory вариант хранения
type MemoryRepository struct {
	mu    sync.RWMutex
	links map[string]*link.ShortLink
}

func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{
		links: make(map[string]*link.ShortLink),
	}
}

func (r *MemoryRepository) Save(l *link.ShortLink) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.links[l.ShortCode().Value()] = l
	return nil
}

func (r *MemoryRepository) Find(code link.ShortCode) (*link.ShortLink, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	l, ok := r.links[code.Value()]
	if !ok {
		return nil, errors.New("not found")
	}
	return l, nil
}

func (r *MemoryRepository) Exists(code link.ShortCode) (bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, ok := r.links[code.Value()]
	return ok, nil
}
