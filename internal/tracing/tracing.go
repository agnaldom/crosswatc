// Package tracing configures OpenTelemetry (spec §52). Exporting is enabled by the standard
// OTEL_EXPORTER_OTLP_ENDPOINT / OTEL_EXPORTER_OTLP_TRACES_ENDPOINT variables; otherwise spans are no-ops.
package tracing

import (
	"context"
	"os"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"

	"github.com/agnaldom/crosswatc/internal/version"
)

// Tracer is the Crosswatc tracer.
func Tracer() trace.Tracer { return otel.Tracer("github.com/agnaldom/crosswatc") }

// Setup installs the global tracer provider. The returned shutdown flushes spans.
func Setup(ctx context.Context, service string) (func(context.Context) error, error) {
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))
	if os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT") == "" && os.Getenv("OTEL_EXPORTER_OTLP_TRACES_ENDPOINT") == "" {
		return func(context.Context) error { return nil }, nil
	}
	exp, err := otlptracehttp.New(ctx)
	if err != nil {
		return nil, err
	}
	res, err := resource.Merge(resource.Default(), resource.NewSchemaless(
		attribute.String("service.name", service), attribute.String("service.version", version.Version)))
	if err != nil {
		return nil, err
	}
	tp := sdktrace.NewTracerProvider(sdktrace.WithBatcher(exp), sdktrace.WithResource(res))
	otel.SetTracerProvider(tp)
	return tp.Shutdown, nil
}

// Attribute keys (spec §52 tracing attributes).
var (
	ResourceID    = attribute.Key("resource.id")
	RevisionID    = attribute.Key("revision.id")
	OperationID   = attribute.Key("operation.id")
	CorrelationID = attribute.Key("correlation.id")
	Owner         = attribute.Key("owner")
	Environment   = attribute.Key("environment")
	Provider      = attribute.Key("provider")
	Kind          = attribute.Key("kind")
)
