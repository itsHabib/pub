package pub

import (
	"fmt"
	"time"

	"github.com/couchbase/gocb/v2"

	"pub/internal/couchbase"
)

type Lease struct {
	ID        string    `json:"id"`
	Sub       string    `json:"sub"`
	MessageID string    `json:"messageID"`
	Offset    uint64    `json:"offset"`
	Expires   time.Time `json:"expires"`

	couchbase.Cas `json:"-"`
}

func NewLeasesStore(cluster *gocb.Cluster, bucket *gocb.Bucket, scope string) (*couchbase.Couchbase[Lease], error) {
	collection := bucket.Scope(scope).Collection("leases")
	store, err := couchbase.NewCouchbase[Lease](cluster, bucket, collection)
	if err != nil {
		return nil, err
	}

	return store, nil
}

func LeaseKey(sub, msgID string) string {
	return fmt.Sprintf("lease::%s::%s", sub, msgID)
}
