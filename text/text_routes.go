package text

import (
	"github.com/Strangebrewer/go-writer/tracer"
	"github.com/go-chi/chi/v5"
)

func Routes(store *Store, tc *tracer.Client) chi.Router {
	r := chi.NewRouter()
	h := NewHandler(store)

	r.Get("/{id}", h.GetOne)

	return r
}
