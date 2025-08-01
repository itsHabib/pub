package pub

import "context"

// Consumer defines the interface for consuming messages from topics.
// Consumers provide at-least-once delivery semantics through lease-based
// message processing and acknowledgment mechanisms.
type Consumer interface {
	// Pull retrieves and processes messages from a topic shard for a subscription.
	// This method handles cursor management, lease acquisition, and message processing
	// in a single atomic operation.
	// Returns the number of messages processed, or an error if the operation fails.
	// A return value of 0 indicates no messages were available for processing.
	Pull(ctx context.Context, topic, sub string, shard int) (int, error)

	// Ack acknowledges that a message has been successfully processed.
	// This removes the lease on the message and advances the cursor for the subscription.
	// Messages that are not acknowledged within the lease timeout will be redelivered.
	// Returns an error if the acknowledgment fails.
	Ack(ctx context.Context, sub string, msg Message) error
}
