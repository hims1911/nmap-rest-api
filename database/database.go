package databse

import (
	"database/sql"
	"fmt"
	"log"
	"strings"
	"time"

	models "nmap-rest-api/models/v1"

	_ "github.com/lib/pq"
)

var DB *sql.DB

func InitDB(dsn string) {
	var err error

	for i := 0; i < 10; i++ {
		DB, err = sql.Open("postgres", dsn)
		if err != nil {
			log.Printf("Attempt %d: Failed to open DB: %v", i+1, err)
		} else if err = DB.Ping(); err == nil {
			log.Println("✅ Connected to Postgres")
			return
		} else {
			log.Printf("Attempt %d: Waiting for DB to be ready...", i+1)
		}
		time.Sleep(2 * time.Second)
	}

	log.Fatalf("❌ Could not connect to DB after retries: %v", err)
}

func StoreResult(res models.ScanResult) error {
	_, err := DB.Exec(`INSERT INTO scan_results (scan_id, host, scanned_at, open_ports) VALUES ($1, $2, $3, $4)`,
		res.ScanID,
		res.Host,
		res.ScannedAt,
		fmt.Sprintf("{%s}", strings.Trim(strings.Join(strings.Fields(fmt.Sprint(res.OpenPorts)), ","), "[]")),
	)
	return err
}

func SetScanStatus(scanID, host, status string) error {
	query := `
		INSERT INTO scan_status (scan_id, host, status, started_at)
		VALUES ($1, $2, $3, CASE WHEN $3 = 'in_progress' THEN now() ELSE NULL END)
		ON CONFLICT (scan_id, host)
		DO UPDATE SET 
			status = $3,
			started_at = CASE WHEN $3 = 'in_progress' THEN now() ELSE scan_status.started_at END,
			completed_at = CASE WHEN $3 = 'done' OR $3 = 'failed' THEN now() ELSE scan_status.completed_at END
	`
	_, err := DB.Exec(query, scanID, host, status)
	if err != nil {
		log.Printf("Failed to update scan_status: %v", err)
	}
	return err
}

func GetScanStatuses(scanID string) ([]map[string]string, error) {
	rows, err := DB.Query(`SELECT host, status FROM scan_status WHERE scan_id = $1`, scanID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var statuses []map[string]string
	for rows.Next() {
		var host, status string
		if err := rows.Scan(&host, &status); err == nil {
			statuses = append(statuses, map[string]string{
				"host":   host,
				"status": status,
			})
		}
	}
	return statuses, nil
}
