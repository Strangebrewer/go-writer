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

type SubjectHandler struct {
	subjectStore *SubjectStore
	textStore    *TextStore
}

func NewSubjectHandler(subjectStore *SubjectStore, textStore *TextStore) *SubjectHandler {
	return &SubjectHandler{subjectStore: subjectStore, textStore: textStore}
}

func (h *SubjectHandler) GetOne(w http.ResponseWriter, r *http.Request) {
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

	subject, err := h.subjectStore.GetByID(r.Context(), id, userId)
	if err != nil {
		if errors.Is(err, ErrSubjectNotFound) {
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

func (h *SubjectHandler) Create(w http.ResponseWriter, r *http.Request) {
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

	if middleware.IsDemoFromContext(r.Context()) {
		count, err := h.subjectStore.CountByUser(r.Context(), userId)
		if err != nil {
			slog.Error("count subjects", "error", err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}
		if count >= 5 {
			http.Error(w, "demo subject limit reached", http.StatusForbidden)
			return
		}
	}

	expiresAt := middleware.ExpiresAtFromContext(r.Context())

	created, err := h.subjectStore.Create(r.Context(), userId, req, expiresAt)
	if err != nil {
		if errors.Is(err, ErrProjectNotFound) {
			http.Error(w, "project not found", http.StatusBadRequest)
			return
		}
		slog.Error("create subject", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(created)
}

func (h *SubjectHandler) Update(w http.ResponseWriter, r *http.Request) {
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

	var req UpdateSubjectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	subject, err := h.subjectStore.Update(r.Context(), id, userId, req)
	if err != nil {
		if errors.Is(err, ErrSubjectNotFound) {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		slog.Error("update subject", "id", id, "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(subject)
}

func (h *SubjectHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	userID, err := extraction.UserIDFromRequest(r)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	if err := h.subjectStore.Delete(r.Context(), id, userID); err != nil {
		if errors.Is(err, ErrSubjectNotFound) {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		slog.Error("delete subject", "id", id, "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
