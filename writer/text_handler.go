package writer

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

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

}

func (h *TextHandler) Update(w http.ResponseWriter, r *http.Request) {

}

func (h *TextHandler) Delete(w http.ResponseWriter, r *http.Request) {

}
