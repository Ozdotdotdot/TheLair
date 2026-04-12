package actions

import (
	"fmt"
	"net/http"
	"net/url"
	"os"
	"time"
)

var client = &http.Client{Timeout: 5 * time.Second}

// HealthClient returns an HTTP client suitable for the health check goroutine.
// Shorter timeout so a slow HA doesn't block the check for too long.
func HealthClient() *http.Client {
	return &http.Client{Timeout: 3 * time.Second}
}

// HealthCheckURL derives the HA health endpoint from HA_LIGHT_TOGGLE_URL.
// Uses /api/ which returns {"message":"API running."} — a no-op status check.
func HealthCheckURL() string {
	raw := os.Getenv("HA_LIGHT_TOGGLE_URL")
	if raw == "" {
		return ""
	}
	u, err := url.Parse(raw)
	if err != nil {
		return ""
	}
	u.Path = "/api/"
	return u.String()
}

// ToggleLights sends a GET to the Home Assistant webhook.
// Returns true on success (2xx response).
func ToggleLights() bool {
	url := os.Getenv("HA_LIGHT_TOGGLE_URL")
	if url == "" {
		fmt.Println("[actions] HA_LIGHT_TOGGLE_URL not set")
		return false
	}
	resp, err := client.Get(url)
	if err != nil {
		fmt.Printf("[actions] light toggle failed: %v\n", err)
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode >= 200 && resp.StatusCode < 300
}
