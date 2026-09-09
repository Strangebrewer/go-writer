package writer

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/Strangebrewer/go-writer/middleware"
	"github.com/Strangebrewer/go-writer/utils/extraction"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type TextHandler struct {
	textStore *TextStore
}

func NewTextHandler(textStore *TextStore) *TextHandler {
	return &TextHandler{textStore: textStore}
}

func (h *TextHandler) GetOne(w http.ResponseWriter, r *http.Request) {
	userId, err := extraction.UserIDFromRequest(r)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	text, err := h.textStore.GetByID(r.Context(), id, userId)
	if err != nil {
		if errors.Is(err, ErrTextNotFound) {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		slog.Error("get text", "id", id, "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(text)
}

func (h *TextHandler) Create(w http.ResponseWriter, r *http.Request) {
	userId, err := extraction.UserIDFromRequest(r)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var req CreateTextRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}

	if req.Title == "" {
		http.Error(w, "title is required", http.StatusBadRequest)
		return
	}

	if req.Content == "" {
		http.Error(w, "text is required", http.StatusBadRequest)
		return
	}

	if req.ProjectID == "" {
		http.Error(w, "projectId is required", http.StatusBadRequest)
		return
	}

	if req.SubjectID == "" {
		http.Error(w, "subjectId is required", http.StatusBadRequest)
		return
	}

	if middleware.IsDemoFromContext(r.Context()) {
		count, err := h.textStore.CountByUser(r.Context(), userId)
		if err != nil {
			slog.Error("count texts", "error", err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}
		if count >= 20 {
			http.Error(w, "demo text limit reached", http.StatusForbidden)
			return
		}
	}

	expiresAt := middleware.ExpiresAtFromContext(r.Context())

	created, err := h.textStore.Create(r.Context(), userId, req, expiresAt)
	if err != nil {
		if errors.Is(err, ErrSubjectNotFound) {
			http.Error(w, "subject not found", http.StatusBadRequest)
			return
		}
		slog.Error("create text", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(created)
}

func (h *TextHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	userId, err := extraction.UserIDFromRequest(r)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var req UpdateTextRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	text, err := h.textStore.Update(r.Context(), id, userId, req)
	if err != nil {
		if errors.Is(err, ErrTextNotFound) {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		slog.Error("update text", "id", id, "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(text)
}

func (h *TextHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	userId, err := extraction.UserIDFromRequest(r)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	if err := h.textStore.Delete(r.Context(), id, userId); err != nil {
		if errors.Is(err, ErrTextNotFound) {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		slog.Error("delete text", "id", id, "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
