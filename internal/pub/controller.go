package pub

import "context"

// Controller defines the interface for message persistence and cursor management.
// The Controller serves as the central coordination layer between producers and consumers,
// managing all aspects of message storage, offset tracking, cursor positioning, and
// lease-based delivery guarantees.
type Controller interface {
	// GetCursor retrieves the current cursor position for a subscription.
	// Cursors track how far a particular subscription has progressed through
	// the messages in a topic shard.
	// Returns the current offset position for the subscription, or an error.
	GetCursor(ctx context.Context, topic, sub string, shard int) (uint64, error)

	// CommitCursor updates the cursor position for a subscription.
	// This is typically called after successfully processing a batch of messages
	// to advance the subscription's position in the message stream.
	// Returns an error if the cursor update fails.
	CommitCursor(topic, sub string, shard int, offset uint64) error

	// GetOffset retrieves the current write offset for a topic shard.
	// Offsets represent the next position where a new message will be written
	// in the message sequence for a given topic shard.
	// Returns the current write offset, or an error.
	GetOffset(ctx context.Context, topic string, shard int) (uint64, error)

	// CommitOffset updates the write offset for a topic shard.
	// This is called by producers after successfully writing a batch of messages
	// to advance the write position.
	// Returns an error if the offset update fails.
	CommitOffset(topic string, shard int, currentOffset uint64) error

	// InsertLease creates a temporary lease on a message for exclusive processing.
	// Leases prevent multiple consumers from processing the same message simultaneously
	// and provide a mechanism for message redelivery if processing fails.
	// Returns an error if the lease creation fails.
	InsertLease(ctx context.Context, sub string, msgID string, offset uint64) error

	// DeleteLease removes a lease on a message after successful processing.
	// This is typically called during message acknowledgment to release
	// the exclusive lock on the message.
	// Returns an error if the lease deletion fails.
	DeleteLease(ctx context.Context, sub string, msgID string) error

	// InsertMessage stores a new message in the persistent storage.
	// Messages are stored with their metadata including topic, shard, offset,
	// and timestamp information for ordered retrieval.
	// Returns an error if the message storage fails.
	InsertMessage(ctx context.Context, msg Message) error

	// LoadMessages retrieves messages from a topic shard starting from a given offset.
	// This method supports pagination through the limit parameter and maintains
	// message ordering based on offset values.
	// Returns a slice of messages in offset order, or an error.
	LoadMessages(ctx context.Context, topic string, shard int, fromOffset uint64, limit int) ([]Message, error)
}
