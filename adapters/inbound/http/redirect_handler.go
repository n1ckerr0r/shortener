package http

import (
	"net/http"
	"strings"

	resolve_link2 "github.com/n1ckerr0r/shortener/core/application/resolve_link"
)

type RedirectHandler struct {
	service *resolve_link2.Service
}

func NewRedirectHandler(service *resolve_link2.Service) *RedirectHandler {
	return &RedirectHandler{service: service}
}

func (h *RedirectHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	code := strings.TrimPrefix(r.URL.Path, "/")
	if code == "" {
		http.NotFound(w, r)
		return
	}

	resp, err := h.service.Resolve(resolve_link2.Request{Code: code})
	if err != nil {
		http.NotFound(w, r)
		return
	}

	http.Redirect(w, r, resp.OriginalURL, http.StatusFound)
}
