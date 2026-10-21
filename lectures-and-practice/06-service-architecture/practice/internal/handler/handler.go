package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/course-go-autumn-2026/practice6-architecture-refactor/internal/domain"
	"github.com/course-go-autumn-2026/practice6-architecture-refactor/internal/service"
)

type Handler struct {
	service *service.ListingService
}

func NewHandler(s *service.ListingService) *Handler {
	return &Handler{service: s}
}

func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("POST /listings", h.CreateListing)
	mux.HandleFunc("GET /listings/{id}", h.GetListing)
	mux.HandleFunc("PATCH /listings/{id}/price", h.UpdatePrice)
}

type createListingRequest struct {
	Title    string  `json:"title"`
	Price    float64 `json:"price"`
	PhotoURL string  `json:"photo_url"`
	SellerID int64   `json:"seller_id"`
}

type updatePriceRequest struct {
	Price float64 `json:"price"`
}

type listingResponse struct {
	ID        int64     `json:"ID"`
	Title     string    `json:"Title"`
	Price     float64   `json:"Price"`
	PhotoURL  string    `json:"PhotoURL"`
	SellerID  int64     `json:"SellerID"`
	Status    string    `json:"Status"`
	CreatedAt time.Time `json:"CreatedAt"`
}

func toResponse(l *domain.Listing) listingResponse {
	s := l.Snapshot()
	return listingResponse{
		ID:        s.ID,
		Title:     s.Title,
		Price:     s.Price,
		PhotoURL:  s.PhotoURL,
		SellerID:  s.SellerID,
		Status:    string(s.Status),
		CreatedAt: s.CreatedAt,
	}
}

func (h *Handler) CreateListing(w http.ResponseWriter, r *http.Request) {
	var req createListingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	listing, err := h.service.Create(r.Context(), service.CreateInput{
		Title:    req.Title,
		Price:    req.Price,
		PhotoURL: req.PhotoURL,
		SellerID: req.SellerID,
	})
	if err != nil {
		writeError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]any{
		"id":     listing.Snapshot().ID,
		"status": listing.Snapshot().Status,
	})
}

func (h *Handler) GetListing(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	listing, err := h.service.Get(r.Context(), id)
	if err != nil {
		writeError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(toResponse(listing))
}

func (h *Handler) UpdatePrice(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	var req updatePriceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	listing, err := h.service.UpdatePrice(r.Context(), id, req.Price)
	if err != nil {
		writeError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(toResponse(listing))
}

func writeError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domain.ErrNotFound):
		w.WriteHeader(http.StatusNotFound)
	case errors.Is(err, domain.ErrInvalidPrice),
		errors.Is(err, domain.ErrPhotoMissing),
		errors.Is(err, domain.ErrSuspiciousPrice):
		http.Error(w, err.Error(), http.StatusBadRequest)
	case errors.Is(err, domain.ErrRejected):
		http.Error(w, err.Error(), http.StatusConflict)
	default:
		http.Error(w, "internal error", http.StatusInternalServerError)
	}
}
