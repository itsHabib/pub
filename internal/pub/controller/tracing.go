package controller

import (
	"context"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"

	"pub/internal/pub"
	"pub/internal/pub/tracing"
)

// TracedController wraps a pub.Controller with distributed tracing
// Layer order: TracedController -> MetricsController -> Controller (real thing)
type TracedController struct {
	controller pub.Controller
	tracer     *tracing.Tracer
}

// NewTracedController creates a new traced controller that wraps a metrics controller
func NewTracedController(controller pub.Controller, tracer *tracing.Tracer) pub.Controller {
	return &TracedController{
		controller: controller,
		tracer:     tracer,
	}
}

// GetCursor implements pub.Controller.GetCursor with distributed tracing
func (c *TracedController) GetCursor(ctx context.Context, topic, sub string, shard int) (uint64, error) {
	ctx, span := c.tracer.StartSpan(ctx, "controller.get_cursor")
	defer span.End()

	span.SetAttributes(
		c.tracer.DatabaseAttributes("get_cursor")...,
	)
	span.SetAttributes(
		attribute.String("pub.topic", topic),
		attribute.String("pub.subscription", sub),
		attribute.Int("pub.shard", shard),
	)

	offset, err := c.controller.GetCursor(ctx, topic, sub, shard)

	if err != nil {
		c.tracer.RecordError(ctx, err)
	} else {
		span.SetStatus(codes.Ok, "")
		span.SetAttributes(attribute.Int64("pub.cursor_offset", int64(offset)))
	}

	span.SetAttributes(c.tracer.ErrorAttributes(err)...)
	return offset, err
}

// CommitCursor implements pub.Controller.CommitCursor with distributed tracing
func (c *TracedController) CommitCursor(topic, sub string, shard int, offset uint64) error {
	ctx, span := c.tracer.StartSpan(context.Background(), "controller.commit_cursor")
	defer span.End()

	span.SetAttributes(
		c.tracer.DatabaseAttributes("commit_cursor")...,
	)
	span.SetAttributes(
		attribute.String("pub.topic", topic),
		attribute.String("pub.subscription", sub),
		attribute.Int("pub.shard", shard),
		attribute.Int64("pub.cursor_offset", int64(offset)),
	)

	err := c.controller.CommitCursor(topic, sub, shard, offset)

	if err != nil {
		c.tracer.RecordError(ctx, err)
	} else {
		span.SetStatus(codes.Ok, "")
	}

	span.SetAttributes(c.tracer.ErrorAttributes(err)...)
	return err
}

// GetOffset implements pub.Controller.GetOffset with distributed tracing
func (c *TracedController) GetOffset(ctx context.Context, topic string, shard int) (uint64, error) {
	ctx, span := c.tracer.StartSpan(ctx, "controller.get_offset")
	defer span.End()

	span.SetAttributes(
		c.tracer.DatabaseAttributes("get_offset")...,
	)
	span.SetAttributes(
		attribute.String("pub.topic", topic),
		attribute.Int("pub.shard", shard),
	)

	offset, err := c.controller.GetOffset(ctx, topic, shard)

	if err != nil {
		c.tracer.RecordError(ctx, err)
	} else {
		span.SetStatus(codes.Ok, "")
		span.SetAttributes(attribute.Int64("pub.write_offset", int64(offset)))
	}

	span.SetAttributes(c.tracer.ErrorAttributes(err)...)
	return offset, err
}

// CommitOffset implements pub.Controller.CommitOffset with distributed tracing
func (c *TracedController) CommitOffset(topic string, shard int, currentOffset uint64) error {
	ctx, span := c.tracer.StartSpan(context.Background(), "controller.commit_offset")
	defer span.End()

	span.SetAttributes(
		c.tracer.DatabaseAttributes("commit_offset")...,
	)
	span.SetAttributes(
		attribute.String("pub.topic", topic),
		attribute.Int("pub.shard", shard),
		attribute.Int64("pub.write_offset", int64(currentOffset)),
	)

	err := c.controller.CommitOffset(topic, shard, currentOffset)

	if err != nil {
		c.tracer.RecordError(ctx, err)
	} else {
		span.SetStatus(codes.Ok, "")
	}

	span.SetAttributes(c.tracer.ErrorAttributes(err)...)
	return err
}

// InsertLease implements pub.Controller.InsertLease with distributed tracing
func (c *TracedController) InsertLease(ctx context.Context, sub string, msgID string, offset uint64) error {
	ctx, span := c.tracer.StartSpan(ctx, "controller.insert_lease")
	defer span.End()

	span.SetAttributes(
		c.tracer.DatabaseAttributes("insert_lease")...,
	)
	span.SetAttributes(
		attribute.String("pub.subscription", sub),
		attribute.String("pub.message_id", msgID),
		attribute.Int64("pub.message_offset", int64(offset)),
	)

	err := c.controller.InsertLease(ctx, sub, msgID, offset)

	if err != nil {
		c.tracer.RecordError(ctx, err)
	} else {
		span.SetStatus(codes.Ok, "")
	}

	span.SetAttributes(c.tracer.ErrorAttributes(err)...)
	return err
}

// DeleteLease implements pub.Controller.DeleteLease with distributed tracing
func (c *TracedController) DeleteLease(ctx context.Context, sub string, msgID string) error {
	ctx, span := c.tracer.StartSpan(ctx, "controller.delete_lease")
	defer span.End()

	span.SetAttributes(
		c.tracer.DatabaseAttributes("delete_lease")...,
	)
	span.SetAttributes(
		attribute.String("pub.subscription", sub),
		attribute.String("pub.message_id", msgID),
	)

	err := c.controller.DeleteLease(ctx, sub, msgID)

	if err != nil {
		c.tracer.RecordError(ctx, err)
	} else {
		span.SetStatus(codes.Ok, "")
	}

	span.SetAttributes(c.tracer.ErrorAttributes(err)...)
	return err
}

// InsertMessage implements pub.Controller.InsertMessage with distributed tracing
func (c *TracedController) InsertMessage(ctx context.Context, msg pub.Message) error {
	ctx, span := c.tracer.StartSpan(ctx, "controller.insert_message")
	defer span.End()

	span.SetAttributes(
		c.tracer.DatabaseAttributes("insert_message")...,
	)
	span.SetAttributes(
		attribute.String("pub.topic", msg.Topic),
		attribute.Int("pub.shard", msg.Shard),
		attribute.String("pub.message_id", msg.ID),
		attribute.Int64("pub.message_offset", int64(msg.Offset)),
		attribute.String("pub.event_type", msg.Event),
	)

	err := c.controller.InsertMessage(ctx, msg)

	if err != nil {
		c.tracer.RecordError(ctx, err)
	} else {
		span.SetStatus(codes.Ok, "")
	}

	span.SetAttributes(c.tracer.ErrorAttributes(err)...)
	return err
}

// LoadMessages implements pub.Controller.LoadMessages with distributed tracing
func (c *TracedController) LoadMessages(ctx context.Context, topic string, shard int, fromOffset uint64, limit int) ([]pub.Message, error) {
	ctx, span := c.tracer.StartSpan(ctx, "controller.load_messages")
	defer span.End()

	span.SetAttributes(
		c.tracer.DatabaseAttributes("load_messages")...,
	)
	span.SetAttributes(
		attribute.String("pub.topic", topic),
		attribute.Int("pub.shard", shard),
		attribute.Int64("pub.from_offset", int64(fromOffset)),
		attribute.Int("pub.limit", limit),
	)

	messages, err := c.controller.LoadMessages(ctx, topic, shard, fromOffset, limit)

	if err != nil {
		c.tracer.RecordError(ctx, err)
	} else {
		span.SetStatus(codes.Ok, "")
		span.SetAttributes(attribute.Int("pub.messages_loaded", len(messages)))
	}

	span.SetAttributes(c.tracer.ErrorAttributes(err)...)
	return messages, err
}
