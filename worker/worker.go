package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"nmap-rest-api/models/v1"
	"nmap-rest-api/telemetry"
	"os/exec"
	"strings"
	"time"

	database "nmap-rest-api/database"

	"go.opentelemetry.io/otel"
)

var tracer = otel.Tracer("nmap-api")

func StartWorkerPool(concurrency int, ctx context.Context) {
	for i := 0; i < concurrency; i++ {
		go func() {
			for {
				telemetry.WorkerIdle.Add(ctx, -1)
				telemetry.WorkerActive.Add(ctx, 1)
				ctxRedis, span := tracer.Start(ctx, "worker.redis.pop")
				val, err := database.RDB.BLPop(ctxRedis, 0, "scan_jobs").Result()
				span.End()
				if err != nil || len(val) < 2 {
					log.Printf("Failed to pop job: %v", err)
					continue
				}

				var job models.ScanJob
				if err := json.Unmarshal([]byte(val[1]), &job); err != nil {
					log.Printf("Invalid job format: %v", err)
					continue
				}

				database.SetScanStatus(job.ScanID, job.Host, "in_progress")
				start := time.Now()
				_, span = tracer.Start(ctx, "nmap.run")
				span.End()

				ports := runNmap(ctx, job.Host)
				res := models.ScanResult{
					ScanID:    job.ScanID,
					Host:      job.Host,
					ScannedAt: time.Now(),
					OpenPorts: ports,
				}

				_, span = tracer.Start(ctx, "db.store_result")
				errDatabase := database.StoreResult(res)
				span.End()

				duration := time.Since(start).Seconds()

				telemetry.ScanCounter.Add(ctx, 1)
				telemetry.ScanHistogram.Record(ctx, duration)

				if errDatabase != nil {
					log.Println("DB error:", errDatabase)
					telemetry.ScanFailures.Add(ctx, 1)
					database.SetScanStatus(job.ScanID, job.Host, "failed")
				} else {
					log.Println("Scan result stored")
					database.SetScanStatus(job.ScanID, job.Host, "done")
				}
				telemetry.WorkerActive.Add(ctx, -1)
				telemetry.WorkerIdle.Add(ctx, 1)
			}
		}()
	}
}

// runNmap will run the nmap
func runNmap(_ context.Context, host string) []int {
	log.Println("nmap function has been called for ", host)
	scanType := "-sT"
	cmd := exec.Command("nmap", "-Pn", scanType, "--max-retries", "2", host)
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("nmap error for %s: %v\nOutput: %s", host, err, output)
		return nil
	}
	log.Println("nmap function has been executed successfully ", host)

	var openPorts []int
	for _, line := range strings.Split(string(output), "\n") {
		if strings.Contains(line, "/tcp") && strings.Contains(line, "open") {
			var port int
			fmt.Sscanf(strings.Fields(line)[0], "%d/tcp", &port)
			openPorts = append(openPorts, port)
		}
	}
	return openPorts
}
