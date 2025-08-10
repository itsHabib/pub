package consumer

import (
	"context"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"

	"pub/internal/pub"
	"pub/internal/pub/tracing"
)

// TracedConsumer wraps a pub.Consumer with distributed tracing
// Layer order: TracedConsumer -> MetricsConsumer -> Consumer (real thing)
type TracedConsumer struct {
	consumer pub.Consumer
	tracer   *tracing.Tracer
}

// NewTracedConsumer creates a new traced consumer that wraps a metrics consumer
func NewTracedConsumer(consumer pub.Consumer, tracer *tracing.Tracer) pub.Consumer {
	return &TracedConsumer{
		consumer: consumer,
		tracer:   tracer,
	}
}

// Pull implements pub.Consumer.Pull with distributed tracing
func (c *TracedConsumer) Pull(ctx context.Context, topic, sub string, shard int) (int, error) {
	// Start the main pull span
	ctx, span := c.tracer.StartSpan(ctx, "consumer.pull")
	defer span.End()

	// Add span attributes
	span.SetAttributes(c.tracer.ConsumerAttributes(topic, sub, shard)...)

	// Call the wrapped consumer (which includes metrics)
	messagesConsumed, err := c.consumer.Pull(ctx, topic, sub, shard)

	// Add result attributes
	span.SetAttributes(
		attribute.Int("pub.messages_consumed", messagesConsumed),
	)

	// Record error if any
	if err != nil {
		c.tracer.RecordError(ctx, err)
	} else {
		span.SetStatus(codes.Ok, "")
	}

	// Add final attributes
	span.SetAttributes(c.tracer.ErrorAttributes(err)...)

	return messagesConsumed, err
}

// Ack implements pub.Consumer.Ack with distributed tracing
func (c *TracedConsumer) Ack(ctx context.Context, sub string, msg pub.Message) error {
	// Start the ack span
	ctx, span := c.tracer.StartSpan(ctx, "consumer.ack")
	defer span.End()

	// Add span attributes
	span.SetAttributes(
		attribute.String("pub.subscription", sub),
		attribute.String("pub.topic", msg.Topic),
		attribute.Int("pub.shard", msg.Shard),
		attribute.String("pub.message_id", msg.ID),
		attribute.Int64("pub.message_offset", int64(msg.Offset)),
	)

	// Call the wrapped consumer (which includes metrics)
	err := c.consumer.Ack(ctx, sub, msg)

	// Record error if any
	if err != nil {
		c.tracer.RecordError(ctx, err)
	} else {
		span.SetStatus(codes.Ok, "")
	}

	// Add final attributes
	span.SetAttributes(c.tracer.ErrorAttributes(err)...)

	return err
}
