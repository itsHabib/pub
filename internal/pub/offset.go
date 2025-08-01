package pub

import (
	"fmt"

	"github.com/couchbase/gocb/v2"

	"pub/internal/couchbase"
)

// Offset represents the current write position for a topic shard.
// Offsets track the next position where a new message will be written,
// enabling proper message ordering and sequential numbering within each shard.
type Offset struct {
	// ID is the unique identifier for this offset, typically formatted as
	// "offset::<topic>::<shard>"
	ID string `json:"id"`
	// N is the current offset value (next write position)
	N uint64 `json:"n"`

	// Cas provides optimistic concurrency control for Couchbase operations
	couchbase.Cas `json:"-"`
}

// NewOffsetsStore creates a new Couchbase-backed storage for Offset entities.
// This initializes the connection to the "offsets" collection within the specified scope.
//
// Parameters:
//   - cluster: Couchbase cluster connection
//   - bucket: Couchbase bucket containing the offsets collection
//   - scope: Name of the scope containing the offsets collection
//
// Returns a configured storage instance for Offset operations, or an error if setup fails.
func NewOffsetsStore(cluster *gocb.Cluster, bucket *gocb.Bucket, scope string) (*couchbase.Couchbase[Offset], error) {
	collection := bucket.Scope(scope).Collection("offsets")
	store, err := couchbase.NewCouchbase[Offset](cluster, bucket, collection)
	if err != nil {
		return nil, err
	}

	return store, nil
}

// OffsetKey generates a unique key for storing offsets in Couchbase.
// The key format is "offset::<topic>::<shard>" to ensure uniqueness
// per topic shard and enable efficient querying.
//
// Parameters:
//   - topic: The topic name
//   - shard: The shard number within the topic
//
// Returns a formatted key string for Couchbase document storage.
func OffsetKey(topic string, shard int) string {
	return fmt.Sprintf("offset::%s::%d", topic, shard)
}
