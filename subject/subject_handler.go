package subject

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	// "github.com/Strangebrewer/go-writer/middleware"
	"github.com/Strangebrewer/go-writer/text"
	"github.com/Strangebrewer/go-writer/utils/extraction"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type Handler struct {
	store     *Store
	textStore *text.Store
}

func NewHandler(store *Store, textStore *text.Store) *Handler {
	return &Handler{store: store, textStore: textStore}
}

func (h *Handler) GetOne(w http.ResponseWriter, r *http.Request) {
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

	subject, err := h.store.GetByID(r.Context(), id, userId)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		slog.Error("get subject", "id", id, "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	texts, err := h.textStore.GetAllBySubject(r.Context(), userId, id)
	if err != nil {
		slog.Error("get texts for subject", "subject", id, "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	resp := SubjectResponse{Subject: subject, Texts: texts}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	userId, err := extraction.UserIDFromRequest(r)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var req CreateSubjectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}

	if req.Title == "" {
		http.Error(w, "title is required", http.StatusBadRequest)
		return
	}

	if req.ProjectID == "" {
		http.Error(w, "projectId is required", http.StatusBadRequest)
		return
	}

	// if middleware.IsDemoFromContext(r.Context()) {
	// 	count, err := h.store.CountByUser(r.Context(), userID)
	// 	if err != nil {
	// 		slog.Error("count recruiters", "error", err)
	// 		http.Error(w, "internal server error", http.StatusInternalServerError)
	// 		return
	// 	}
	// 	if count >= 5 {
	// 		http.Error(w, "demo recruiter limit reached", http.StatusForbidden)
	// 		return
	// 	}
	// }

	created, err := h.store.Create(r.Context(), userId, req, nil)
	if err != nil {
		slog.Error("create subject", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(created)
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {

}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {

}
