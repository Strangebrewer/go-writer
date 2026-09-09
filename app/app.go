package app

import (
	"github.com/Strangebrewer/go-writer/tracer"
	"github.com/Strangebrewer/go-writer/writer"
)

type Application struct {
	ProjectStore *writer.ProjectStore
	SubjectStore *writer.SubjectStore
	TextStore    *writer.TextStore
	Tracer       *tracer.Client
}
