package controller

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/couchbase/gocb/v2"

	"pub/internal/couchbase"
	"pub/internal/pub"
	"pub/internal/validator"
)

type Controller struct {
	cursors      *couchbase.Couchbase[pub.Cursor]
	leases       *couchbase.Couchbase[pub.Lease]
	messages     *couchbase.Couchbase[pub.Message]
	offsets      *couchbase.Couchbase[pub.Offset]
	transactions *couchbase.Transactions
	// don't love that take we in the bucket scope for this, should probably
	// abstract out the querying so i dont have to
	bucket string
	scope  string
}

func NewController(
	cursors *couchbase.Couchbase[pub.Cursor],
	leases *couchbase.Couchbase[pub.Lease],
	messages *couchbase.Couchbase[pub.Message],
	offsets *couchbase.Couchbase[pub.Offset],
	transactions *couchbase.Transactions,
	bucket, scope string,
) (*Controller, error) {
	s := Controller{
		cursors:      cursors,
		leases:       leases,
		messages:     messages,
		offsets:      offsets,
		transactions: transactions,
		bucket:       bucket,
		scope:        scope,
	}

	if err := validator.Validate(
		"storage",
		s.cursors,
		s.leases,
		s.messages,
		s.offsets,
		s.transactions,
		s.bucket,
		s.scope,
	); err != nil {
		return nil, fmt.Errorf("failed to validate storage dependencies: %w", err)
	}

	return &s, nil
}

func (c *Controller) GetCursor(ctx context.Context, topic, sub string, shard int) (uint64, error) {
	key := pub.CursorKey(topic, sub, shard)

	cur, err := c.cursors.Get(ctx, key, nil)
	switch {
	case err == nil:
		return cur.Offset, nil
	case errors.Is(err, gocb.ErrDocumentNotFound):
		// Return default cursor for new subscriptions
		return 0, nil
	default:
		return 0, fmt.Errorf("failed to get cursor: %w", err)
	}
}

// CommitCursor updates the offset for a subscription with optimistic concurrency control
func (c *Controller) CommitCursor(topic, sub string, shard int, offset uint64) error {
	key := pub.CursorKey(topic, sub, shard)

	_, err := c.transactions.Transaction(func(r couchbase.TransactionRunner) error {
		retry := true
		for retry {
			retry = false

			res, err := r.Get(c.cursors, key)
			switch {
			case err == nil:
			case errors.Is(err, gocb.ErrDocumentNotFound):
				cursor := pub.Cursor{
					ID:     key,
					Topic:  topic,
					Sub:    sub,
					Shard:  shard,
					Offset: offset,
				}
				_, err := r.Insert(c.cursors, key, cursor)
				switch {
				case err == nil:
					return nil
				case errors.Is(err, gocb.ErrDocumentExists):
					// allow retry if the document already exists
					retry = true
					continue
				default:
					return fmt.Errorf("failed to insert new cursor: %w", err)
				}
			default:
				return fmt.Errorf("failed to get cursor: %w", err)
			}

			var cursor pub.Cursor
			if err := res.Content(&cursor); err != nil {
				return fmt.Errorf("failed to decode cursor: %w", err)
			}

			if offset <= cursor.Offset {
				// current offset is already greater or equal, no update needed
				return nil
			}

			cursor.Offset = offset
			if _, err := r.Replace(res, cursor); err != nil {
				return fmt.Errorf("failed to replace cursor: %w", err)
			}
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to commit cursor: %w", err)
	}

	return nil
}

func (c *Controller) GetOffset(ctx context.Context, topic string, shard int) (uint64, error) {
	offsetKey := pub.OffsetKey(topic, shard)
	offset, err := c.offsets.Get(ctx, offsetKey, nil)
	if err != nil {
		return 0, fmt.Errorf("failed to get offset: %w", err)
	}

	return offset.N, nil
}

func (c *Controller) CommitOffset(topic string, shard int, currentOffset uint64) error {
	offsetKey := pub.OffsetKey(topic, shard)

	_, err := c.transactions.Transaction(func(r couchbase.TransactionRunner) error {
		retry := true
		for retry {
			retry = false

			offsetRes, err := r.Get(c.offsets, offsetKey)
			switch {
			case err == nil:
			case errors.Is(err, gocb.ErrDocumentNotFound):
				offset := &pub.Offset{ID: offsetKey, N: currentOffset}
				_, err := r.Insert(c.offsets, offsetKey, *offset)
				switch {
				case err == nil:
					return nil
				case errors.Is(err, gocb.ErrDocumentExists):
					// allow retry if the document already exists
					retry = true
					continue
				default:
					return fmt.Errorf("failed to insert new offset: %w", err)
				}
			default:
				return fmt.Errorf("failed to get offset for topic %s shard %d: %w", topic, shard, err)
			}

			var existing pub.Offset
			if err := offsetRes.Content(&existing); err != nil {
				return fmt.Errorf("failed to decode offset: %w", err)
			}

			if currentOffset <= existing.N {
				// No update needed, current offset is already greater or equal
				return nil
			}

			existing.N = currentOffset
			if _, err := r.Replace(offsetRes, existing); err != nil {
				return fmt.Errorf("failed to replace offset: %w", err)
			}
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to commit transaction for commiting offset for topic %s shard %d: %w", topic, shard, err)
	}

	return nil
}

// InsertLease creates a lease for a message
func (c *Controller) InsertLease(ctx context.Context, sub string, msgID string, offset uint64) error {
	key := pub.LeaseKey(sub, msgID)
	timeout := time.Minute

	lease := pub.Lease{
		ID:        key,
		Offset:    offset,
		Sub:       sub,
		MessageID: msgID,
		Expires:   time.Now().UTC().Add(timeout),
	}

	if err := c.leases.Insert(ctx, key, lease, &gocb.InsertOptions{
		Expiry: timeout,
	}); err != nil {
		return fmt.Errorf("failed to insert lease: %w", err)
	}

	return nil
}

// DeleteLease removes a lease for a message
func (c *Controller) DeleteLease(ctx context.Context, sub string, msgID string) error {
	key := pub.LeaseKey(sub, msgID)

	if err := c.leases.Remove(ctx, key, nil); err != nil && !errors.Is(err, gocb.ErrDocumentNotFound) {
		return fmt.Errorf("failed to delete lease: %w", err)
	}

	return nil
}

func (c *Controller) InsertMessage(ctx context.Context, msg pub.Message) error {
	if err := c.messages.Insert(
		ctx,
		msg.ID,
		msg,
		&gocb.InsertOptions{
			Expiry: 7 * 24 * time.Hour,
		},
	); err != nil {
		return fmt.Errorf("failed to insert message: %w", err)
	}

	return nil
}

// LoadMessages loads messages from a topic shard starting from a given offset
func (c *Controller) LoadMessages(ctx context.Context, topic string, shard int, fromOffset uint64, limit int) ([]pub.Message, error) {
	query := fmt.Sprintf(`
		SELECT RAW m
		FROM %s.%s.%s m
		WHERE`+"`offset`"+` >= %d
		AND m.topic = '%s'
		AND m.shard = %d
		ORDER BY`+"`offset`"+`ASC
		LIMIT %d`,
		c.bucket,
		c.scope,
		c.messages.Collection().Name(),
		fromOffset,
		topic,
		shard,
		limit,
	)

	messages, err := c.messages.Query(ctx, query, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to query messages: %w", err)
	}

	return messages, nil
}
