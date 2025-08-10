package consumer

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"sort"
	"time"

	"github.com/couchbase/gocb/v2"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"

	"pub/internal/pub"
	"pub/internal/validator"
)

// Consumer is a concrete implementation of the pub.Consumer interface.
// It provides at-least-once delivery semantics through lease-based message processing,
// cursor management, and automatic acknowledgment handling.
type Consumer struct {
	controller pub.Controller
	logger     *zap.Logger
	batchSize  int
}

// NewConsumer creates a new Consumer instance with the provided dependencies.
// The consumer requires a controller for data operations, a logger for observability,
// and a batch size for controlling how many messages to process concurrently.
// Returns a configured Consumer instance or an error if validation fails.
func NewConsumer(controller pub.Controller, logger *zap.Logger, batchSize int) (*Consumer, error) {
	c := Consumer{
		controller: controller,
		logger:     logger,
		batchSize:  batchSize,
	}

	if err := validator.Validate("consumer", c.controller, c.batchSize); err != nil {
		return nil, fmt.Errorf("failed to validate consumer deps: %w", err)
	}

	return &c, nil
}

// Pull implements pub.Consumer.Pull by retrieving messages from a topic shard,
// acquiring leases for exclusive processing, and handling message acknowledgment.
// This method provides at-least-once delivery semantics and automatic cursor management.
// Returns the number of messages successfully processed, or an error.
func (c *Consumer) Pull(ctx context.Context, topic, sub string, shard int) (int, error) {
	logger := c.logger.With(zap.String("topic", topic), zap.String("sub", sub))
	logger.Info("attempting to pull messages")

	offset, err := c.controller.GetCursor(ctx, topic, sub, shard)
	if err != nil {
		return 0, fmt.Errorf("failed to get cursor: %w", err)
	}
	logger.Debug("got cursor", zap.Uint64("offset", offset))

	msgs, err := c.controller.LoadMessages(ctx, topic, shard, offset, c.batchSize)
	if err != nil {
		return 0, fmt.Errorf("failed to load messages: %w", err)
	}

	sort.Slice(msgs, func(i, j int) bool {
		return msgs[i].Offset < msgs[j].Offset
	})

	logger.Debug("loaded messages",
		zap.Int("count", len(msgs)),
		zap.Uint64("fromOffset", offset),
	)

	leased := make([]pub.Message, 0, len(msgs))
	for _, msg := range msgs {
		err := c.controller.InsertLease(ctx, sub, msg.ID, msg.Offset)
		switch {
		case err == nil:
			leased = append(leased, msg)
		case errors.Is(err, gocb.ErrDocumentExists):
			logger.Debug("lease already exists for message", zap.String("messageId", msg.ID))
		default:
			return 0, fmt.Errorf("failed to insert lease for message %s: %w", msg.ID, err)
		}
	}

	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(c.batchSize)
	for _, msg := range leased {
		msg := msg
		g.Go(func() error {
			logger.Debug("processing message", zap.Any("message", msg))
			time.Sleep(time.Duration(50+rand.Intn(200)) * time.Millisecond)

			if err := c.ack(gctx, logger, sub, msg); err != nil {
				const errMsg = "failed to ack message"
				logger.Error(errMsg, zap.String("messageId", msg.ID), zap.Error(err))
				return fmt.Errorf(errMsg+": %w", err)
			}
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		const msg = "failed to process messages"
		logger.Error(msg, zap.Error(err))
		return 0, fmt.Errorf(msg+": %w", err)
	}

	return len(leased), nil
}

// Ack implements pub.Consumer.Ack by acknowledging a successfully processed message.
// This removes the lease on the message and advances the cursor for the subscription.
func (c *Consumer) Ack(ctx context.Context, sub string, msg pub.Message) error {
	return c.ack(ctx, c.logger, sub, msg)
}

// ack handles the acknowledgment process for a successfully processed message.
// This involves removing the lease to release the message lock and advancing
// the cursor to mark the message as processed.
// Returns an error if either the lease deletion or cursor commit fails.
func (c *Consumer) ack(ctx context.Context, logger *zap.Logger, sub string, msg pub.Message) error {
	logger.Debug("acknowledging message", zap.String("messageId", msg.ID))

	if err := c.controller.DeleteLease(ctx, sub, msg.ID); err != nil {
		return fmt.Errorf("failed to delete lease for message %s: %w", msg.ID, err)
	}

	logger.Debug("lease deleted for message", zap.String("messageId", msg.ID))

	// Always commit to next offset since we've processed this message
	if err := c.controller.CommitCursor(msg.Topic, sub, msg.Shard, msg.Offset+1); err != nil {
		return fmt.Errorf("failed to commit cursor for topic %s sub %s shard %d: %w", msg.Topic, sub, msg.Shard, err)
	}

	logger.Debug("cursor committed for offset", zap.Uint64("offset", msg.Offset), zap.String("messageId", msg.ID))

	return nil
}
