package v1_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"nmap-rest-api/utils"
	"os"

	"testing"

	v1 "nmap-rest-api/api/v1"
	businessv1 "nmap-rest-api/business/v1"
	modelsv1 "nmap-rest-api/models/v1"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

// mockQueueScan replaces real QueueScan
var mockQueueScanFunc func(context.Context, []string) (string, error)

func init() {
	// Override actual implementation
	businessv1.QueueScan = func(c context.Context, hosts []string) (string, error) {
		return mockQueueScanFunc(c, hosts)
	}
}

// Valid hostname checker override
func TestMain(m *testing.M) {
	utils.IsValidHostname = func(host string) bool {
		return host == "example.com"
	}
	os.Exit(m.Run()) // 🔧 This is essential
}

func setupRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.Default()
	r.POST("/v1/scan", v1.HandleScanRequest)
	return r
}

func TestHandleScanRequest_InvalidJSON(t *testing.T) {
	router := setupRouter()

	req := httptest.NewRequest(http.MethodPost, "/v1/scan", bytes.NewBufferString("{invalid"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandleScanRequest_EmptyHosts(t *testing.T) {
	router := setupRouter()

	payload := `{"hosts":[]}`
	req := httptest.NewRequest(http.MethodPost, "/v1/scan", bytes.NewBufferString(payload))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandleScanRequest_InvalidHost(t *testing.T) {
	router := setupRouter()

	body := modelsv1.ScanRequest{Hosts: []string{"!!badhost"}}
	jsonData, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/v1/scan", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandleScanRequest_BackendError(t *testing.T) {
	router := setupRouter()

	mockQueueScanFunc = func(c context.Context, hosts []string) (string, error) {
		return "", errors.New("backend failed")
	}

	body := modelsv1.ScanRequest{Hosts: []string{"example.com"}}
	jsonData, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/v1/scan", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestHandleScanRequest_Success(t *testing.T) {
	router := setupRouter()

	mockQueueScanFunc = func(c context.Context, hosts []string) (string, error) {
		return "12345", nil
	}

	body := modelsv1.ScanRequest{Hosts: []string{"example.com"}}
	jsonData, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/v1/scan", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusAccepted, w.Code)
}
