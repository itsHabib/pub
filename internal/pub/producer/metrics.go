package producer

import (
	"context"
	"time"

	"pub/internal/pub"
	"pub/internal/pub/metrics"
)

// MetricsProducer wraps a pub.Producer with metrics collection
type MetricsProducer struct {
	producer pub.Producer
	registry *metrics.Registry
}

// NewMetricsProducer creates a new instrumented producer
func NewMetricsProducer(producer pub.Producer, registry *metrics.Registry) pub.Producer {
	return &MetricsProducer{
		producer: producer,
		registry: registry,
	}
}

// PublishBatch implements pub.Producer.PublishBatch with metrics collection
func (p *MetricsProducer) PublishBatch(ctx context.Context, topic string, shard int, events ...pub.Event) error {
	start := time.Now()
	batchSize := len(events)

	err := p.producer.PublishBatch(ctx, topic, shard, events...)
	duration := time.Since(start)

	p.registry.RecordProducerPublish(topic, shard, batchSize, duration, err)

	return err
}
