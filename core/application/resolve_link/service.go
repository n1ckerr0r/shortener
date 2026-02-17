package resolve_link

import (
	"github.com/n1ckerr0r/shortener/core/domain/link"
	. "github.com/n1ckerr0r/shortener/ports"
)

type Service struct {
	repo  LinkRepository
	clock Clock
}

func NewService(repo LinkRepository, clock Clock) *Service {
	return &Service{
		repo:  repo,
		clock: clock,
	}
}

func (s *Service) Resolve(req Request) (*Response, error) {
	code, err := link.NewShortCode(req.Code)
	if err != nil {
		return nil, err
	}

	shortLink, err := s.repo.Find(code)
	if err != nil {
		return nil, err
	}

	if shortLink.IsExpired(s.clock.Now()) {
		return nil, link.ErrExpiredLink
	}

	if shortLink.IsBlocked() {
		return nil, link.ErrBlockedLink
	}

	return &Response{
		OriginalURL: shortLink.OriginalURL().Value(),
	}, nil
}
