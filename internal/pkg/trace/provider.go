package trace

import (
	"context"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	sdkresource "go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.12.0"
	"go.opentelemetry.io/otel/trace"
)

// ProviderConfig represents the provider configuration and used to create a new
// `Provider` type.
type ProviderConfig struct {
	OtelEndpoint   string
	ServiceName    string
	ServiceVersion string
	Environment    string
	// Set this to `true` if you want to disable tracing completly.
	Disabled bool
}

// Provider represents the tracer provider. Depending on the `config.Disabled`
// parameter, it will either use a "live" provider or a "no operations" version.
// The "no operations" means, tracing will be globally disabled.
type Provider struct {
	provider trace.TracerProvider
}

// New returns a new `Provider` type. It uses Jaeger exporter and globally sets
// the tracer provider as well as the global tracer for spans.
func NewProvider(config ProviderConfig) (Provider, error) {
	if config.Disabled {
		return Provider{provider: trace.NewNoopTracerProvider()}, nil
	}

	ctx := context.Background()

	//Resources
	res, err := sdkresource.New(ctx,
		sdkresource.WithAttributes(
			// the service name used to display traces in backends
			semconv.ServiceNameKey.String("test-service"),
		),
	)
	if err != nil {
		return Provider{}, err
	}

	//connection
	ctx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()
	conn, err := grpc.DialContext(ctx, config.OtelEndpoint, grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithBlock())
	if err != nil {
		return Provider{}, err
	}

	exp, err := otlptracegrpc.New(ctx, otlptracegrpc.WithGRPCConn(conn))

	if err != nil {
		return Provider{}, err
	}

	bsp := sdktrace.NewBatchSpanProcessor(exp)
	prv := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
		sdktrace.WithResource(res),
		sdktrace.WithSpanProcessor(bsp),
	)

	otel.SetTracerProvider(prv)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	return Provider{provider: prv}, nil
}

// Close shuts down the tracer provider only if it was not "no operations"
// version.
func (p Provider) Close(ctx context.Context) error {
	if prv, ok := p.provider.(*sdktrace.TracerProvider); ok {
		return prv.Shutdown(ctx)
	}

	return nil
}
