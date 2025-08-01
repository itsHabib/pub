package pub

import "context"

// Producer defines the interface for publishing messages to topics.
// Producers are responsible for taking events and persisting them as messages
// in the distributed messaging system with proper ordering and delivery guarantees.
type Producer interface {
	// PublishBatch publishes a batch of events to a specific topic shard.
	// This method ensures atomic publishing of multiple events and handles
	// offset management automatically. Individual event failures within a batch
	// will cause the entire batch to fail.
	// Returns an error if the publishing operation fails.
	PublishBatch(ctx context.Context, topic string, shard int, events ...Event) error
}
