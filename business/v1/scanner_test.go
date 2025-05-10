package v1_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	business "nmap-rest-api/business/v1"
	database "nmap-rest-api/database"
	"nmap-rest-api/models/v1"
	utils "nmap-rest-api/utils"

	"github.com/go-redis/redismock/v9"
	"github.com/stretchr/testify/assert"
)

func TestQueueScan_Success(t *testing.T) {
	ctx := context.Background()

	// Step 1: Match scan ID exactly
	utils.GenerateScanID = func() string {
		return "mock-scan-id"
	}

	// Step 2: Set up Redis mock
	rdb, mockRedis := redismock.NewClientMock()
	database.RDB = rdb

	// Step 3: Mock DB status setter
	database.SetScanStatus = func(scanID, host, status string) error {
		return nil
	}

	// Step 4: Add expected job payloads
	for _, host := range []string{"host1", "host2"} {
		job := models.ScanJob{ScanID: "mock-scan-id", Host: host}
		jobJSON, _ := json.Marshal(job)
		mockRedis.ExpectRPush("scan_jobs", jobJSON).SetVal(1)
	}

	// Step 5: Call the function
	scanID, err := business.QueueScan(ctx, []string{"host1", "host2"})

	// Step 6: Assert
	assert.NoError(t, err)
	assert.Equal(t, "mock-scan-id", scanID)
	assert.NoError(t, mockRedis.ExpectationsWereMet())
}

func TestQueueScan_DBError(t *testing.T) {
	ctx := context.Background()

	utils.GenerateScanID = func() string {
		return "bad-id"
	}

	database.SetScanStatus = func(scanID, host, status string) error {
		// Fail on purpose
		return errors.New("mock DB error")
	}

	scanID, err := business.QueueScan(ctx, []string{"failhost"})

	assert.Error(t, err)
	assert.Equal(t, "", scanID)
}

func TestQueueScan_RedisError_StillReturnsID(t *testing.T) {
	ctx := context.Background()

	utils.GenerateScanID = func() string {
		return "redis-fail-id"
	}

	database.SetScanStatus = func(scanID, host, status string) error {
		return nil
	}

	// Mock Redis with forced RPush error
	rdb, mockRedis := redismock.NewClientMock()
	database.RDB = rdb

	job := models.ScanJob{ScanID: "redis-fail-id", Host: "hostX"}
	jobJSON, _ := json.Marshal(job)

	// Simulate Redis error but continue anyway
	mockRedis.ExpectRPush("scan_jobs", jobJSON).SetErr(errors.New("redis down"))

	scanID, err := business.QueueScan(ctx, []string{"hostX"})

	assert.NoError(t, err) // still no error returned
	assert.Equal(t, "redis-fail-id", scanID)
	assert.NoError(t, mockRedis.ExpectationsWereMet())
}
