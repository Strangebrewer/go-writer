package app

import (
	"github.com/Strangebrewer/go-writer/project"
	"github.com/Strangebrewer/go-writer/subject"
	"github.com/Strangebrewer/go-writer/text"
	"github.com/Strangebrewer/go-writer/tracer"
)

type Application struct {
	ProjectStore *project.Store
	SubjectStore *subject.Store
	TextStore    *text.Store
	Tracer       *tracer.Client
}
