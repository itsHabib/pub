package pub

import "fmt"

// ReceiptKey generates a unique key for message receipts in Couchbase.
// Receipts can be used to track acknowledgment or processing status of messages.
// The key format is "receipt::<topic>::<shard>::<offset>" to align with message keys.
func ReceiptKey(topic string, shard int, offset uint64) string {
	return fmt.Sprintf("receipt::%s::%d::%d", topic, shard, offset)
}
