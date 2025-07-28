package pub

import (
	"fmt"

	"github.com/couchbase/gocb/v2"

	"pub/internal/couchbase"
)

type Offset struct {
	ID string `json:"id"`
	N  uint64 `json:"n"`

	couchbase.Cas `json:"-"`
}

func NewOffsetsStore(cluster *gocb.Cluster, bucket *gocb.Bucket, scope string) (*couchbase.Couchbase[Offset], error) {
	collection := bucket.Scope(scope).Collection("offsets")
	store, err := couchbase.NewCouchbase[Offset](cluster, bucket, collection)
	if err != nil {
		return nil, err
	}

	return store, nil
}

func OffsetKey(topic string, shard int) string {
	return fmt.Sprintf("offset::%s::%d", topic, shard)
}
