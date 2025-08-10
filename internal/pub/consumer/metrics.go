package consumer

import (
	"context"
	"time"

	"pub/internal/pub"
	"pub/internal/pub/metrics"
)

// MetricsConsumer wraps a pub.Consumer with metrics collection
type MetricsConsumer struct {
	consumer pub.Consumer
	registry *metrics.Registry
}

// NewMetricsConsumer creates a new instrumented consumer
func NewMetricsConsumer(consumer pub.Consumer, registry *metrics.Registry) pub.Consumer {
	return &MetricsConsumer{
		consumer: consumer,
		registry: registry,
	}
}

// Pull implements pub.Consumer.Pull with metrics collection
func (c *MetricsConsumer) Pull(ctx context.Context, topic, sub string, shard int) (int, error) {
	start := time.Now()

	messagesConsumed, err := c.consumer.Pull(ctx, topic, sub, shard)
	duration := time.Since(start)

	c.registry.RecordConsumerPull(topic, sub, shard, messagesConsumed, duration, err)

	return messagesConsumed, err
}

// Ack implements pub.Consumer.Ack with metrics collection
func (c *MetricsConsumer) Ack(ctx context.Context, sub string, msg pub.Message) error {
	err := c.consumer.Ack(ctx, sub, msg)

	c.registry.RecordConsumerAck(msg.Topic, sub, msg.Shard, err)

	return err
}
