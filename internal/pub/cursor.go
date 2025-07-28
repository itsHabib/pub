package pub

import (
	"fmt"

	"github.com/couchbase/gocb/v2"

	"pub/internal/couchbase"
)

type Cursor struct {
	ID     string `json:"id"`
	Topic  string `json:"topic"`
	Sub    string `json:"sub"`
	Shard  int    `json:"shard"`
	Offset uint64 `json:"offset"`

	couchbase.Cas `json:"-"`
}

func NewCursorsStore(cluster *gocb.Cluster, bucket *gocb.Bucket, scope string) (*couchbase.Couchbase[Cursor], error) {
	collection := bucket.Scope(scope).Collection("cursors")
	store, err := couchbase.NewCouchbase[Cursor](cluster, bucket, collection)
	if err != nil {
		return nil, err
	}

	return store, nil
}

func CursorKey(topic, sub string, shard int) string {
	return fmt.Sprintf("cursor::%s::%s::%d", topic, sub, shard)
}
