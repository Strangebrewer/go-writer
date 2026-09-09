package writer

import (
	"github.com/Strangebrewer/go-writer/tracer"
	"github.com/go-chi/chi/v5"
)

func TextRoutes(textStore *TextStore, tc *tracer.Client) chi.Router {
	r := chi.NewRouter()
	h := NewTextHandler(textStore)

	r.Get("/{id}", h.GetOne)
	r.Post("/", h.Create)
	r.Put("/{id}", h.Update)
	r.Delete("/{id}", h.Delete)

	return r
}
