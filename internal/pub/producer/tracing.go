package producer

import (
	"context"

	"go.opentelemetry.io/otel/codes"

	"pub/internal/pub"
	"pub/internal/pub/tracing"
)

// TracedProducer wraps a pub.Producer with distributed tracing
// Layer order: TracedProducer -> MetricsProducer -> Producer (real thing)
type TracedProducer struct {
	producer pub.Producer
	tracer   *tracing.Tracer
}

// NewTracedProducer creates a new traced producer that wraps a metrics producer
func NewTracedProducer(producer pub.Producer, tracer *tracing.Tracer) pub.Producer {
	return &TracedProducer{
		producer: producer,
		tracer:   tracer,
	}
}

// PublishBatch implements pub.Producer.PublishBatch with distributed tracing
func (p *TracedProducer) PublishBatch(ctx context.Context, topic string, shard int, events ...pub.Event) error {
	// Start the main publish span
	ctx, span := p.tracer.StartSpan(ctx, "producer.publish_batch")
	defer span.End()

	// Add span attributes
	span.SetAttributes(p.tracer.ProducerAttributes(topic, shard, len(events))...)

	// Call the wrapped producer (which includes metrics)
	err := p.producer.PublishBatch(ctx, topic, shard, events...)

	// Record error if any
	if err != nil {
		p.tracer.RecordError(ctx, err)
	} else {
		span.SetStatus(codes.Ok, "")
	}

	// Add final attributes
	span.SetAttributes(p.tracer.ErrorAttributes(err)...)

	return err
}
