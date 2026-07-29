package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/n1ckerr0r/shortener/internal/link"
)

type LinkCreator interface {
	Create(ctx context.Context, req link.CreateRequest) (*link.CreateResponse, error)
}

type LinkResolver interface {
	Resolve(ctx context.Context, req link.ResolveRequest) (*link.ResolveResponse, error)
}

type CreateLinkHandler struct {
	service LinkCreator
}

func NewCreateLinkHandler(service LinkCreator) *CreateLinkHandler {
	return &CreateLinkHandler{service: service}
}

func (h *CreateLinkHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req CreateLinkRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}
	if err := decoder.Decode(&struct{}{}); err != nil && !errors.Is(err, io.EOF) {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	resp, err := h.service.Create(r.Context(), link.CreateRequest{
		OriginalURL: req.URL,
		ExpiresAt:   req.ExpiresAt,
	})
	if err != nil {
		writeError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, CreateLinkResponse{
		ShortCode: resp.ShortCode,
	})
}

type RedirectHandler struct {
	service LinkResolver
}

func NewRedirectHandler(service LinkResolver) *RedirectHandler {
	return &RedirectHandler{service: service}
}

func (h *RedirectHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Set("Allow", strings.Join([]string{http.MethodGet, http.MethodHead}, ", "))
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	code := strings.TrimPrefix(r.URL.Path, "/")
	if code == "" {
		http.NotFound(w, r)
		return
	}

	resp, err := h.service.Resolve(r.Context(), link.ResolveRequest{
		Code:       code,
		RemoteAddr: r.RemoteAddr,
		UserAgent:  r.UserAgent(),
	})
	if err != nil {
		writeError(w, err)
		return
	}

	http.Redirect(w, r, resp.OriginalURL, http.StatusFound)
}

func writeJSON(w http.ResponseWriter, statusCode int, payload any) {
	var body bytes.Buffer
	if err := json.NewEncoder(&body).Encode(payload); err != nil {
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_, _ = w.Write(body.Bytes())
}

func writeError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, link.ErrInvalidURL),
		errors.Is(err, link.ErrInvalidExpirationDate),
		errors.Is(err, link.ErrEmptyShortCode):
		http.Error(w, err.Error(), http.StatusBadRequest)
	case errors.Is(err, link.ErrNotFound),
		errors.Is(err, link.ErrExpiredLink),
		errors.Is(err, link.ErrBlockedLink):
		http.Error(w, "not found", http.StatusNotFound)
	default:
		http.Error(w, "internal server error", http.StatusInternalServerError)
	}
}
