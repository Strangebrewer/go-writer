package app

import (
	"github.com/Strangebrewer/go-writer/example"
	"github.com/Strangebrewer/go-writer/tracer"
)

type Application struct {
	ExampleStore *example.Store
	Tracer       *tracer.Client
}
