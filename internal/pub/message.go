package pub

import (
	"fmt"
	"time"

	"github.com/couchbase/gocb/v2"

	"pub/internal/couchbase"
)

// Message represents a persisted event in the pub/sub system.
// Messages are created from Events by producers and include additional
// metadata required for ordering, delivery, and storage management.
type Message struct {
	// ID is the unique identifier for this message, typically formatted as
	// "message::<topic>::<shard>::<offset>"
	ID string `json:"id"`
	// Topic is the name of the topic this message belongs to
	Topic string `json:"topic"`
	// Shard is the shard number within the topic for horizontal scaling
	Shard int `json:"shard"`
	// Offset is the sequential position of this message within the topic shard
	Offset uint64 `json:"offset"`
	// Event is the type/name of the event (copied from Event.Type)
	Event string `json:"event"`
	// Payload is the event data (copied from Event.Payload)
	Payload any `json:"payload"`
	// PublishTime is the timestamp when this message was published (UTC)
	PublishTime *time.Time `json:"publishTime,omitempty"`

	// Cas provides optimistic concurrency control for Couchbase operations
	couchbase.Cas `json:"-"`
}

// NewMessagesStore creates a new Couchbase-backed storage for Message entities.
// This initializes the connection to the "messages" collection within the specified scope.
// Returns a configured storage instance for Message operations, or an error if setup fails.
func NewMessagesStore(cluster *gocb.Cluster, bucket *gocb.Bucket, scope string) (*couchbase.Couchbase[Message], error) {
	collection := bucket.Scope(scope).Collection("messages")
	store, err := couchbase.NewCouchbase[Message](cluster, bucket, collection)
	if err != nil {
		return nil, err
	}

	return store, nil
}

// MessageKey generates a unique key for storing messages in Couchbase.
// The key format is "message::<topic>::<shard>::<offset>" to ensure uniqueness
// and enable efficient querying by topic and shard.
// Returns a formatted key string for Couchbase document storage.
func MessageKey(topic string, shard int, offset uint64) string {
	return fmt.Sprintf("message::%s::%d::%d", topic, shard, offset)
}
