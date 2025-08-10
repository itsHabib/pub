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

1. Clone the repository and start the observability stack:

```bash
docker-compose up -d
```

This will start:
- **Couchbase**: Database backend with pre-configured collections
- **Prometheus**: Metrics collection and storage 
- **Grafana**: Metrics visualization dashboards
- **Jaeger**: Distributed tracing collection and UI

2. Build and run the e2e example:

```bash
go run cmd/e2e/main.go
```

3. Access the monitoring dashboards:
   - **Grafana**: http://localhost:3000 (admin/admin)
   - **Prometheus**: http://localhost:9091
   - **Jaeger**: http://localhost:16686 
   - **Couchbase**: http://localhost:8091 (Administrator/password)


## Configuration

The system can be configured using environment variables:

### Core Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `COUCHBASE_CONNECTION_STRING` | `couchbase://localhost` | Couchbase cluster connection string |
| `COUCHBASE_USERNAME` | `Administrator` | Couchbase username |
| `COUCHBASE_PASSWORD` | `password` | Couchbase password |
| `COUCHBASE_BUCKET_NAME` | `pubsub` | Bucket name for storing data |
| `COUCHBASE_SCOPE_NAME` | `default` | Scope name within the bucket |
| `CONSUMER_BATCH_SIZE` | `50` | Number of messages to process per batch |
| `EVENT_COUNT` | `100` | Number of events to publish per round |
| `PUBLISH_MESSAGES_PER_SEC` | `0` | Rate limiting (0 = no limit) |
| `PUBLISH_ROUNDS` | `1` | Number of publishing rounds |
| `CONSUMER_MAX_EMPTY_COUNT` | `2` | Max empty polls before consumer stops |
| `LOG_LEVEL` | `info` | Logging level (debug, info, warn, error) |

### Observability Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `METRICS_PORT` | `9090` | Port for Prometheus metrics endpoint |
| `METRICS_TIMEOUT` | `30s` | HTTP timeout for metrics server |
| `TRACING_SERVICE_NAME` | `pub-e2e` | Service name in distributed traces |
| `TRACING_SERVICE_VERSION` | `1.0.0` | Service version for tracing |
| `JAEGER_ENDPOINT` | `http://localhost:4318` | Jaeger OTLP HTTP endpoint |
| `TRACING_SAMPLE_RATE` | `1.0` | Trace sampling rate (0.0-1.0) |
| `ENABLE_PROFILING` | `false` | Enable CPU and memory profiling |

## Monitoring & Observability

The system provides comprehensive observability through three key areas:

### Metrics (Prometheus + Grafana)
- **Producer metrics**: Publish throughput, batch sizes, error rates
- **Consumer metrics**: Consumption rates, processing latency, queue lag
- **Database metrics**: Operation latency, connection pool status
- **System metrics**: Memory usage, active connections, lease counts

Access metrics at:
- Raw metrics: http://localhost:9090/metrics
- Grafana dashboards: http://localhost:3000

### Distributed Tracing (Jaeger)
- End-to-end request flow visualization
- Performance bottleneck identification  
- Cross-service dependency mapping
- Automatic span creation for all pub/sub operations

Access traces at: http://localhost:16686

### Profiling (pprof)
Enable CPU and memory profiling by setting `ENABLE_PROFILING=true`:
```bash
ENABLE_PROFILING=true go run cmd/e2e/main.go
```
This generates `cpu.pprof` and `mem.pprof` files for performance analysis.

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
