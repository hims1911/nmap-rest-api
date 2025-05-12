package telemetry

import (
	"context"
	"log"
	"os"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
)

var (
	Meter         metric.Meter
	ScanCounter   metric.Int64Counter
	ScanFailures  metric.Int64Counter
	ScanHistogram metric.Float64Histogram

	// New metrics for worker and scan queue
	WorkerActive     metric.Int64UpDownCounter
	WorkerIdle       metric.Int64UpDownCounter
	ScanQueueLength  metric.Int64ObservableGauge
)

func InitMetrics(ctx context.Context, redisQueueLengthFunc func() int64) {

	endpoint := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")
	if endpoint == "" {
		endpoint = "otel-collector:4318" // default fallback
	}
	exporter, err := otlpmetrichttp.New(ctx,
		otlpmetrichttp.WithEndpoint(endpoint),
		otlpmetrichttp.WithInsecure(),
	)
	if err != nil {
		log.Fatalf("Failed to create OTLP metrics exporter: %v", err)
	}

	provider := sdkmetric.NewMeterProvider(
		sdkmetric.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName("nmap-api"),
		)),
		sdkmetric.WithReader(sdkmetric.NewPeriodicReader(exporter)),
	)

	otel.SetMeterProvider(provider)

	Meter = provider.Meter("nmap-api")

	ScanCounter, _ = Meter.Int64Counter("nmap_scans_total")
	ScanFailures, _ = Meter.Int64Counter("nmap_scan_failures_total")
	ScanHistogram, _ = Meter.Float64Histogram("nmap_scan_duration_seconds")

	// New metrics
	WorkerActive, _ = Meter.Int64UpDownCounter("worker_active_total")
	WorkerIdle, _ = Meter.Int64UpDownCounter("worker_idle_total")
	ScanQueueLength, _ = Meter.Int64ObservableGauge("scan_queue_length")

	// Redis queue length gauge
	Meter.RegisterCallback(
		func(ctx context.Context, observer metric.Observer) error {
			observer.ObserveInt64(ScanQueueLength, redisQueueLengthFunc())
			return nil
		},
		ScanQueueLength,
	)

	log.Println("OpenTelemetry metrics via OTLP configured")
}
