package link

import (
	"time"
)

type ShortLink struct {
	shortCode   ShortCode
	originalURL OriginalURL
	createAt    time.Time
	expiration  Expiration
	blocked     bool
}

func NewShortLink(
	shortCode ShortCode,
	originalURL OriginalURL,
	createAt time.Time,
	expiration Expiration,
) (*ShortLink, error) {

	if expiration.ExpiresAt() != nil &&
		expiration.ExpiresAt().Before(createAt) {

		return nil, ErrInvalidExpirationDate
	}

	return &ShortLink{
		shortCode:   shortCode,
		originalURL: originalURL,
		createAt:    time.Now(),
		expiration:  expiration,
		blocked:     false,
	}, nil
}

// Логика
func (s *ShortLink) IsExpired(now time.Time) bool {
	return s.expiration.IsExpired(now)
}

func (s *ShortLink) IsActive(now time.Time) bool {
	if s.blocked {
		return false
	}

	if s.IsExpired(now) {
		return false
	}

	return true
}

func (s *ShortLink) Block() {
	s.blocked = true
}

func (s *ShortLink) Unblock() {
	s.blocked = false
}

// Геттеры
func (s *ShortLink) ShortCode() ShortCode {
	return s.shortCode
}

func (s *ShortLink) OriginalURL() OriginalURL {
	return s.originalURL
}

func (s *ShortLink) CreateAt() time.Time {
	return s.createAt
}

func (s *ShortLink) ExpiresAt() time.Time {
	return *s.expiration.expiresAt
}

func (s *ShortLink) IsBlocked() bool {
	return s.blocked
}
