package pub

import (
	"fmt"
	"time"

	"github.com/couchbase/gocb/v2"

	"pub/internal/couchbase"
)

type Message struct {
	ID          string     `json:"id"`
	Topic       string     `json:"topic"`
	Shard       int        `json:"shard"`
	Offset      uint64     `json:"offset"`
	Event       string     `json:"event"`
	Payload     any        `json:"payload"`
	PublishTime *time.Time `json:"publishTime,omitempty"`

	couchbase.Cas `json:"-"`
}

func NewMessagesStore(cluster *gocb.Cluster, bucket *gocb.Bucket, scope string) (*couchbase.Couchbase[Message], error) {
	collection := bucket.Scope(scope).Collection("messages")
	store, err := couchbase.NewCouchbase[Message](cluster, bucket, collection)
	if err != nil {
		return nil, err
	}

	return store, nil
}

func MessageKey(topic string, shard int, offset uint64) string {
	return fmt.Sprintf("message::%s::%d::%d", topic, shard, offset)
}
