package http

import "time"

type CreateLinkRequest struct {
	URL       string     `json:"url"`
	ExpiresAt *time.Time `json:"expires_at"`
}
