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

// Config holds configuration for the tracing setup
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

// Tracer wraps the OpenTelemetry tracer with convenience methods
type Tracer struct {
	tracer trace.Tracer
	config Config
	tp     *sdktrace.TracerProvider
}

// NewTracer creates and configures a new OpenTelemetry tracer
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

// StartSpan starts a new span with the given name and options
func (t *Tracer) StartSpan(ctx context.Context, spanName string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	return t.tracer.Start(ctx, spanName, opts...)
}

// SpanFromContext returns the current span from the context
func (t *Tracer) SpanFromContext(ctx context.Context) trace.Span {
	return trace.SpanFromContext(ctx)
}

// WithAttributes adds attributes to the current span in the context
func (t *Tracer) WithAttributes(ctx context.Context, attrs ...attribute.KeyValue) {
	span := trace.SpanFromContext(ctx)
	span.SetAttributes(attrs...)
}

// RecordError records an error on the current span
func (t *Tracer) RecordError(ctx context.Context, err error) {
	span := trace.SpanFromContext(ctx)
	span.RecordError(err)
	span.SetStatus(codes.Error, err.Error())
}

// SetStatus sets the status of the current span
func (t *Tracer) SetStatus(ctx context.Context, code codes.Code, description string) {
	span := trace.SpanFromContext(ctx)
	span.SetStatus(code, description)
}

// Common attribute helpers for pub/sub operations
func (t *Tracer) PubAttributes(topic string, shard int) []attribute.KeyValue {
	return []attribute.KeyValue{
		attribute.String("pub.topic", topic),
		attribute.Int("pub.shard", shard),
	}
}

func (t *Tracer) ProducerAttributes(topic string, shard int, batchSize int) []attribute.KeyValue {
	attrs := t.PubAttributes(topic, shard)
	attrs = append(attrs, attribute.Int("pub.batch_size", batchSize))
	return attrs
}

func (t *Tracer) ConsumerAttributes(topic, subscription string, shard int) []attribute.KeyValue {
	attrs := t.PubAttributes(topic, shard)
	attrs = append(attrs, attribute.String("pub.subscription", subscription))
	return attrs
}

func (t *Tracer) DatabaseAttributes(operation string) []attribute.KeyValue {
	return []attribute.KeyValue{
		attribute.String("db.operation", operation),
		attribute.String("db.system", "couchbase"),
	}
}

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

// GetTracer returns the underlying OpenTelemetry tracer
func (t *Tracer) GetTracer() trace.Tracer {
	return t.tracer
}
