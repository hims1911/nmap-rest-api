package v1

import (
	"context"
	"database/sql"
	"encoding/json"
	"log"
	"strconv"
	"strings"

	database "nmap-rest-api/database"
	models "nmap-rest-api/models/v1"
	"nmap-rest-api/utils"

	"go.opentelemetry.io/otel"
)

var tracer = otel.Tracer("nmap-api")

var QueueScan = queueScan

func queueScan(ctx context.Context, hosts []string) (string, error) {
	scanID := utils.GenerateScanID()
	for _, host := range hosts {
		// setting database status as pending
		err := database.SetScanStatus(scanID, host, "pending")
		if err != nil {
			return "", err
		}

		// creating a job model
		job := models.ScanJob{ScanID: scanID, Host: host}
		jobJSON, err := json.Marshal(job)
		if err != nil {
			log.Println("error receieved while marshaling")
			return "", err
		}

		// creating the tracer
		ctxTracer, span := tracer.Start(ctx, "queue.redis.push")
		defer span.End()

		// pushing it to redis
		errRedis := database.RDB.RPush(ctxTracer, "scan_jobs", jobJSON).Err()
		if errRedis != nil {
			log.Println("error generated while storing scan_jobs to redis")
			// TODO: redis store not getting the value then worker can't pick it
		}
	}
	return scanID, nil
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

func ComputeDiff(host string) models.PortDiff {
	rows, err := database.DB.Query(`
		SELECT open_ports
		FROM scan_results
		WHERE host = $1
		ORDER BY scanned_at DESC
		LIMIT 2
	`, host)
	if err != nil {
		log.Printf("DB error: %v", err)
		return models.PortDiff{Host: host}
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
		return models.PortDiff{Host: host}
	}

	latest, previous := results[0], results[1]
	return models.PortDiff{
		Host:        host,
		NewlyOpened: utils.Diff(latest, previous),
		NewlyClosed: utils.Diff(previous, latest),
	}
}
