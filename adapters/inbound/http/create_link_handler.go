package http

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/n1ckerr0r/shortener/core/application/create_link"
)

type CreateLinkHandler struct {
	service *create_link.Service
}

func NewCreateLinkHandler(service *create_link.Service) *CreateLinkHandler {
	return &CreateLinkHandler{
		service: service,
	}
}

func (h *CreateLinkHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var req CreateLinkRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	appReq := create_link.Request{
		OriginalURL: req.URL,
		ExpiresAt:   req.ExpiresAt,
	}

	resp, err := h.service.CreateShortLink(appReq)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	out := CreateLinkResponse{
		ShortCode: resp.ShortCode,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(out); err != nil {
		log.Fatal(err)
	}
}
