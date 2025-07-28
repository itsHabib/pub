package pub

import "context"

// Controller defines the interface for message persistence and cursor management.
type Controller interface {
	// GetCursor retrieves the current cursor position for a subscription.
	GetCursor(ctx context.Context, topic, sub string, shard int) (uint64, error)

	// CommitCursor updates the offset for a subscription.
	CommitCursor(topic, sub string, shard int, offset uint64) error

	// GetOffset retrieves the current offset for a topic shard.
	GetOffset(ctx context.Context, topic string, shard int) (uint64, error)

	// CommitOffset updates the offset for a topic shard.
	CommitOffset(topic string, shard int, currentOffset uint64) error

	// InsertLease creates a lease for a message.
	InsertLease(ctx context.Context, sub string, msgID string, offset uint64) error

	// DeleteLease removes a lease for a message.
	DeleteLease(ctx context.Context, sub string, msgID string) error

	// InsertMessage stores a new message.
	InsertMessage(ctx context.Context, msg Message) error

	// LoadMessages loads messages from a topic shard starting from a given offset.
	LoadMessages(ctx context.Context, topic string, shard int, fromOffset uint64, limit int) ([]Message, error)
}
