package example

import (
	"github.com/Strangebrewer/go-writer/tracer"
	"github.com/go-chi/chi/v5"
)

func Routes(store *Store, tc *tracer.Client) chi.Router {
	h := NewHandler(store, tc)
	r := chi.NewRouter()

	r.Get("/", h.GetAll)
	r.Get("/{id}", h.GetOne)
	r.Post("/", h.Create)
	r.Put("/{id}", h.Update)
	r.Delete("/{id}", h.Delete)

	return r
}
