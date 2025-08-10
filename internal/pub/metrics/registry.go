package metrics

import (
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Registry encapsulates all metrics and provides a clean interface
// for recording metrics without global state
type Registry struct {
	registry *prometheus.Registry

	// Producer metrics
	publishTotal     *prometheus.CounterVec
	publishDuration  *prometheus.HistogramVec
	publishBatchSize *prometheus.HistogramVec

	// Consumer metrics
	consumeTotal     *prometheus.CounterVec
	consumeDuration  *prometheus.HistogramVec
	messagesConsumed *prometheus.CounterVec
	ackTotal         *prometheus.CounterVec
	consumerLag      *prometheus.GaugeVec

	// Controller/Database metrics
	databaseOperationTotal    *prometheus.CounterVec
	databaseOperationDuration *prometheus.HistogramVec

	// Lease metrics
	activeLeases        *prometheus.GaugeVec
	leaseOperationTotal *prometheus.CounterVec

	// System health metrics
	systemInfo        *prometheus.GaugeVec
	startTime         prometheus.Gauge
	connectionsActive prometheus.Gauge
	connectionsIdle   prometheus.Gauge
}

// NewRegistry creates a new metrics registry with all metrics initialized
func NewRegistry() *Registry {
	registry := prometheus.NewRegistry()

	r := &Registry{
		registry: registry,

		// Producer metrics
		publishTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "pub_producer_publish_total",
				Help: "Total number of publish operations",
			},
			[]string{"topic", "shard", "status"}, // status: success, error
		),

		publishDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "pub_producer_publish_duration_seconds",
				Help:    "Time spent publishing batches",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"topic", "shard"},
		),

		publishBatchSize: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "pub_producer_batch_size",
				Help:    "Number of events in published batches",
				Buckets: []float64{1, 5, 10, 25, 50, 100, 250, 500, 1000},
			},
			[]string{"topic", "shard"},
		),

		// Consumer metrics
		consumeTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "pub_consumer_consume_total",
				Help: "Total number of consume operations",
			},
			[]string{"topic", "subscription", "shard", "status"}, // status: success, error, empty
		),

		consumeDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "pub_consumer_consume_duration_seconds",
				Help:    "Time spent consuming messages",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"topic", "subscription", "shard"},
		),

		messagesConsumed: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "pub_consumer_messages_consumed_total",
				Help: "Total number of messages consumed",
			},
			[]string{"topic", "subscription", "shard"},
		),

		ackTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "pub_consumer_ack_total",
				Help: "Total number of message acknowledgments",
			},
			[]string{"topic", "subscription", "shard", "status"}, // status: success, error
		),

		consumerLag: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "pub_consumer_lag",
				Help: "Number of messages behind the latest offset",
			},
			[]string{"topic", "subscription", "shard"},
		),

		// Controller/Database metrics
		databaseOperationTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "pub_database_operation_total",
				Help: "Total number of database operations",
			},
			[]string{"operation", "status"}, // operation: get_offset, commit_offset, insert_message, etc.
		),

		databaseOperationDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "pub_database_operation_duration_seconds",
				Help:    "Time spent on database operations",
				Buckets: []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5},
			},
			[]string{"operation"},
		),

		// Lease metrics
		activeLeases: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "pub_active_leases",
				Help: "Current number of active leases",
			},
			[]string{"subscription"},
		),

		leaseOperationTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "pub_lease_operation_total",
				Help: "Total number of lease operations",
			},
			[]string{"operation", "status"}, // operation: create, delete, expire
		),

		// System health metrics
		systemInfo: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "pub_system_info",
				Help: "System information (value is always 1, labels contain info)",
			},
			[]string{"version", "build_time"},
		),

		startTime: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "pub_start_time_seconds",
				Help: "Unix timestamp when the application started",
			},
		),

		connectionsActive: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "pub_connections_active",
				Help: "Number of active database connections",
			},
		),

		connectionsIdle: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "pub_connections_idle",
				Help: "Number of idle database connections",
			},
		),
	}

	// add default Go metrics (memory, GC, goroutines, etc.)
	registry.MustRegister(collectors.NewGoCollector())
	registry.MustRegister(collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}))

	// Register application metrics
	registry.MustRegister(
		r.publishTotal,
		r.publishDuration,
		r.publishBatchSize,
		r.consumeTotal,
		r.consumeDuration,
		r.messagesConsumed,
		r.ackTotal,
		r.consumerLag,
		r.databaseOperationTotal,
		r.databaseOperationDuration,
		r.activeLeases,
		r.leaseOperationTotal,
		r.systemInfo,
		r.startTime,
		r.connectionsActive,
		r.connectionsIdle,
	)

	// Set start time
	r.startTime.SetToCurrentTime()

	return r
}

// Handler returns an HTTP handler for the Prometheus metrics endpoint
func (r *Registry) Handler() http.Handler {
	return promhttp.HandlerFor(r.registry, promhttp.HandlerOpts{
		EnableOpenMetrics: true,
		Registry:          r.registry,
	})
}

// RecordProducerPublish records a producer publish operation
func (r *Registry) RecordProducerPublish(topic string, shard int, batchSize int, duration time.Duration, err error) {
	shardStr := strconv.Itoa(shard)
	status := "success"
	if err != nil {
		status = "error"
	}

	r.publishTotal.WithLabelValues(topic, shardStr, status).Inc()
	r.publishDuration.WithLabelValues(topic, shardStr).Observe(duration.Seconds())
	if err == nil {
		r.publishBatchSize.WithLabelValues(topic, shardStr).Observe(float64(batchSize))
	}
}

// RecordConsumerPull records a consumer pull operation
func (r *Registry) RecordConsumerPull(topic, subscription string, shard int, messagesConsumed int, duration time.Duration, err error) {
	shardStr := strconv.Itoa(shard)

	status := "success"
	if err != nil {
		status = "error"
	} else if messagesConsumed == 0 {
		status = "empty"
	}

	r.consumeTotal.WithLabelValues(topic, subscription, shardStr, status).Inc()
	r.consumeDuration.WithLabelValues(topic, subscription, shardStr).Observe(duration.Seconds())
	if messagesConsumed > 0 {
		r.messagesConsumed.WithLabelValues(topic, subscription, shardStr).Add(float64(messagesConsumed))
	}
}

// RecordConsumerAck records a consumer acknowledgment operation
func (r *Registry) RecordConsumerAck(topic, subscription string, shard int, err error) {
	status := "success"
	if err != nil {
		status = "error"
	}

	r.ackTotal.WithLabelValues(topic, subscription, strconv.Itoa(shard), status).Inc()
}

// RecordDatabaseOperation records a database operation
func (r *Registry) RecordDatabaseOperation(operation string, duration time.Duration, err error) {
	status := "success"
	if err != nil {
		status = "error"
	}

	r.databaseOperationTotal.WithLabelValues(operation, status).Inc()
	r.databaseOperationDuration.WithLabelValues(operation).Observe(duration.Seconds())
}

// UpdateConsumerLag updates the consumer lag metric
func (r *Registry) UpdateConsumerLag(topic, subscription string, shard int, lag float64) {
	r.consumerLag.WithLabelValues(topic, subscription, strconv.Itoa(shard)).Set(lag)
}

// UpdateActiveLeases updates the active leases metric
func (r *Registry) UpdateActiveLeases(subscription string, count float64) {
	r.activeLeases.WithLabelValues(subscription).Set(count)
}

// RecordLeaseOperation records a lease operation
func (r *Registry) RecordLeaseOperation(operation string, err error) {
	status := "success"
	if err != nil {
		status = "error"
	}

	r.leaseOperationTotal.WithLabelValues(operation, status).Inc()
}

// SetSystemInfo sets system information metrics
func (r *Registry) SetSystemInfo(version, buildTime string) {
	r.systemInfo.WithLabelValues(version, buildTime).Set(1)
}

// UpdateConnectionMetrics updates database connection metrics
func (r *Registry) UpdateConnectionMetrics(active, idle int) {
	r.connectionsActive.Set(float64(active))
	r.connectionsIdle.Set(float64(idle))
}
