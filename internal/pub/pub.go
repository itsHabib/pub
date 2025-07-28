package pub

import "fmt"

func ReceiptKey(topic string, shard int, offset uint64) string {
	return fmt.Sprintf("receipt::%s::%d::%d", topic, shard, offset)
}
