package tracing

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.34.0"
	"go.opentelemetry.io/otel/trace"
)

// Config holds configuration parameters for OpenTelemetry tracing setup.
// This includes service identification, Jaeger endpoint, sampling configuration,
// and batch processing settings for optimal trace delivery.
type Config struct {
	ServiceName    string        `env:"TRACING_SERVICE_NAME" envDefault:"pub-e2e"`
	ServiceVersion string        `env:"TRACING_SERVICE_VERSION" envDefault:"1.0.0"`
	JaegerEndpoint string        `env:"JAEGER_ENDPOINT" envDefault:"localhost:4318"`
	SampleRate     float64       `env:"TRACING_SAMPLE_RATE" envDefault:"1.0"`
	BatchTimeout   time.Duration `env:"TRACING_BATCH_TIMEOUT" envDefault:"1s"`
	ExportTimeout  time.Duration `env:"TRACING_EXPORT_TIMEOUT" envDefault:"30s"`
	MaxExportBatch int           `env:"TRACING_MAX_EXPORT_BATCH" envDefault:"512"`
	MaxQueueSize   int           `env:"TRACING_MAX_QUEUE_SIZE" envDefault:"2048"`
}

// Tracer wraps the OpenTelemetry tracer with convenience methods for pub/sub operations.
// It provides a simplified interface for creating spans, recording errors, and adding
// pub/sub-specific attributes while maintaining the underlying OpenTelemetry functionality.
type Tracer struct {
	tracer trace.Tracer
	config Config
	tp     *sdktrace.TracerProvider
}

// NewTracer creates and configures a new OpenTelemetry tracer with OTLP HTTP export.
// It sets up the tracer provider, configures batch processing for efficient trace delivery,
// and returns both the tracer instance and a cleanup function for graceful shutdown.
func NewTracer(config Config) (*Tracer, func(context.Context) error, error) {
	res, err := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName(config.ServiceName),
			semconv.ServiceVersion(config.ServiceVersion),
			attribute.String("service.environment", "development"),
		),
	)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create resource: %w", err)
	}

	exporter, err := otlptracehttp.New(
		context.Background(),
		otlptracehttp.WithEndpoint(config.JaegerEndpoint),
		otlptracehttp.WithInsecure(), // Use HTTP instead of HTTPS for local development
		otlptracehttp.WithTimeout(config.ExportTimeout),
	)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create OTLP exporter: %w", err)
	}

	// Use batch span processor with shorter timeout for better trace delivery
	processor := sdktrace.NewBatchSpanProcessor(
		exporter,
		sdktrace.WithBatchTimeout(config.BatchTimeout),
		sdktrace.WithExportTimeout(config.ExportTimeout),
		sdktrace.WithMaxExportBatchSize(config.MaxExportBatch),
		sdktrace.WithMaxQueueSize(config.MaxQueueSize),
	)

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithResource(res),
		sdktrace.WithSpanProcessor(processor),
		sdktrace.WithSampler(sdktrace.TraceIDRatioBased(config.SampleRate)),
	)

	otel.SetTracerProvider(tp)

	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	tracer := &Tracer{
		tracer: tp.Tracer(config.ServiceName),
		config: config,
		tp:     tp, // Store reference to tracer provider
	}

	cleanup := func(ctx context.Context) error {
		// Force flush all pending spans before shutdown
		if err := tp.ForceFlush(ctx); err != nil {
			return fmt.Errorf("failed to flush traces: %w", err)
		}
		return tp.Shutdown(ctx)
	}

	return tracer, cleanup, nil
}

// StartSpan creates a new tracing span with the specified name and options.
// Returns the updated context containing the span and the span instance for further manipulation.
func (t *Tracer) StartSpan(ctx context.Context, spanName string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	return t.tracer.Start(ctx, spanName, opts...)
}

// SpanFromContext extracts the active span from the provided context.
// Returns a no-op span if no active span is found in the context.
func (t *Tracer) SpanFromContext(ctx context.Context) trace.Span {
	return trace.SpanFromContext(ctx)
}

// WithAttributes adds key-value attributes to the active span in the context.
// This is a convenience method for enriching spans with additional metadata.
func (t *Tracer) WithAttributes(ctx context.Context, attrs ...attribute.KeyValue) {
	span := trace.SpanFromContext(ctx)
	span.SetAttributes(attrs...)
}

// RecordError records an error event on the active span and sets the span status to error.
// This automatically marks the span as failed and includes the error message.
func (t *Tracer) RecordError(ctx context.Context, err error) {
	span := trace.SpanFromContext(ctx)
	span.RecordError(err)
	span.SetStatus(codes.Error, err.Error())
}

// SetStatus explicitly sets the status code and description for the active span.
// This is useful for marking spans as successful, failed, or providing custom status information.
func (t *Tracer) SetStatus(ctx context.Context, code codes.Code, description string) {
	span := trace.SpanFromContext(ctx)
	span.SetStatus(code, description)
}

// PubAttributes creates standard attributes for pub/sub operations.
// Returns a slice of attributes containing topic and shard information.
func (t *Tracer) PubAttributes(topic string, shard int) []attribute.KeyValue {
	return []attribute.KeyValue{
		attribute.String("pub.topic", topic),
		attribute.Int("pub.shard", shard),
	}
}

// ProducerAttributes creates attributes specific to producer operations.
// Includes basic pub/sub attributes plus batch size information.
func (t *Tracer) ProducerAttributes(topic string, shard int, batchSize int) []attribute.KeyValue {
	attrs := t.PubAttributes(topic, shard)
	attrs = append(attrs, attribute.Int("pub.batch_size", batchSize))
	return attrs
}

// ConsumerAttributes creates attributes specific to consumer operations.
// Includes basic pub/sub attributes plus subscription information.
func (t *Tracer) ConsumerAttributes(topic, subscription string, shard int) []attribute.KeyValue {
	attrs := t.PubAttributes(topic, shard)
	attrs = append(attrs, attribute.String("pub.subscription", subscription))
	return attrs
}

// DatabaseAttributes creates attributes for database operations.
// Includes operation type and database system information.
func (t *Tracer) DatabaseAttributes(operation string) []attribute.KeyValue {
	return []attribute.KeyValue{
		attribute.String("db.operation", operation),
		attribute.String("db.system", "couchbase"),
	}
}

// ErrorAttributes creates attributes based on error state.
// Returns error information if an error is provided, or success indication if nil.
func (t *Tracer) ErrorAttributes(err error) []attribute.KeyValue {
	if err == nil {
		return []attribute.KeyValue{
			attribute.Bool("error", false),
		}
	}
	return []attribute.KeyValue{
		attribute.Bool("error", true),
		attribute.String("error.type", fmt.Sprintf("%T", err)),
		attribute.String("error.message", err.Error()),
	}
}

// GetTracer returns the underlying OpenTelemetry tracer instance.
// This provides access to the raw tracer for advanced usage not covered by the convenience methods.
func (t *Tracer) GetTracer() trace.Tracer {
	return t.tracer
}
