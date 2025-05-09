package v1

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os/exec"
	"strconv"
	"strings"
	"time"

	database "nmap-rest-api/database"
	models "nmap-rest-api/models/v1"
	"nmap-rest-api/telemetry"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel"
)

type PortDiff struct {
	Host        string `json:"host"`
	NewlyOpened []int  `json:"newly_opened"`
	NewlyClosed []int  `json:"newly_closed"`
}

var tracer = otel.Tracer("nmap-api")

func QueueScan(hosts []string) string {
	scanID := generateScanID()
	for _, host := range hosts {
		database.SetScanStatus(scanID, host, "pending")
		job := models.ScanJob{ScanID: scanID, Host: host}
		jobJSON, _ := json.Marshal(job)
		ctx, span := tracer.Start(context.Background(), "queue.redis.push")
		defer span.End()
		_ = database.RDB.RPush(ctx, "scan_jobs", jobJSON).Err()
	}
	return scanID
}

func StartWorkerPool(concurrency int, ctx context.Context) {
	for i := 0; i < concurrency; i++ {
		go func() {
			for {
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
					log.Println("❌ DB error:", errDatabase)
					telemetry.ScanFailures.Add(ctx, 1)
					database.SetScanStatus(job.ScanID, job.Host, "failed")
				} else {
					log.Println("✅ Scan result stored")
					database.SetScanStatus(job.ScanID, job.Host, "done")
				}
			}
		}()
	}
}

func FetchScanHistoryFiltered(host string, scanID string) []models.ScanResult {
	var (
		rows *sql.Rows
		err  error
	)

	if scanID != "" {
		rows, err = database.DB.Query(`
			SELECT scan_id, host, scanned_at, open_ports
			FROM scan_results
			WHERE host = $1 AND scan_id = $2
			ORDER BY scanned_at DESC
		`, host, scanID)
	} else {
		rows, err = database.DB.Query(`
			SELECT scan_id, host, scanned_at, open_ports
			FROM scan_results
			WHERE host = $1
			ORDER BY scanned_at DESC
			LIMIT 10
		`, host)
	}

	if err != nil {
		log.Printf("DB query failed: %v", err)
		return nil
	}
	defer rows.Close()

	var results []models.ScanResult
	for rows.Next() {
		var res models.ScanResult
		var portsRaw string
		if err := rows.Scan(&res.ScanID, &res.Host, &res.ScannedAt, &portsRaw); err == nil {
			portsStr := strings.Trim(portsRaw, "{}")
			for _, p := range strings.Split(portsStr, ",") {
				if port, err := strconv.Atoi(p); err == nil {
					res.OpenPorts = append(res.OpenPorts, port)
				}
			}
			results = append(results, res)
		}
	}
	return results
}

func ComputeDiff(host string) PortDiff {
	rows, err := database.DB.Query(`
		SELECT open_ports
		FROM scan_results
		WHERE host = $1
		ORDER BY scanned_at DESC
		LIMIT 2
	`, host)
	if err != nil {
		log.Printf("DB error: %v", err)
		return PortDiff{Host: host}
	}
	defer rows.Close()

	var results [][]int
	for rows.Next() {
		var portsRaw string
		if err := rows.Scan(&portsRaw); err == nil {
			var parsed []int
			for _, p := range strings.Split(strings.Trim(portsRaw, "{}"), ",") {
				if port, err := strconv.Atoi(p); err == nil {
					parsed = append(parsed, port)
				}
			}
			results = append(results, parsed)
		}
	}

	if len(results) < 2 {
		return PortDiff{Host: host}
	}

	latest, previous := results[0], results[1]
	return PortDiff{
		Host:        host,
		NewlyOpened: diff(latest, previous),
		NewlyClosed: diff(previous, latest),
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

func generateScanID() string {
	return uuid.New().String()
}

func diff(a, b []int) []int {
	m := make(map[int]bool)
	for _, v := range b {
		m[v] = true
	}
	var result []int
	for _, v := range a {
		if !m[v] {
			result = append(result, v)
		}
	}
	return result
}
