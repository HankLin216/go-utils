package tracer

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

// Config holds the configuration for tracing.
type Config struct {
	Enable   bool
	Endpoint string
}

// Info holds the service information for tracing resource.
type Info struct {
	Name    string
	Version string
	Env     string
}

// NewTracerProvider initializes and returns a new TracerProvider.
// It also sets the global TracerProvider and TextMapPropagator as a side effect (legacy/global usage).
func NewTracerProvider(c *Config, info *Info) (*sdktrace.TracerProvider, func(), error) {
	if !c.Enable {
		// Return a no-op cleanup if disabled
		return nil, func() {}, nil
	}

	ctx := context.Background()
	exp, err := otlptracegrpc.New(ctx,
		otlptracegrpc.WithEndpoint(c.Endpoint),
		otlptracegrpc.WithInsecure(),
	)
	if err != nil {
		return nil, nil, err
	}

	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceNameKey.String(info.Name),
			semconv.ServiceVersionKey.String(info.Version),
			semconv.DeploymentEnvironmentKey.String(info.Env),
		),
	)
	if err != nil {
		return nil, nil, err
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exp),
		sdktrace.WithResource(res),
	)

	// Set global provider
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))

	cleanup := func() {
		if err := tp.Shutdown(context.Background()); err != nil {
			// In a real app, you might want to log this error using a logger passed in dependencies
		}
	}

	return tp, cleanup, nil
}
