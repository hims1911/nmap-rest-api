package utils

import (
	"regexp"
	"net"

	"github.com/google/uuid"
)

// Assign to a function variable for test mocking
var (
	IsValidHostname = isValidHostname
	GenerateScanID  = generateScanID
)

func isValidHostname(host string) bool {
	// First, check if it's a valid IP address
	if ip := net.ParseIP(host); ip != nil {
		return true
	}

	hostnameRegex := `^(([a-zA-Z0-9]|[a-zA-Z0-9][a-zA-Z0-9\-]*[a-zA-Z0-9])\.)*([A-Za-z0-9]|[A-Za-z0-9][A-Za-z0-9\-]*[A-Za-z0-9])$`
	r := regexp.MustCompile(hostnameRegex)
	return r.MatchString(host)
}

func generateScanID() string {
	return uuid.New().String()
}

func Diff(a, b []int) []int {
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
