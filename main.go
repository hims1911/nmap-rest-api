package main

import (
	"context"
	"log"
	"os"
	"time"

	businessv1 "nmap-rest-api/business/v1"
	database "nmap-rest-api/database"
	"nmap-rest-api/router"
	"nmap-rest-api/telemetry"

	"go.opentelemetry.io/otel"
)

func main() {
	// Set up tracing first
	telemetry.InitTracer()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer func() {
		defer cancel()
		telemetry.ShutdownTracer(ctx)
	}()

	// Emit test trace span
	func() {
		tracer := otel.Tracer("nmap-api")
		_, span := tracer.Start(ctx, "startup.test")
		defer span.End()
		log.Println("✅ Test span 'startup.test' created")
	}()

	dsn := os.Getenv("DB_DSN")
	if dsn == "" {
		log.Fatal("DB_DSN environment variable is not set")
	}
	database.InitDB(dsn)

	// connecting through redis
	database.InitRedis(ctx)

	// Metrics
	telemetry.InitMetrics(ctx)

	// Start async workers
	businessv1.StartWorkerPool(5)

	// HTTP server
	r := router.SetupRouter()
	log.Println("🚀 API Server running on :8080")
	r.Run(":8080")
}
