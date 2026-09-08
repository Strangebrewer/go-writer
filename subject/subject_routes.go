package subject

import (
	"github.com/Strangebrewer/go-writer/text"
	"github.com/Strangebrewer/go-writer/tracer"
	"github.com/go-chi/chi/v5"
)

func Routes(store *Store, textStore *text.Store, tc *tracer.Client) chi.Router {
	r := chi.NewRouter()
	h := NewHandler(store, textStore)

	r.Get("/{id}", h.GetOne) // fetches texts
	r.Post("/", h.Create)
	r.Put("/{id}", h.Update)
	r.Delete("/{id}", h.Delete)

	return r
}
