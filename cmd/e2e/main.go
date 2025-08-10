package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"os"
	"os/signal"
	"pub/internal/couchbase"
	"pub/internal/pub"
	"pub/internal/pub/producer"
	"runtime"
	"runtime/pprof"
	"syscall"
	"time"

	"github.com/caarlos0/env/v11"
	"github.com/couchbase/gocb/v2"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"golang.org/x/sync/errgroup"

	"pub/internal/pub/consumer"
	"pub/internal/pub/controller"
	"pub/internal/pub/metrics"
	"pub/internal/pub/tracing"
)

type Config struct {
	CouchbaseConnectionString string        `env:"COUCHBASE_CONNECTION_STRING" envDefault:"couchbase://localhost"`
	CouchbaseUsername         string        `env:"COUCHBASE_USERNAME" envDefault:"Administrator"`
	CouchbasePassword         string        `env:"COUCHBASE_PASSWORD" envDefault:"password"`
	CouchbaseBucketName       string        `env:"COUCHBASE_BUCKET_NAME" envDefault:"pubsub"`
	CouchbaseScopeName        string        `env:"COUCHBASE_SCOPE_NAME" envDefault:"default"`
	ConsumerBatchSize         int           `env:"CONSUMER_BATCH_SIZE" envDefault:"50"`
	EventCount                int           `env:"EVENT_COUNT" envDefault:"100"`
	PublishMessagesPerSec     int           `env:"PUBLISH_MESSAGES_PER_SEC" envDefault:"0"`
	PublishRounds             int           `env:"PUBLISH_ROUNDS" envDefault:"1"`
	ConsumerMaxEmptyCount     int           `env:"CONSUMER_MAX_EMPTY_COUNT" envDefault:"2"`
	LogLevel                  string        `env:"LOG_LEVEL" envDefault:"info"`
	MetricsPort               int           `env:"METRICS_PORT" envDefault:"9090"`
	MetricsTimeout            time.Duration `env:"METRICS_TIMEOUT" envDefault:"30s"`
	TracingServiceName        string        `env:"TRACING_SERVICE_NAME" envDefault:"pub-e2e"`
	TracingServiceVersion     string        `env:"TRACING_SERVICE_VERSION" envDefault:"1.0.0"`
	JaegerEndpoint            string        `env:"JAEGER_ENDPOINT" envDefault:"localhost:4318"`
	TracingSampleRate         float64       `env:"TRACING_SAMPLE_RATE" envDefault:"1.0"`
}

func main() {
	cpuProfile, err := os.Create("cpu.pprof")
	if err != nil {
		log.Fatal("could not create CPU profile: ", err)
	}
	defer cpuProfile.Close()
	if err := pprof.StartCPUProfile(cpuProfile); err != nil {
		log.Fatal("could not start CPU profile: ", err)
	}
	defer pprof.StopCPUProfile()

	// Memory Profile
	defer func() {
		memProfile, err := os.Create("mem.pprof")
		if err != nil {
			log.Fatal("could not create memory profile: ", err)
		}
		defer memProfile.Close()
		runtime.GC()
		if err := pprof.WriteHeapProfile(memProfile); err != nil {
			log.Fatal("could not write memory profile: ", err)
		}
	}()

	var cfg Config
	if err := env.Parse(&cfg); err != nil {
		log.Fatalf("failed to parse environment variables: %v", err)
	}

	cluster, bucket, err := newCouchbase(cfg)
	if err != nil {
		log.Fatalf("failed to connect to Couchbase: %v", err)
	}

	config := zap.NewProductionConfig()

	var zapLevel zapcore.Level
	if err := zapLevel.UnmarshalText([]byte(cfg.LogLevel)); err != nil {
		log.Printf("invalid log level %q, defaulting to info: %v", cfg.LogLevel, err)
		zapLevel = zapcore.InfoLevel
	}
	config.Level = zap.NewAtomicLevelAt(zapLevel)
	logger, err := config.Build(zap.AddCaller())
	defer logger.Sync()
	if err != nil {
		log.Fatalf("failed to initialize logger: %v", err)
	}

	metricsRegistry := metrics.NewRegistry()
	metricsRegistry.SetSystemInfo("e2e-test", time.Now().Format(time.RFC3339))

	metricsServer := metrics.NewServer(
		metrics.ServerConfig{
			Port:    cfg.MetricsPort,
			Timeout: cfg.MetricsTimeout,
		},
		metricsRegistry,
		logger,
	)

	go func() {
		if err := metricsServer.Start(context.Background()); err != nil {
			logger.Error("metrics server failed", zap.Error(err))
		}
	}()

	logger.Info("metrics server started",
		zap.String("endpoint", fmt.Sprintf("http://localhost:%d/metrics", cfg.MetricsPort)),
		zap.String("health", fmt.Sprintf("http://localhost:%d/health", cfg.MetricsPort)),
	)

	tracingConfig := tracing.Config{
		ServiceName:    cfg.TracingServiceName,
		ServiceVersion: cfg.TracingServiceVersion,
		JaegerEndpoint: cfg.JaegerEndpoint,
		SampleRate:     cfg.TracingSampleRate,
	}
	tracer, tracingCleanup, err := tracing.NewTracer(tracingConfig)
	if err != nil {
		log.Fatalf("failed to initialize tracing: %v", err)
	}
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := tracingCleanup(shutdownCtx); err != nil {
			logger.Error("failed to cleanup tracing", zap.Error(err))
		}
	}()

	logger.Info("tracing initialized",
		zap.String("service", cfg.TracingServiceName),
		zap.String("jaeger_endpoint", cfg.JaegerEndpoint),
		zap.Float64("sample_rate", cfg.TracingSampleRate),
	)

	cursors, err := pub.NewCursorsStore(cluster, bucket, cfg.CouchbaseScopeName)
	if err != nil {
		log.Fatalf("failed to create cursors store: %v", err)
	}
	leases, err := pub.NewLeasesStore(cluster, bucket, cfg.CouchbaseScopeName)
	if err != nil {
		log.Fatalf("failed to create leases store: %v", err)
	}
	messages, err := pub.NewMessagesStore(cluster, bucket, cfg.CouchbaseScopeName)
	if err != nil {
		log.Fatalf("failed to create messages store: %v", err)
	}
	offsets, err := pub.NewOffsetsStore(cluster, bucket, cfg.CouchbaseScopeName)
	if err != nil {
		log.Fatalf("failed to create offsets store: %v", err)
	}

	transactions, err := couchbase.NewTransactions(cluster)
	if err != nil {
		log.Fatalf("failed to create transactions: %v", err)
	}

	baseController, err := controller.NewController(
		cursors,
		leases,
		messages,
		offsets,
		transactions,
		cfg.CouchbaseBucketName,
		cfg.CouchbaseScopeName,
	)
	if err != nil {
		log.Fatalf("failed to create controller: %v", err)
	}

	metricsController := controller.NewMetricsController(baseController, metricsRegistry)
	ctlr := controller.NewTracedController(metricsController, tracer)

	baseConsumer, err := consumer.NewConsumer(ctlr, logger, cfg.ConsumerBatchSize)
	if err != nil {
		log.Fatalf("failed to create consumer: %v", err)
	}
	metricsConsumer := consumer.NewMetricsConsumer(baseConsumer, metricsRegistry)
	consumer := consumer.NewTracedConsumer(metricsConsumer, tracer)

	baseProducer, err := producer.NewProducer(ctlr, logger)
	if err != nil {
		log.Fatalf("failed to create producer: %v", err)
	}
	metricsProducer := producer.NewMetricsProducer(baseProducer, metricsRegistry)
	producer := producer.NewTracedProducer(metricsProducer, tracer)

	ctx, cancel := context.WithCancel(context.Background())
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		select {
		case <-sig:
			cancel()
		case <-ctx.Done():
		}
	}()

	now := time.Now()
	g, gctx := errgroup.WithContext(ctx)
	g.Go(func() error {
		// default rate of 0 means no rate limiting
		ticker := time.NewTicker(time.Second * max(time.Duration(cfg.PublishMessagesPerSec), 1))
		defer ticker.Stop()
		rounds := 0

		for {
			select {
			case <-gctx.Done():
				return gctx.Err()
			case <-ticker.C:
				topic := "orders"
				e := events(cfg.EventCount)
				if err := producer.PublishBatch(gctx, topic, 0, e...); err != nil {
					logger.Error("failed to publish events", zap.Error(err))
					return fmt.Errorf("failed to publish events: %w", err)
				}
				logger.Info(fmt.Sprintf("published %d events", len(e)))
				rounds++
				if rounds >= cfg.PublishRounds {
					logger.Info("publish rounds complete, stopping producer")
					return nil
				}
			}
		}
	})

	time.Sleep(time.Millisecond * 10)
	for _, sub := range []string{
		"biz-2",
		"orders-3",
		"sales",
		"analytics",
		"marketing",
		"alerts",
		"support",
		"another",
		"test",
	} {
		g.Go(func() error {
			return consume(gctx, logger, consumer, sub, cfg.ConsumerMaxEmptyCount)
		})
	}

	if err := g.Wait(); err != nil {
		logger.Error("error in goroutine", zap.Error(err))
	}

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()
	if err := metricsServer.Stop(shutdownCtx); err != nil {
		logger.Error("failed to stop metrics server", zap.Error(err))
	}

	fmt.Printf("\n\n TEST COMPLETE IN %.2f seconds", time.Since(now).Seconds())
}

func consume(ctx context.Context, logger *zap.Logger, consumer pub.Consumer, sub string, maxEmpty int) error {
	var empty int
	tick := time.NewTicker(100 * time.Millisecond)
	defer tick.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-tick.C:
			pulled, err := consumer.Pull(ctx, "orders", sub, 0)
			if err != nil {
				log.Printf("failed to pull messages: %v", err)
				return err
			}
			switch {
			case pulled == 0:
				empty++
				if empty >= maxEmpty {
					logger.Info("empty message received >= max empty count, stopping consumer")
					return nil
				}
			default:
				empty = 0
			}
		}
	}
}

func events(count int) []pub.Event {
	customers := []string{"A", "B", "C", "D", "E", "F", "G", "H", "I", "J"}
	products := []string{"0", "1", "2", "3", "4", "5", "6", "7", "8", "10"}
	events := make([]pub.Event, 0, count)

	for i := 0; i < count; i++ {
		orderId := fmt.Sprintf("ORD-%04d", i+1)
		customerId := customers[rand.Intn(len(customers))]
		productId := products[rand.Intn(len(products))]
		amount := 10.0 + rand.Float64()*990.0

		pl := map[string]any{
			"order_id":    orderId,
			"customer_id": customerId,
			"product_id":  productId,
			"amount":      amount,
			"timestamp":   time.Now().Format(time.RFC3339),
		}
		e := pub.Event{Type: "order", Payload: pl}

		events = append(events, e)
	}

	return events
}

func newCouchbase(config Config) (*gocb.Cluster, *gocb.Bucket, error) {
	cluster, err := gocb.Connect(config.CouchbaseConnectionString, gocb.ClusterOptions{
		Authenticator: gocb.PasswordAuthenticator{
			Username: config.CouchbaseUsername,
			Password: config.CouchbasePassword,
		},
		TimeoutsConfig: gocb.TimeoutsConfig{
			ConnectTimeout: 10 * time.Second,
			KVTimeout:      5 * time.Second,
			QueryTimeout:   30 * time.Second,
		},
	})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to connect to cluster: %w", err)
	}

	bucket := cluster.Bucket(config.CouchbaseBucketName)

	err = bucket.WaitUntilReady(5*time.Second, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("bucket not ready: %w", err)
	}

	return cluster, bucket, nil
}
