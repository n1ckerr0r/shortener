package link

import (
	"time"
)

type Expiration struct {
	expiresAt *time.Time
}

func NewExpiration(expiresAt *time.Time) Expiration {
	return Expiration{expiresAt: expiresAt}
}

func (e Expiration) IsExpired(now time.Time) bool {
	if e.expiresAt == nil {
		return false
	}

	return now.After(*e.expiresAt)
}

func (e Expiration) ExpiresAt() *time.Time {
	return e.expiresAt
}
