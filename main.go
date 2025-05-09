package main

import (
	"context"
	"log"
	"os"

	businessv1 "nmap-rest-api/business/v1"
	database "nmap-rest-api/database"
	"nmap-rest-api/router"
	"nmap-rest-api/telemetry"
)

func main() {
	// Set up tracing first
	telemetry.InitTracer()
	ctx := context.Background()

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
