package pub

import (
	"fmt"
	"time"

	"github.com/couchbase/gocb/v2"

	"pub/internal/couchbase"
)

// Lease represents a temporary exclusive lock on a message during processing.
// Leases prevent multiple consumers from processing the same message simultaneously
// and provide a mechanism for automatic redelivery if processing fails or times out.
type Lease struct {
	// ID is the unique identifier for this lease, typically formatted as
	// "lease::<subscription>::<messageId>"
	ID string `json:"id"`
	// Sub is the subscription name that holds this lease
	Sub string `json:"sub"`
	// MessageID is the unique identifier of the message being leased
	MessageID string `json:"messageId"`
	// Offset is the offset position of the message within its topic shard
	Offset uint64 `json:"offset"`
	// Expires is the timestamp when this lease expires and the message becomes
	// available for redelivery (UTC)
	Expires time.Time `json:"expires"`

	// Cas provides optimistic concurrency control for Couchbase operations
	couchbase.Cas `json:"-"`
}

// NewLeasesStore creates a new Couchbase-backed storage for Lease entities.
// This initializes the connection to the "leases" collection within the specified scope.
func NewLeasesStore(cluster *gocb.Cluster, bucket *gocb.Bucket, scope string) (*couchbase.Couchbase[Lease], error) {
	collection := bucket.Scope(scope).Collection("leases")
	store, err := couchbase.NewCouchbase[Lease](cluster, bucket, collection)
	if err != nil {
		return nil, err
	}

	return store, nil
}

// LeaseKey generates a unique key for storing leases in Couchbase.
// The key format is "lease::<subscription>::<messageId>" to ensure uniqueness
// per subscription and message combination.
func LeaseKey(sub, msgID string) string {
	return fmt.Sprintf("lease::%s::%s", sub, msgID)
}
