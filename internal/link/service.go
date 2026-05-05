package link

import (
	"context"
	"time"
)

const generateAttempts = 5
const defaultResolveCacheTTL = 24 * time.Hour

type Service struct {
	repository Repository
	generator  CodeGenerator
	clock      Clock
	cache      ResolveCache
	cacheTTL   time.Duration
}

func NewService(repository Repository, generator CodeGenerator, clock Clock) *Service {
	return &Service{
		repository: repository,
		generator:  generator,
		clock:      clock,
		cacheTTL:   defaultResolveCacheTTL,
	}
}

func NewServiceWithCache(
	repository Repository,
	generator CodeGenerator,
	clock Clock,
	cache ResolveCache,
	cacheTTL time.Duration,
) *Service {
	service := NewService(repository, generator, clock)
	service.cache = cache
	if cacheTTL > 0 {
		service.cacheTTL = cacheTTL
	}

	return service
}

// CreateRequest описывает входные данные API для создания короткой ссылки.
type CreateRequest struct {
	// OriginalURL - исходный адрес, на который будет вести короткая ссылка.
	OriginalURL string
	// ExpiresAt - необязательное время истечения срока действия ссылки.
	ExpiresAt *time.Time
}

// CreateResponse описывает ответ API после создания короткой ссылки.
type CreateResponse struct {
	// ShortCode - код, который используется в коротком URL.
	ShortCode string
}

// ResolveRequest описывает входные данные API для раскрытия короткой ссылки.
type ResolveRequest struct {
	// Code - короткий код ссылки.
	Code string
}

// ResolveResponse описывает ответ API с исходным адресом ссылки.
type ResolveResponse struct {
	// OriginalURL - исходный URL, на который нужно перенаправить пользователя.
	OriginalURL string
}

func (s *Service) Create(ctx context.Context, request CreateRequest) (*CreateResponse, error) {
	originalURL, err := NewOriginalURL(request.OriginalURL)
	if err != nil {
		return nil, err
	}

	expiration := NewExpiration(request.ExpiresAt)
	now := s.clock.Now()

	for attempt := 0; attempt < generateAttempts; attempt++ {
		code, err := s.generator.Generate()
		if err != nil {
			return nil, err
		}

		exists, err := s.repository.Exists(ctx, code)
		if err != nil {
			return nil, err
		}
		if exists {
			continue
		}

		shortLink, err := NewShortLink(code, originalURL, now, expiration)
		if err != nil {
			return nil, err
		}

		if err = s.repository.Save(ctx, shortLink); err != nil {
			return nil, err
		}

		return &CreateResponse{
			ShortCode: code.Value(),
		}, nil
	}

	return nil, ErrShortCodeCollision
}

func (s *Service) Resolve(ctx context.Context, req ResolveRequest) (*ResolveResponse, error) {
	code, err := NewShortCode(req.Code)
	if err != nil {
		return nil, err
	}

	if s.cache != nil {
		originalURL, ok, err := s.cache.Get(ctx, code)
		if err == nil && ok {
			return &ResolveResponse{
				OriginalURL: originalURL.Value(),
			}, nil
		}
	}

	shortLink, err := s.repository.Find(ctx, code)
	if err != nil {
		return nil, err
	}

	now := s.clock.Now()
	if shortLink.IsExpired(now) {
		return nil, ErrExpiredLink
	}

	if shortLink.IsBlocked() {
		return nil, ErrBlockedLink
	}

	s.cacheResolvedLink(ctx, shortLink, now)

	return &ResolveResponse{
		OriginalURL: shortLink.OriginalURL().Value(),
	}, nil
}

func (s *Service) cacheResolvedLink(ctx context.Context, shortLink *ShortLink, now time.Time) {
	if s.cache == nil {
		return
	}

	ttl := s.resolveCacheTTL(shortLink, now)
	if ttl <= 0 {
		return
	}

	_ = s.cache.Set(ctx, shortLink.ShortCode(), shortLink.OriginalURL(), ttl)
}

func (s *Service) resolveCacheTTL(shortLink *ShortLink, now time.Time) time.Duration {
	ttl := s.cacheTTL
	if ttl <= 0 {
		ttl = defaultResolveCacheTTL
	}

	expiresAt := shortLink.ExpiresAt()
	if expiresAt == nil {
		return ttl
	}

	untilExpiration := expiresAt.Sub(now)
	if untilExpiration <= 0 {
		return 0
	}
	if untilExpiration < ttl {
		return untilExpiration
	}

	return ttl
}
