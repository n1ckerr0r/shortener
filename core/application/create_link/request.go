package create_link

import "time"

type Request struct {
	OriginalURL string
	ExpiresAt   *time.Time
}
