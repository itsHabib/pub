package producer

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/couchbase/gocb/v2"
	"go.uber.org/zap"

	"pub/internal/pub"
	"pub/internal/validator"
)

// Producer is a concrete implementation of the pub.Producer interface.
// It handles the conversion of events to messages, manages offset tracking,
// and ensures atomic publishing operations through the controller layer.
type Producer struct {
	controller pub.Controller
	logger     *zap.Logger
}

// NewProducer creates a new Producer instance with the provided dependencies.
// The producer requires a controller for message persistence and offset management,
// and a logger for operational observability.
//
// Parameters:
//   - controller: Implementation of pub.Controller for data operations
//   - logger: Structured logger for operational events and errors
//
// Returns a configured Producer instance or an error if validation fails.
func NewProducer(controller pub.Controller, logger *zap.Logger) (*Producer, error) {
	p := Producer{
		controller: controller,
		logger:     logger,
	}

	if err := validator.Validate("producer", p.controller); err != nil {
		return nil, fmt.Errorf("failed to validate producer controller: %w", err)
	}

	return &p, nil
}

// PublishBatch implements pub.Producer.PublishBatch by converting events to messages
// and persisting them with proper offset management. This method ensures atomic
// publishing of all events in the batch - either all succeed or all fail.
//
// The publishing process:
// 1. Retrieves the current offset for the topic shard (defaults to 0 if not found)
// 2. Converts each event to a message with sequential offset numbering
// 3. Persists each message (ignoring duplicates for idempotency)
// 4. Updates the offset to reflect the new write position
//
// Returns an error if any step in the publishing process fails.
func (p *Producer) PublishBatch(ctx context.Context, topic string, shard int, events ...pub.Event) error {
	if len(events) == 0 {
		return nil
	}

	logger := p.logger.With(zap.String("topic", topic), zap.Int("shard", shard))

	offset, err := p.controller.GetOffset(ctx, topic, shard)
	switch {
	case err == nil:
		logger.Debug("retrieved offset", zap.Uint64("offset", offset))
	case errors.Is(err, gocb.ErrDocumentNotFound):
		offset = 0
	default:
		const msg = "failed to get offset for topic and shard"
		logger.Error(msg, zap.Error(err))
		return fmt.Errorf(msg+": %w", err)
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
			const msg = "failed to insert message"
			logger.Error(msg, zap.String("messageId", m.ID), zap.Error(err))
			return fmt.Errorf("msg: %w", err)
		}
	}

	offset += uint64(len(events))

	if err := p.controller.CommitOffset(topic, shard, offset); err != nil {
		const msg = "failed to commit offset for topic and shard"
		logger.Error(msg, zap.Error(err))
		return fmt.Errorf(msg+": %w", err)
	}

	logger.Debug("committed offset", zap.Uint64("offset", offset))

	return nil
}

// ptr is a utility function that returns a pointer to the provided value.
// This is commonly used for optional fields that need pointer types.
func ptr[T any](v T) *T {
	return &v
}
