package httpapi

import "time"

// CreateLinkRequest описывает JSON-запрос на создание короткой ссылки.
type CreateLinkRequest struct {
	// URL - исходный адрес, который нужно сократить.
	URL string `json:"url"`
	// ExpiresAt - необязательная дата и время истечения ссылки.
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
}

// CreateLinkResponse описывает JSON-ответ с созданным коротким кодом.
type CreateLinkResponse struct {
	// ShortCode - короткий код, который клиент добавляет к базовому адресу сервиса.
	ShortCode string `json:"short_code"`
}
