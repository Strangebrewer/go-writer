package app

import (
	"github.com/Strangebrewer/go-service-template/example"
	"github.com/Strangebrewer/go-service-template/tracer"
)

type Application struct {
	ExampleStore *example.Store
	Tracer       *tracer.Client
}
