# Pub - Distributed Pub/Sub System

A distributed publish-subscribe messaging system built with Go and Couchbase,
 designed for reliable message delivery with at-least-once semantics.

## Architecture

The system is built around four core components:

### Core Components

- **Producer**: Publishes events to topics with automatic sharding
- **Consumer**: Subscribes to topics and processes messages with lease-based acknowledgment
- **Controller**: Manages message persistence, cursors, offsets, and leases
- **Storage Layer**: Couchbase-based persistent storage with collections for different data types

### Data Model

- **Messages**: Events published to topics, stored with metadata and offsets
- **Cursors**: Track consumption progress per subscription and shard
- **Offsets**: Maintain the current write position for each topic shard
- **Leases**: Temporary locks on messages to ensure at-least-once delivery

## Quick Start

### Prerequisites

- Go 1.24 or later
- Docker and Docker Compose

### Running with Docker

1. Clone the repository and start the Couchbase container:

```bash
docker-compose up -d
```

This will:
- Start a Couchbase server with pre-configured collections
- Create the `pubsub` bucket with required collections (`messages`, `cursors`, `offsets`, `leases`)
- Set up default credentials (Username: `Administrator`, Password: `password`)

2. Build and run the e2e example:

```bash
go run cmd/e2e/main.go
```


## Configuration

The system can be configured using environment variables:

| Variable | Default | Description |
|----------|---------|-------------|
| `COUCHBASE_CONNECTION_STRING` | `couchbase://localhost` | Couchbase cluster connection string |
| `COUCHBASE_USERNAME` | `Administrator` | Couchbase username |
| `COUCHBASE_PASSWORD` | `password` | Couchbase password |
| `COUCHBASE_BUCKET_NAME` | `pubsub` | Bucket name for storing data |
| `COUCHBASE_SCOPE_NAME` | `default` | Scope name within the bucket |
| `CONSUMER_BATCH_SIZE` | `50` | Number of messages to process per batch |
| `EVENT_COUNT` | `1000` | Number of events to publish per round |
| `PUBLISH_MESSAGES_PER_SEC` | `0` | Rate limiting (0 = no limit) |
| `PUBLISH_ROUNDS` | `1` | Number of publishing rounds |
| `CONSUMER_MAX_EMPTY_COUNT` | `2` | Max empty polls before consumer stops |
| `LOG_LEVEL` | `info` | Logging level (debug, info, warn, error) |

## Usage Examples

### Basic Producer

```go
package main

import (
    "context"
    "pub/internal/pub"
    "pub/internal/pub/producer"
    "pub/internal/pub/controller"
)

func main() {
    // Setup controller (see full example in cmd/e2e/main.go)
    ctrl, err := controller.NewController(/* ... */)
    if err != nil {
        log.Fatal(err)
    }

    // Create producer
    producer, err := producer.NewProducer(ctrl, logger)
    if err != nil {
        log.Fatal(err)
    }

    // Publish events
    events := []pub.Event{
        {Type: "order.created", Payload: map[string]interface{}{"id": "123"}},
        {Type: "order.updated", Payload: map[string]interface{}{"id": "123", "status": "shipped"}},
    }

    err = producer.PublishBatch(context.Background(), "orders", 0, events...)
    if err != nil {
        log.Fatal(err)
    }
}
```

### Basic Consumer

```go
package main

import (
    "context"
    "pub/internal/pub/consumer"
    "pub/internal/pub/controller"
)

func main() {
    // Setup controller (see full example in cmd/e2e/main.go)
    ctrl, err := controller.NewController(/* ... */)
    if err != nil {
        log.Fatal(err)
    }

    // Create consumer
    consumer, err := consumer.NewConsumer(ctrl, logger, 50)
    if err != nil {
        log.Fatal(err)
    }

    // Start consuming
    for {
        count, err := consumer.Pull(context.Background(), "orders", "my-subscription", 0)
        if err != nil {
            log.Printf("Error consuming: %v", err)
            break
        }
        
        if count == 0 {
            // No messages available
            time.Sleep(time.Second)
        }
    }
}
```
