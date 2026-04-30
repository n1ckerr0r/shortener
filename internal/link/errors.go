package link

import "errors"

var (
	ErrInvalidURL            = errors.New("invalid original url")
	ErrEmptyShortCode        = errors.New("short code is empty")
	ErrNotFound              = errors.New("link not found")
	ErrExpiredLink           = errors.New("link is expired")
	ErrBlockedLink           = errors.New("link is blocked")
	ErrInvalidExpirationDate = errors.New("expiration date is invalid")
	ErrShortCodeCollision    = errors.New("could not generate unique short code")
)
