package http

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	nethttp "net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/iamseth/tiny-headend/internal/service"
)

type ContentHandler struct {
	svc *service.ContentService
}

func NewContentHandler(svc *service.ContentService) *ContentHandler {
	return &ContentHandler{svc: svc}
}

type contentReq struct {
	Title  string  `json:"title"`
	Size   int64   `json:"size"`
	Length float64 `json:"length"`
	Path   string  `json:"path"`
}

const maxContentBodyBytes = 1 << 20

func (h *ContentHandler) Create(w nethttp.ResponseWriter, r *nethttp.Request) {
	var req contentReq
	if !decodeContentRequest(w, r, &req) {
		return
	}

	c := &service.Content{Title: req.Title, Size: req.Size, Length: req.Length, Path: req.Path}
	if err := h.svc.Create(r.Context(), c); err != nil {
		writeErr(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(nethttp.StatusCreated)
	if err := json.NewEncoder(w).Encode(c); err != nil {
		slog.Error("encode create response", "error", err)
	}
}

func (h *ContentHandler) List(w nethttp.ResponseWriter, r *nethttp.Request) {
	limit, offset, ok := parsePagination(w, r)
	if !ok {
		return
	}

	contents, err := h.svc.List(r.Context(), limit, offset)
	if err != nil {
		writeErr(w, err)
		return
	}
	if contents == nil {
		contents = []service.Content{}
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(contents); err != nil {
		slog.Error("encode list response", "error", err)
	}
}

func (h *ContentHandler) Get(w nethttp.ResponseWriter, r *nethttp.Request) {
	id, ok := parseContentID(w, r)
	if !ok {
		return
	}

	c, err := h.svc.Get(r.Context(), id)
	if err != nil {
		writeErr(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(c); err != nil {
		slog.Error("encode get response", "error", err)
	}
}

func (h *ContentHandler) Update(w nethttp.ResponseWriter, r *nethttp.Request) {
	id, ok := parseContentID(w, r)
	if !ok {
		return
	}

	var req contentReq
	if !decodeContentRequest(w, r, &req) {
		return
	}

	c := &service.Content{ID: id, Title: req.Title, Size: req.Size, Length: req.Length, Path: req.Path}
	if err := h.svc.Update(r.Context(), c); err != nil {
		writeErr(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(c); err != nil {
		slog.Error("encode update response", "error", err)
	}
}

func (h *ContentHandler) Delete(w nethttp.ResponseWriter, r *nethttp.Request) {
	id, ok := parseContentID(w, r)
	if !ok {
		return
	}

	if err := h.svc.Delete(r.Context(), id); err != nil {
		writeErr(w, err)
		return
	}

	w.WriteHeader(nethttp.StatusNoContent)
}

func decodeContentRequest(w nethttp.ResponseWriter, r *nethttp.Request, dst *contentReq) bool {
	r.Body = nethttp.MaxBytesReader(w, r.Body, maxContentBodyBytes)
	if err := decodeJSONBody(r, dst); err != nil {
		var maxErr *nethttp.MaxBytesError
		if errors.As(err, &maxErr) {
			nethttp.Error(w, "payload too large", nethttp.StatusRequestEntityTooLarge)
			return false
		}
		nethttp.Error(w, "bad json", nethttp.StatusBadRequest)
		return false
	}
	return true
}

func decodeJSONBody(r *nethttp.Request, dst any) error {
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		return fmt.Errorf("decode json body: %w", err)
	}
	if err := dec.Decode(&struct{}{}); err != io.EOF {
		return errors.New("unexpected trailing json")
	}
	return nil
}

func parseContentID(w nethttp.ResponseWriter, r *nethttp.Request) (uint, bool) {
	id, err := strconv.ParseUint(chi.URLParam(r, "id"), 10, strconv.IntSize)
	if err != nil {
		nethttp.Error(w, "invalid id", nethttp.StatusBadRequest)
		return 0, false
	}
	if id == 0 {
		nethttp.Error(w, "invalid id", nethttp.StatusBadRequest)
		return 0, false
	}
	return uint(id), true
}

func parsePagination(w nethttp.ResponseWriter, r *nethttp.Request) (int, int, bool) {
	const defaultLimit = 100
	const maxLimit = 500
	limit := defaultLimit
	offset := 0

	if rawLimit := r.URL.Query().Get("limit"); rawLimit != "" {
		parsedLimit, err := strconv.Atoi(rawLimit)
		if err != nil || parsedLimit <= 0 || parsedLimit > maxLimit {
			nethttp.Error(w, "invalid limit", nethttp.StatusBadRequest)
			return 0, 0, false
		}
		limit = parsedLimit
	}

	if rawOffset := r.URL.Query().Get("offset"); rawOffset != "" {
		parsedOffset, err := strconv.Atoi(rawOffset)
		if err != nil || parsedOffset < 0 {
			nethttp.Error(w, "invalid offset", nethttp.StatusBadRequest)
			return 0, 0, false
		}
		offset = parsedOffset
	}

	return limit, offset, true
}

func writeErr(w nethttp.ResponseWriter, err error) {
	var ve service.ValidationError
	switch {
	case errors.Is(err, service.ErrNotFound):
		nethttp.Error(w, "not found", nethttp.StatusNotFound)
	case errors.As(err, &ve):
		nethttp.Error(w, ve.Error(), nethttp.StatusBadRequest)
	default:
		nethttp.Error(w, "internal error", nethttp.StatusInternalServerError)
	}
}
