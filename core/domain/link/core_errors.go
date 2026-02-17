package link

import "errors"

var (
	ErrInvalidURL            = errors.New("invalid original url")
	ErrEmptyShortCode        = errors.New("short code is empty")
	ErrExpiredLink           = errors.New("link is expired")
	ErrBlockedLink           = errors.New("link is blocked")
	ErrInvalidExpirationDate = errors.New("expiration date is invalid")
)
