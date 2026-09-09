package writer

import (
	"github.com/Strangebrewer/go-writer/tracer"
	"github.com/go-chi/chi/v5"
)

func ProjectRoutes(projectStore *ProjectStore, subjectStore *SubjectStore, tc *tracer.Client) chi.Router {
	r := chi.NewRouter()
	h := NewProjectHandler(projectStore, subjectStore)

	r.Get("/", h.GetAll) // does not fetch subjects
	r.Post("/", h.Create)
	r.Get("/{id}", h.GetOne) // fetches subjects
	r.Put("/{id}", h.Update)
	r.Delete("/{id}", h.Delete)

	return r
}
