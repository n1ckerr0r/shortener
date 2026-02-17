package create_link

import (
	"github.com/n1ckerr0r/shortener/core/domain/link"
	port "github.com/n1ckerr0r/shortener/ports"
)

type Service struct {
	repository port.LinkRepository
	generator  port.CodeGenerator
	clock      port.Clock
}

func NewService(
	repository port.LinkRepository,
	generator port.CodeGenerator,
	clock port.Clock,
) *Service {
	return &Service{
		repository: repository,
		generator:  generator,
		clock:      clock,
	}
}

func (s *Service) CreateShortLink(request Request) (*Response, error) {
	originalURL, err := link.NewOriginalURL(request.OriginalURL)
	if err != nil {
		return nil, err
	}

	code, err := s.generator.Generate()
	if err != nil {
		return nil, err
	}

	expiration := link.NewExpiration(request.ExpiresAt)
	now := s.clock.Now()

	shortLink, err := link.NewShortLink(
		code,
		originalURL,
		now,
		expiration,
	)
	if err != nil {
		return nil, err
	}

	if err = s.repository.Save(shortLink); err != nil {
		return nil, err
	}

	return &Response{
		ShortCode: code.Value(),
	}, nil
}
