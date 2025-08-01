package pub

import (
	"fmt"

	"github.com/couchbase/gocb/v2"

	"pub/internal/couchbase"
)

// Cursor represents the current position of a subscription within a topic shard.
// Cursors track how far each subscription has progressed through the message stream,
// enabling resumable consumption and preventing message loss during consumer restarts.
type Cursor struct {
	// ID is the unique identifier for this cursor, typically formatted as
	// "cursor::<topic>::<subscription>::<shard>"
	ID string `json:"id"`
	// Topic is the name of the topic this cursor is tracking
	Topic string `json:"topic"`
	// Sub is the subscription name (consumer group identifier)
	Sub string `json:"sub"`
	// Shard is the shard number within the topic
	Shard int `json:"shard"`
	// Offset is the current position in the message stream that this subscription has processed
	Offset uint64 `json:"offset"`

	// Cas provides optimistic concurrency control for Couchbase operations
	couchbase.Cas `json:"-"`
}

// NewCursorsStore creates a new Couchbase-backed storage for Cursor entities.
// This initializes the connection to the "cursors" collection within the specified scope.
// Returns a configured storage instance for Cursor operations, or an error if setup fails.
func NewCursorsStore(cluster *gocb.Cluster, bucket *gocb.Bucket, scope string) (*couchbase.Couchbase[Cursor], error) {
	collection := bucket.Scope(scope).Collection("cursors")
	store, err := couchbase.NewCouchbase[Cursor](cluster, bucket, collection)
	if err != nil {
		return nil, err
	}

	return store, nil
}

// CursorKey generates a unique key for storing cursors in Couchbase.
// The key format is "cursor::<topic>::<subscription>::<shard>" to ensure uniqueness
// per subscription and enable efficient querying.
// Returns a formatted key string for Couchbase document storage.
// Returns a formatted key string for Couchbase document storage.
func CursorKey(topic, sub string, shard int) string {
	return fmt.Sprintf("cursor::%s::%s::%d", topic, sub, shard)
}
