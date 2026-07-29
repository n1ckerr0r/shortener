package link

import "time"

type ShortLink struct {
	shortCode   ShortCode
	originalURL OriginalURL
	createdAt   time.Time
	expiration  Expiration
	blocked     bool
}

func NewShortLink(
	shortCode ShortCode,
	originalURL OriginalURL,
	createdAt time.Time,
	expiration Expiration,
) (*ShortLink, error) {
	expiresAt := expiration.ExpiresAt()
	if expiresAt != nil && expiresAt.Before(createdAt) {
		return nil, ErrInvalidExpirationDate
	}

	return &ShortLink{
		shortCode:   shortCode,
		originalURL: originalURL,
		createdAt:   createdAt,
		expiration:  expiration,
		blocked:     false,
	}, nil
}

func (s *ShortLink) IsExpired(now time.Time) bool {
	return s.expiration.IsExpired(now)
}

func (s *ShortLink) IsActive(now time.Time) bool {
	return !s.blocked && !s.IsExpired(now)
}

func (s *ShortLink) Block() {
	s.blocked = true
}

func (s *ShortLink) Unblock() {
	s.blocked = false
}

func (s *ShortLink) ShortCode() ShortCode {
	return s.shortCode
}

func (s *ShortLink) OriginalURL() OriginalURL {
	return s.originalURL
}

func (s *ShortLink) CreatedAt() time.Time {
	return s.createdAt
}

func (s *ShortLink) ExpiresAt() *time.Time {
	return s.expiration.ExpiresAt()
}

func (s *ShortLink) IsBlocked() bool {
	return s.blocked
}
