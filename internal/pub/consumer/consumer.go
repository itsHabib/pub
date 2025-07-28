package consumer

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"time"

	"github.com/couchbase/gocb/v2"
	"golang.org/x/sync/errgroup"

	"pub/internal/pub"
	"pub/internal/validator"
)

type Consumer struct {
	controller pub.Controller
	batchSize  int
}

func NewConsumer(storage pub.Controller, batchSize int) (*Consumer, error) {
	c := Consumer{
		controller: storage,
		batchSize:  batchSize,
	}

	if err := validator.Validate("consumer", c.controller, c.batchSize); err != nil {
		return nil, fmt.Errorf("failed to validate consumer deps: %w", err)
	}

	return &c, nil
}

func (c *Consumer) Pull(ctx context.Context, topic, sub string, shard int) error {
	offset, err := c.controller.GetCursor(ctx, topic, sub, shard)
	if err != nil {
		return fmt.Errorf("failed to get cursor: %w", err)
	}

	msgs, err := c.controller.LoadMessages(ctx, topic, shard, offset, c.batchSize)
	if err != nil {
		return fmt.Errorf("failed to load messages: %w", err)
	}

	leased := make([]pub.Message, 0, len(msgs))
	for _, msg := range msgs {
		err := c.controller.InsertLease(ctx, sub, msg.ID, msg.Offset)
		switch {
		case err == nil:
			leased = append(leased, msg)
		case errors.Is(err, gocb.ErrDocumentExists):
		default:
			return fmt.Errorf("failed to insert lease for message %s: %w", msg.ID, err)
		}
	}

	// process messages
	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(c.batchSize / 2)
	for _, msg := range leased {
		msg := msg
		g.Go(func() error {
			// Simulate message processing
			time.Sleep(time.Duration(50+rand.Intn(450)) * time.Millisecond)

			if err := c.Ack(gctx, topic, msg); err != nil {
				return fmt.Errorf("failed to ack message %s: %w", msg.ID, err)
			}

			return nil
		})
	}

	return nil
}

func (c *Consumer) Ack(ctx context.Context, sub string, msg pub.Message) error {
	if err := c.controller.DeleteLease(ctx, sub, msg.ID); err != nil {
		return fmt.Errorf("failed to delete lease for message %s: %w", msg.ID, err)
	}

	if err := c.controller.CommitCursor(msg.Topic, sub, msg.Shard, msg.Offset); err != nil {
		return fmt.Errorf("failed to commit cursor for topic %s sub %s shard %d: %w", msg.Topic, sub, msg.Shard, err)
	}

	return nil
}
