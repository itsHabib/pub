package pub

import "context"

// Consumer defines the interface for consuming messages from topics.
type Consumer interface {
	// Pull retrieves and processes messages from a topic shard for a subscription.
	Pull(ctx context.Context, topic, sub string, shard int) error

	// Ack acknowledges that a message has been processed.
	Ack(ctx context.Context, sub string, msg Message) error
}
