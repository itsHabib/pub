package producer

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/couchbase/gocb/v2"

	"pub/internal/pub"
	"pub/internal/validator"
)

type Producer struct {
	controller pub.Controller
}

func NewProducer(controller pub.Controller) (*Producer, error) {
	p := Producer{
		controller: controller,
	}

	if err := validator.Validate("producer", p.controller); err != nil {
		return nil, fmt.Errorf("failed to validate producer controller: %w", err)
	}

	return &p, nil
}

func (p *Producer) PublishBatch(ctx context.Context, topic string, shard int, events []pub.Event) error {
	if len(events) == 0 {
		return nil
	}

	offset, err := p.controller.GetOffset(ctx, topic, shard)
	switch {
	case err == nil:
	case errors.Is(err, gocb.ErrDocumentNotFound):
		offset = 0
	default:
		return fmt.Errorf("failed to get offset for topic %s shard %d: %w", topic, shard, err)
	}

	for i, e := range events {
		m := pub.Message{
			ID:          pub.MessageKey(topic, shard, offset+uint64(i)),
			Offset:      offset + uint64(i),
			Topic:       topic,
			Shard:       shard,
			PublishTime: ptr(time.Now().UTC()),
			Event:       e.Type,
			Payload:     e.Payload,
		}

		if err := p.controller.InsertMessage(ctx, m); err != nil && !errors.Is(err, gocb.ErrDocumentExists) {
			return fmt.Errorf("failed to insert message with ID %s: %w", m.ID, err)
		}

		offset++
	}

	if err := p.controller.CommitOffset(topic, shard, offset); err != nil {
		return fmt.Errorf("failed to commit offset for topic %s shard %d: %w", topic, shard, err)
	}

	return nil
}
func ptr[T any](v T) *T {
	return &v
}
