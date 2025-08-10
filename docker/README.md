# Monitoring Stack Setup

This directory contains the complete monitoring stack for the pub/sub system with Prometheus and Grafana.

## Services

- **Couchbase**: Database backend (ports 8091-8096, 11210)
- **Prometheus**: Metrics collection and storage (port 9091)
- **Grafana**: Metrics visualization and dashboards (port 3000)
- **Jaeger**: Distributed tracing (port 16686)
- **Loki**: Log aggregation and search (port 3100)
- **Promtail**: Log collection agent

## Quick Start

1. **Start the monitoring stack:**
   ```bash
   docker-compose up -d
   ```

2. **Run the e2e test with metrics:**
   ```bash
   go run ./cmd/e2e
   ```

3. **Access the dashboards:**
   - Grafana: http://localhost:3000 (admin/admin)
   - Prometheus: http://localhost:9091 (metrics)
   - Jaeger: http://localhost:16686 (traces)
   - Loki: http://localhost:3100 (logs)
   - Application metrics: http://localhost:9090/metrics

## Services Overview

### Prometheus (port 9091)
- Scrapes metrics from your application on `host.docker.internal:9090`
- Stores metrics with 200h retention
- Configuration: `docker/prometheus.yml`

### Grafana (port 3000)
- Pre-configured with Prometheus datasource
- Auto-loads the "Pub/Sub System Metrics" dashboard
- Default credentials: admin/admin

### Jaeger (port 16686)
- Distributed tracing UI
- Shows request flows and performance bottlenecks
- Collects traces from your application via OTLP
- Configuration: Automatic trace collection from `host.docker.internal:14268`

### Loki (port 3100)
- Log aggregation system
- Indexes logs by labels, not content (efficient storage)
- Provides LogQL query language for log search
- Integrates with Grafana for log visualization
- Configuration: `docker/loki-config.yml`

### Promtail
- Log collection agent
- Ships logs from containers to Loki
- Extracts structured fields from JSON logs
- Adds metadata labels for filtering
- Configuration: `docker/promtail-config.yml`

### Application Metrics (port 9090)
Your pub/sub application exposes metrics on port 9090:
- `/metrics` - Prometheus metrics endpoint
- `/health` - Health check endpoint
- `/ready` - Readiness check endpoint

## Dashboard Features

The included Grafana dashboard shows:

### System Overview
- **Throughput**: Messages published and consumed per second
- **Latency**: P95 latencies for publish, consume, and database operations

### Producer Metrics
- **Publish Rate**: Success vs error rates
- **Batch Size Distribution**: P50 and P95 batch sizes
- **Error Rate**: Producer error percentage with alerting threshold

### Consumer Metrics
- **Consumer Lag**: Messages behind latest offset per subscription
- **Consumer Operations**: Success, error, and empty pull rates

### Database & System
- **Database Operations**: Operation rates by type and status
- **Memory Usage**: Go runtime memory metrics

## Observability Stack Features

The complete stack provides three pillars of observability:

### 📊 Metrics (Prometheus + Grafana)
- **Producer**: Publish rates, batch sizes, latencies, error rates
- **Consumer**: Consume rates, processing times, lag, acknowledgments
- **Database**: Operation latencies, error rates, connection pools
- **System**: Memory, GC, goroutines, uptime

### 🔍 Traces (Jaeger)
- **Request Flow**: See exactly how requests flow through components
- **Performance Analysis**: Identify bottlenecks in publish/consume operations
- **Error Debugging**: Trace exactly where and why operations fail
- **Dependencies**: Understand service interactions and database call patterns

### 📋 Logs (Structured)
- **Zap Integration**: Structured logging with trace correlation
- **Error Context**: Rich error information with traces
- **Debug Information**: Detailed operation logs with request IDs

## Tracing Features

Your pub/sub application now includes comprehensive distributed tracing:

### Producer Traces
```
span: producer.publish_batch (500ms)
├── span: controller.get_offset (50ms)
├── span: controller.insert_message (300ms) ← Bottleneck!
├── span: controller.insert_message (50ms)  
└── span: controller.commit_offset (100ms)
```

### Consumer Traces  
```
span: consumer.pull (200ms)
├── span: controller.get_cursor (20ms)
├── span: controller.load_messages (80ms)
├── span: controller.insert_lease (30ms)
└── span: consumer.ack (70ms)
    ├── span: controller.delete_lease (30ms)
    └── span: controller.commit_cursor (40ms)
```

### Tracing Environment Variables
```bash
TRACING_SERVICE_NAME=pub-e2e        # Service name in traces
TRACING_SERVICE_VERSION=1.0.0       # Service version
JAEGER_ENDPOINT=http://localhost:4318  # Jaeger OTLP HTTP endpoint
TRACING_SAMPLE_RATE=1.0             # Sample rate (0.0-1.0)
```

## Log Search with Loki

### Accessing Logs in Grafana
1. Open Grafana: http://localhost:3000
2. Go to "Explore" (compass icon)
3. Select "Loki" as datasource
4. Use LogQL to search logs

### LogQL Query Examples

**Basic log filtering:**
```logql
{container_name="pub-e2e"}
```

**Filter by log level:**
```logql
{container_name="pub-e2e"} | json | level="ERROR"
```

**Search by message content:**
```logql
{container_name="pub-e2e"} |= "failed to publish"
```

**Find logs for specific trace:**
```logql
{container_name="pub-e2e"} | json | trace_id="abc123..."
```

**Show logs with topic and shard:**
```logql
{container_name="pub-e2e"} | json | topic="orders" | shard="0"
```

**Logs around a specific time:**
```logql
{container_name="pub-e2e"} | json | level="ERROR" | __error__=""
```

### Log Correlation
- **From Metrics**: Click anomaly in Grafana → View logs for that time range
- **From Traces**: Copy trace ID in Jaeger → Search logs by trace_id
- **From Logs**: Click trace_id → Jump to Jaeger trace view

### Live Log Tailing
Click the "Live" button in Grafana Explore to watch logs stream in real-time.

## Configuration

### Prometheus Targets
Edit `docker/prometheus.yml` to add more application instances:

```yaml
scrape_configs:
  - job_name: 'pub-producer-1'
    static_configs:
      - targets: ['host.docker.internal:9090']
  
  - job_name: 'pub-consumer-1'  
    static_configs:
      - targets: ['host.docker.internal:9091']
```

### Application Metrics Port
Configure the metrics port via environment variable:
```bash
METRICS_PORT=8080 go run ./cmd/e2e
```

### Grafana Customization
- Dashboards: Place `.json` files in `docker/grafana/dashboards/`
- Data sources: Configure in `docker/grafana/provisioning/datasources/`
- Settings: Configure in `docker/grafana/provisioning/dashboards/`

## Alerting (Future)

The dashboard includes threshold configurations for:
- High error rates (>5%)
- High latency (>1s P95)
- Consumer lag (>1000 messages)

To enable alerting:
1. Configure Grafana notification channels
2. Set up alert rules based on the threshold queries
3. Add Prometheus AlertManager for advanced routing

## Troubleshooting

### Application metrics not showing
1. Ensure your app is running and exposing metrics on port 9090
2. Check Prometheus targets: http://localhost:9091/targets
3. Verify network connectivity to `host.docker.internal:9090`

### Dashboard not loading
1. Check Grafana logs: `docker logs grafana`
2. Verify dashboard files are in `docker/grafana/dashboards/`
3. Check Grafana provisioning: http://localhost:3000/admin/provisioning

### Data retention
- Prometheus retains data for 200h (configurable in docker-compose.yml)
- Grafana stores dashboards and settings in named volumes
- To reset: `docker-compose down -v`

## Performance Monitoring

Key metrics to watch for production:

**Autoscaling Triggers:**
- `pub_consumer_lag > 1000` → Scale up consumers
- `rate(pub_producer_publish_total{status="error"}[5m]) > 0.05` → Investigate errors
- `histogram_quantile(0.95, rate(pub_database_operation_duration_seconds_bucket[5m])) > 1.0` → Database performance issues

**Health Indicators:**
- Consistent throughput rates
- Low error rates (<1%)
- Stable memory usage
- Consumer lag near zero