package main

import (
	"context"
	"log"
	"os"

	database "nmap-rest-api/database"
	"nmap-rest-api/router"
	"nmap-rest-api/telemetry"
	"nmap-rest-api/worker"
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
	// TODO: Instead of 5 we can any number of worker coming from config
	worker.StartWorkerPool(5, ctx)

	// HTTP server
	r := router.SetupRouter()
	log.Println("API Server running on :8080")
	r.Run(":8080")
}
