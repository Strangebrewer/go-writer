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

type ProjectHandler struct {
	projectStore *ProjectStore
	subjectStore *SubjectStore
}

func NewProjectHandler(projectStore *ProjectStore, subjectStore *SubjectStore) *ProjectHandler {
	return &ProjectHandler{projectStore: projectStore, subjectStore: subjectStore}
}

func (h *ProjectHandler) GetAll(w http.ResponseWriter, r *http.Request) {
	userId, err := extraction.UserIDFromRequest(r)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	projects, err := h.projectStore.GetAll(r.Context(), userId)
	if err != nil {
		slog.Error("get projects", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(projects)
}

func (h *ProjectHandler) GetOne(w http.ResponseWriter, r *http.Request) {
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

	project, err := h.projectStore.GetByID(r.Context(), id, userId)
	if err != nil {
		if errors.Is(err, ErrProjectNotFound) {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		slog.Error("get project", "id", id, "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	subjects, err := h.subjectStore.GetAllByProject(r.Context(), userId, id)
	if err != nil {
		slog.Error("get subjects for project", "project", id, "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	resp := ProjectResponse{Project: project, Subjects: subjects}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

func (h *ProjectHandler) Create(w http.ResponseWriter, r *http.Request) {
	userId, err := extraction.UserIDFromRequest(r)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var req CreateProjectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}

	if req.Title == "" {
		http.Error(w, "title is required", http.StatusBadRequest)
		return
	}

	if middleware.IsDemoFromContext(r.Context()) {
		count, err := h.projectStore.CountByUser(r.Context(), userId)
		if err != nil {
			slog.Error("count projects", "error", err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}
		if count >= 5 {
			http.Error(w, "demo project limit reached", http.StatusForbidden)
			return
		}
	}

	expiresAt := middleware.ExpiresAtFromContext(r.Context())

	created, err := h.projectStore.Create(r.Context(), userId, req, expiresAt)
	if err != nil {
		slog.Error("create project", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(created)
}

func (h *ProjectHandler) Update(w http.ResponseWriter, r *http.Request) {
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

	var req UpdateProjectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	project, err := h.projectStore.Update(r.Context(), id, userID, req)
	if err != nil {
		if errors.Is(err, ErrProjectNotFound) {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		slog.Error("update project", "id", id, "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(project)
}

func (h *ProjectHandler) Delete(w http.ResponseWriter, r *http.Request) {
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

	if err := h.projectStore.Delete(r.Context(), id, userID); err != nil {
		if errors.Is(err, ErrProjectNotFound) {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		slog.Error("delete project", "id", id, "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
