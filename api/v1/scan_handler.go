package v1

import (
	"net"
	"net/http"

	businessv1 "nmap-rest-api/business/v1"
	database "nmap-rest-api/database"
	modelsv1 "nmap-rest-api/models/v1"
	"nmap-rest-api/utils"

	"github.com/gin-gonic/gin"
)

// HandleScanRequest godoc
// @Summary     Initiate a scan
// @Description Scans one or more IPs or hostnames in the background and returns a scan ID.
// @Tags        scan
// @Accept      json
// @Produce     json
// @Param       request body modelsv1.ScanRequest true "Scan input"
// @Success     202 {object} map[string]string
// @Failure     400 {object} map[string]interface{}
// @Router      /scan [post]
func HandleScanRequest(c *gin.Context) {
	var req modelsv1.ScanRequest
	if err := c.BindJSON(&req); err != nil || len(req.Hosts) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
		return
	}

	var invalidHosts []string
	for _, h := range req.Hosts {
		if net.ParseIP(h) == nil && !utils.IsValidHostname(h) {
			invalidHosts = append(invalidHosts, h)
		}
	}
	if len(invalidHosts) > 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid hostnames/IPs",
			"invalid": invalidHosts,
		})
		return
	}

	scanID, err := businessv1.QueueScan(c, req.Hosts)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Some Error Occourred At Backed",
			"invalid": err.Error(),
		})
	}

	// for success
	c.JSON(http.StatusAccepted, gin.H{
		"message": "Scan scheduled",
		"scan_id": scanID,
	})
}

// GetScanResults godoc
// @Summary     Get scan results
// @Description Returns up to 10 recent scan results for a host. Optionally filter by scan ID.
// @Tags        scan
// @Produce     json
// @Param       host path string true "Host or IP address"
// @Param       scan_id query string false "Filter by scan ID"
// @Success     200 {array} modelsv1.ScanResult
// @Failure     404 {object} map[string]string
// @Router      /results/{host} [get]
func GetScanResults(c *gin.Context) {
	host := c.Param("host")
	scanID := c.Query("scan_id")

	results := businessv1.FetchScanHistoryFiltered(host, scanID)
	if len(results) == 0 {
		c.JSON(http.StatusNotFound, gin.H{"message": "No results found"})
		return
	}

	c.JSON(http.StatusOK, results)
}

// GetScanDiff godoc
// @Summary     Compare last 2 scans
// @Description Returns ports that were newly opened or closed in the most recent scan for the host.
// @Tags        scan
// @Produce     json
// @Param       host path string true "Host or IP address"
// @Success     200 {object} modelsv1.PortDiff
// @Failure     500 {object} map[string]string
// @Router      /diff/{host} [get]
func GetScanDiff(c *gin.Context) {
	host := c.Param("host")
	diff := businessv1.ComputeDiff(host)
	c.JSON(http.StatusOK, diff)
}

// GetScanStatus godoc
// @Summary     Get scan job status
// @Description Returns progress status for a scan ID, including host-wise scan completion states.
// @Tags        scan
// @Produce     json
// @Param       scan_id path string true "Scan ID"
// @Success     200 {object} map[string]interface{}
// @Failure     404 {object} map[string]string
// @Router      /scan/status/{scan_id} [get]
func GetScanStatus(c *gin.Context) {
	scanID := c.Param("scan_id")
	statuses, err := database.GetScanStatuses(scanID)
	if err != nil || len(statuses) == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Scan ID not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"scan_id":  scanID,
		"statuses": statuses,
	})
}
