package link

import (
	"net/url"
)

type OriginalURL struct {
	value string
}

func NewOriginalURL(strURL string) (OriginalURL, error) {
	parsed, err := url.ParseRequestURI(strURL)

	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return OriginalURL{}, ErrInvalidURL
	}

	return OriginalURL{value: strURL}, nil
}

func (ou OriginalURL) Value() string {
	return ou.value
}
