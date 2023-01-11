package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/jaeger"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
)

// Package-level tracer.
// This should be configured in your code setup instead of here.
var tracer = otel.Tracer("github.com/cndoit18/observability-damo")

// sleepy mocks work that your application does.
func sleepy(ctx context.Context) {
	_, span := tracer.Start(ctx, "sleep")
	defer span.End()

	sleepTime := 1 * time.Second
	time.Sleep(sleepTime)
	span.SetAttributes(attribute.Int("sleep.duration", int(sleepTime)))
}

// httpHandler is an HTTP handler function that is going to be instrumented.
func httpHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Hello, World! I am instrumented automatically!")
	ctx := r.Context()
	sleepy(ctx)
}

func main() {
	l := log.New(os.Stdout, "", 0)

	exp, err := jaeger.New(jaeger.WithCollectorEndpoint())
	if err != nil {
		l.Fatal(err)
	}

	tp := trace.NewTracerProvider(
		trace.WithBatcher(exp),
		trace.WithResource(newResource()),
	)
	defer func() {
		if err := tp.Shutdown(context.Background()); err != nil {
			l.Fatal(err)
		}
	}()
	otel.SetTracerProvider(tp)

	// Wrap your httpHandler function.
	handler := http.HandlerFunc(httpHandler)
	wrappedHandler := otelhttp.NewHandler(handler, "hello-instrumented")
	http.Handle("/hello-instrumented", wrappedHandler)

	// And start the HTTP serve.
	log.Fatal(http.ListenAndServe(":3030", nil))
}

func newResource() *resource.Resource {
	r, _ := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceNameKey.String("http"),
			semconv.ServiceVersionKey.String("v0.1.0"),
			attribute.String("environment", "demo"),
		),
	)
	return r
}
