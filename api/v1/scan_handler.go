package v1

import (
	"net"
	"net/http"
	"regexp"

	businessv1 "nmap-rest-api/business/v1"
	database "nmap-rest-api/database"
	v1 "nmap-rest-api/models/v1"

	"github.com/gin-gonic/gin"
)

// HandleScanRequest godoc
// @Summary     Initiate a scan
// @Description Scans one or more IPs or hostnames in background and returns a scan ID.
// @Tags        scan
// @Accept      json
// @Produce     json
// @Param       request body v1.ScanRequest true "Scan input"
// @Success     202 {object} map[string]string
// @Failure     400 {object} map[string]interface{}
// @Router      /scan [post]
func HandleScanRequest(c *gin.Context) {
	var req v1.ScanRequest
	if err := c.BindJSON(&req); err != nil || len(req.Hosts) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
		return
	}

	var invalidHosts []string
	for _, h := range req.Hosts {
		if net.ParseIP(h) == nil && !isValidHostname(h) {
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

	scanID := businessv1.QueueScan(req.Hosts)
	c.JSON(http.StatusAccepted, gin.H{
		"message": "Scan scheduled",
		"scan_id": scanID,
	})
}

// GetScanResults godoc
// @Summary     Get recent scans for a host
// @Description Returns up to 10 recent scan results for the given host.
// @Tags        scan
// @Produce     json
// @Param       host path string true "Host or IP address"
// @Success     200 {array} v1.ScanResult
// @Router      /results/{host} [get]
// GET /results/:host?scan_id=...
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
// @Summary     Compare last 2 scans for a host
// @Description Returns ports that were newly opened or closed in the latest scan.
// @Tags        scan
// @Produce     json
// @Param       host path string true "Host or IP address"
// @Success     200 {object} v1.PortDiff
// @Router      /diff/{host} [get]
func GetScanDiff(c *gin.Context) {
	host := c.Param("host")
	diff := businessv1.ComputeDiff(host)
	c.JSON(http.StatusOK, diff)
}

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

func isValidHostname(host string) bool {
	hostnameRegex := `^(([a-zA-Z0-9]|[a-zA-Z0-9][a-zA-Z0-9\-]*[a-zA-Z0-9])\.)*([A-Za-z0-9]|[A-Za-z0-9][A-Za-z0-9\-]*[A-Za-z0-9])$`
	r := regexp.MustCompile(hostnameRegex)
	return r.MatchString(host)
}
