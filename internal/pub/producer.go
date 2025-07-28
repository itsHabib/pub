package pub

import "context"

// Producer defines the interface for publishing messages to topics.
type Producer interface {
	// PublishBatch publishes a batch of events to a topic shard.
	PublishBatch(ctx context.Context, topic string, shard int, events ...Event) error
}
