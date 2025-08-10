package controller

import (
	"context"
	"time"

	"pub/internal/pub"
	"pub/internal/pub/metrics"
)

// MetricsController wraps a pub.Controller with metrics collection
type MetricsController struct {
	controller pub.Controller
	registry   *metrics.Registry
}

// NewMetricsController creates a new instrumented controller
func NewMetricsController(controller pub.Controller, registry *metrics.Registry) pub.Controller {
	return &MetricsController{
		controller: controller,
		registry:   registry,
	}
}

// GetCursor implements pub.Controller.GetCursor with metrics collection
func (c *MetricsController) GetCursor(ctx context.Context, topic, sub string, shard int) (uint64, error) {
	start := time.Now()

	offset, err := c.controller.GetCursor(ctx, topic, sub, shard)
	duration := time.Since(start)

	c.registry.RecordDatabaseOperation("get_cursor", duration, err)

	return offset, err
}

// CommitCursor implements pub.Controller.CommitCursor with metrics collection
func (c *MetricsController) CommitCursor(topic, sub string, shard int, offset uint64) error {
	start := time.Now()

	err := c.controller.CommitCursor(topic, sub, shard, offset)
	duration := time.Since(start)

	c.registry.RecordDatabaseOperation("commit_cursor", duration, err)

	return err
}

// GetOffset implements pub.Controller.GetOffset with metrics collection
func (c *MetricsController) GetOffset(ctx context.Context, topic string, shard int) (uint64, error) {
	start := time.Now()

	offset, err := c.controller.GetOffset(ctx, topic, shard)
	duration := time.Since(start)

	c.registry.RecordDatabaseOperation("get_offset", duration, err)

	return offset, err
}

// CommitOffset implements pub.Controller.CommitOffset with metrics collection
func (c *MetricsController) CommitOffset(topic string, shard int, currentOffset uint64) error {
	start := time.Now()

	err := c.controller.CommitOffset(topic, shard, currentOffset)
	duration := time.Since(start)

	c.registry.RecordDatabaseOperation("commit_offset", duration, err)

	return err
}

// InsertLease implements pub.Controller.InsertLease with metrics collection
func (c *MetricsController) InsertLease(ctx context.Context, sub string, msgID string, offset uint64) error {
	start := time.Now()

	err := c.controller.InsertLease(ctx, sub, msgID, offset)
	duration := time.Since(start)

	c.registry.RecordDatabaseOperation("insert_lease", duration, err)
	c.registry.RecordLeaseOperation("create", err)

	return err
}

// DeleteLease implements pub.Controller.DeleteLease with metrics collection
func (c *MetricsController) DeleteLease(ctx context.Context, sub string, msgID string) error {
	start := time.Now()

	err := c.controller.DeleteLease(ctx, sub, msgID)
	duration := time.Since(start)

	c.registry.RecordDatabaseOperation("delete_lease", duration, err)
	c.registry.RecordLeaseOperation("delete", err)

	return err
}

// InsertMessage implements pub.Controller.InsertMessage with metrics collection
func (c *MetricsController) InsertMessage(ctx context.Context, msg pub.Message) error {
	start := time.Now()

	err := c.controller.InsertMessage(ctx, msg)
	duration := time.Since(start)

	c.registry.RecordDatabaseOperation("insert_message", duration, err)

	return err
}

// LoadMessages implements pub.Controller.LoadMessages with metrics collection
func (c *MetricsController) LoadMessages(ctx context.Context, topic string, shard int, fromOffset uint64, limit int) ([]pub.Message, error) {
	start := time.Now()

	messages, err := c.controller.LoadMessages(ctx, topic, shard, fromOffset, limit)
	duration := time.Since(start)

	c.registry.RecordDatabaseOperation("load_messages", duration, err)

	return messages, err
}
