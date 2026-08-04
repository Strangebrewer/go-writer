package example

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/Strangebrewer/go-service-template/middleware"
	"github.com/Strangebrewer/go-service-template/tracer"
	"github.com/go-chi/chi/v5"
)

type Handler struct {
	store  *Store
	tracer *tracer.Client
}

func NewHandler(store *Store, tc *tracer.Client) *Handler {
	return &Handler{store: store, tracer: tc}
}

func (h *Handler) GetAll(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	traceID := r.Header.Get("X-Trace-ID")

	start := time.Now()
	examples, err := h.store.GetAll(r.Context(), userID)
	end := time.Now()
	if err != nil {
		slog.Error("example: GetAll", "error", err)
		errMsg := "internal server error"
		h.tracer.SendErrorSpan(traceID, "get_all_examples", errMsg, start, end)
		http.Error(w, errMsg, http.StatusInternalServerError)
		return
	}

	h.tracer.SendSpan(traceID, "get_all_examples", start, end, len(examples))

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(examples)
}

func (h *Handler) GetOne(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	traceID := r.Header.Get("X-Trace-ID")
	op := fmt.Sprintf("get_example by id: %s", id)

	start := time.Now()
	example, err := h.store.GetOne(r.Context(), id, userID)
	end := time.Now()
	if err != nil {
		slog.Error("example: GetOne", "error", err)
		errMsg := "internal server error"
		h.tracer.SendErrorSpan(traceID, op, errMsg, start, end)
		http.Error(w, errMsg, http.StatusInternalServerError)
		return
	}

	h.tracer.SendSpan(traceID, op, start, end)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(example)
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	traceID := r.Header.Get("X-Trace-ID")

	start := time.Now()
	var req CreateExampleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		errMsg := "invalid request body"
		end := time.Now()
		h.tracer.SendErrorSpan(traceID, "create_example", errMsg, start, end)
		http.Error(w, errMsg, http.StatusBadRequest)
		return
	}

	example, err := h.store.Create(r.Context(), userID, req)
	end := time.Now()
	if err != nil {
		slog.Error("example: Create", "error", err)
		errMsg := "internal server error"
		h.tracer.SendErrorSpan(traceID, "create_example", errMsg, start, end)
		http.Error(w, errMsg, http.StatusInternalServerError)
		return
	}

	h.tracer.SendSpan(traceID, "create_example", start, end)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(example)
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	traceID := r.Header.Get("X-Trace-ID")

	start := time.Now()
	id := chi.URLParam(r, "id")
	op := fmt.Sprintf("update_example by id: %s", id)

	var req UpdateExampleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		errMsg := "invalid request body"
		end := time.Now()
		h.tracer.SendErrorSpan(traceID, op, errMsg, start, end)
		http.Error(w, errMsg, http.StatusBadRequest)
		return
	}

	example, err := h.store.Update(r.Context(), id, userID, req)
	end := time.Now()
	if err != nil {
		slog.Error("example: Update", "error", err)
		errMsg := "internal server error"
		h.tracer.SendErrorSpan(traceID, op, errMsg, start, end)
		http.Error(w, errMsg, http.StatusInternalServerError)
		return
	}

	h.tracer.SendSpan(traceID, op, start, end)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(example)
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	traceID := r.Header.Get("X-Trace-ID")

	start := time.Now()
	id := chi.URLParam(r, "id")
	op := fmt.Sprintf("delete_example by id: %s", id)

	if err := h.store.Delete(r.Context(), id, userID); err != nil {
		slog.Error("example: Delete", "error", err)
		errMsg := "internal server error"
		end := time.Now()
		h.tracer.SendErrorSpan(traceID, op, errMsg, start, end)
		http.Error(w, errMsg, http.StatusInternalServerError)
		return
	}

	end := time.Now()
	h.tracer.SendSpan(traceID, op, start, end)

	w.WriteHeader(http.StatusNoContent)
}
