package link

import (
	"context"
	"errors"
	"time"
)

const generateAttempts = 5
const defaultResolveCacheTTL = 24 * time.Hour

type Service struct {
	repository        Repository
	generator         CodeGenerator
	clock             Clock
	cache             ResolveCache
	cacheTTL          time.Duration
	clickPublisher    ClickPublisher
	urlCheckPublisher URLCheckPublisher
}

func NewService(repository Repository, generator CodeGenerator, clock Clock) *Service {
	return &Service{
		repository: repository,
		generator:  generator,
		clock:      clock,
		cacheTTL:   defaultResolveCacheTTL,
	}
}

type ServiceOption func(*Service)

func WithResolveCache(cache ResolveCache, cacheTTL time.Duration) ServiceOption {
	return func(service *Service) {
		service.cache = cache
		if cacheTTL > 0 {
			service.cacheTTL = cacheTTL
		}
	}
}

func WithClickPublisher(publisher ClickPublisher) ServiceOption {
	return func(service *Service) {
		service.clickPublisher = publisher
	}
}

func WithURLCheckPublisher(publisher URLCheckPublisher) ServiceOption {
	return func(service *Service) {
		service.urlCheckPublisher = publisher
	}
}

func NewServiceWithOptions(
	repository Repository,
	generator CodeGenerator,
	clock Clock,
	options ...ServiceOption,
) *Service {
	service := NewService(repository, generator, clock)
	for _, option := range options {
		option(service)
	}

	return service
}

func NewServiceWithCache(
	repository Repository,
	generator CodeGenerator,
	clock Clock,
	cache ResolveCache,
	cacheTTL time.Duration,
) *Service {
	return NewServiceWithOptions(repository, generator, clock, WithResolveCache(cache, cacheTTL))
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
	// RemoteAddr - адрес клиента для аналитики переходов.
	RemoteAddr string
	// UserAgent - user agent клиента для аналитики переходов.
	UserAgent string
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
			if errors.Is(err, ErrShortCodeAlreadyExists) {
				continue
			}
			return nil, err
		}

		s.publishURLCheck(ctx, shortLink, now)

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
	s.publishClick(ctx, shortLink, now, req)

	return &ResolveResponse{
		OriginalURL: shortLink.OriginalURL().Value(),
	}, nil
}

func (s *Service) publishURLCheck(ctx context.Context, shortLink *ShortLink, createdAt time.Time) {
	if s.urlCheckPublisher == nil {
		return
	}

	_ = s.urlCheckPublisher.PublishURLCheck(ctx, URLCheckJob{
		Code:        shortLink.ShortCode().Value(),
		OriginalURL: shortLink.OriginalURL().Value(),
		CreatedAt:   createdAt,
	})
}

func (s *Service) publishClick(ctx context.Context, shortLink *ShortLink, clickedAt time.Time, req ResolveRequest) {
	if s.clickPublisher == nil {
		return
	}

	_ = s.clickPublisher.PublishClick(ctx, ClickEvent{
		Code:        shortLink.ShortCode().Value(),
		OriginalURL: shortLink.OriginalURL().Value(),
		ClickedAt:   clickedAt,
		RemoteAddr:  req.RemoteAddr,
		UserAgent:   req.UserAgent,
	})
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
