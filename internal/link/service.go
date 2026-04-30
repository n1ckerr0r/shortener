package link

import (
	"context"
	"time"
)

const generateAttempts = 5

type Service struct {
	repository Repository
	generator  CodeGenerator
	clock      Clock
}

func NewService(repository Repository, generator CodeGenerator, clock Clock) *Service {
	return &Service{
		repository: repository,
		generator:  generator,
		clock:      clock,
	}
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

	shortLink, err := s.repository.Find(ctx, code)
	if err != nil {
		return nil, err
	}

	if shortLink.IsExpired(s.clock.Now()) {
		return nil, ErrExpiredLink
	}

	if shortLink.IsBlocked() {
		return nil, ErrBlockedLink
	}

	return &ResolveResponse{
		OriginalURL: shortLink.OriginalURL().Value(),
	}, nil
}
