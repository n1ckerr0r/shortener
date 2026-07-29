package link

import "time"

type ClickEvent struct {
	Code        string    `json:"code" bson:"code"`
	OriginalURL string    `json:"original_url" bson:"original_url"`
	ClickedAt   time.Time `json:"clicked_at" bson:"clicked_at"`
	RemoteAddr  string    `json:"remote_addr,omitempty" bson:"remote_addr,omitempty"`
	UserAgent   string    `json:"user_agent,omitempty" bson:"user_agent,omitempty"`
}

type URLCheckJob struct {
	Code        string    `json:"code"`
	OriginalURL string    `json:"original_url"`
	CreatedAt   time.Time `json:"created_at"`
}
