package telemetry

import (
	"context"
	"log/slog"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
	"go.opentelemetry.io/otel/trace"
)

var Tracer trace.Tracer

func Init(ctx context.Context, serviceName string) func() {
	exporter, err := stdouttrace.New(
		stdouttrace.WithPrettyPrint(),
	)
	if err != nil {
		slog.Error("failed to create trace exporter", "error", err)
		return func() {}
	}

	res := resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.ServiceName(serviceName),
	)

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
	)

	otel.SetTracerProvider(tp)
	Tracer = otel.Tracer(serviceName)

	// THIS IS THE KEY LINE — registers W3C traceparent encoder/decoder
	otel.SetTextMapPropagator(propagation.TraceContext{})

	slog.Info("tracing initialized", "service", serviceName)

	return func() {
		if err := tp.Shutdown(ctx); err != nil {
			slog.Error("failed to shutdown tracer", "error", err)
		}
	}
}
