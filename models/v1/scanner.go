package models

import "time"

type ScanRequest struct {
	Hosts []string `json:"hosts"`
}

type ScanResult struct {
	ScanID    string    `json:"scan_id"`
	Host      string    `json:"host"`
	ScannedAt time.Time `json:"scanned_at"`
	OpenPorts []int     `json:"open_ports"`
}

type PortDiff struct {
	Host        string `json:"host"`
	NewlyOpened []int  `json:"newly_opened"`
	NewlyClosed []int  `json:"newly_closed"`
}

// ScanJob is the message sent to Redis to trigger a background scan
type ScanJob struct {
	ScanID string `json:"scan_id"`
	Host   string `json:"host"`
}
