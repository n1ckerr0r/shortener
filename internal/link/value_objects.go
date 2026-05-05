package link

import (
	"net/url"
	"strings"
	"time"
)

type OriginalURL struct {
	value string
}

func NewOriginalURL(rawURL string) (OriginalURL, error) {
	rawURL = strings.TrimSpace(rawURL)
	parsed, err := url.ParseRequestURI(rawURL)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return OriginalURL{}, ErrInvalidURL
	}

	return OriginalURL{value: rawURL}, nil
}

func (u OriginalURL) Value() string {
	return u.value
}

type ShortCode struct {
	value string
}

func NewShortCode(code string) (ShortCode, error) {
	code = strings.TrimSpace(code)
	if code == "" {
		return ShortCode{}, ErrEmptyShortCode
	}

	return ShortCode{value: code}, nil
}

func (c ShortCode) Value() string {
	return c.value
}

type Expiration struct {
	expiresAt *time.Time
}

func NewExpiration(expiresAt *time.Time) Expiration {
	if expiresAt == nil {
		return Expiration{}
	}

	value := *expiresAt
	return Expiration{expiresAt: &value}
}

func (e Expiration) IsExpired(now time.Time) bool {
	if e.expiresAt == nil {
		return false
	}

	return now.After(*e.expiresAt)
}

func (e Expiration) ExpiresAt() *time.Time {
	if e.expiresAt == nil {
		return nil
	}

	value := *e.expiresAt
	return &value
}
