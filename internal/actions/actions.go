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

// toggleWebhook sends a GET to a Home Assistant webhook URL from an env var.
// Returns true on success (2xx response).
func toggleWebhook(envVar string) bool {
	url := os.Getenv(envVar)
	if url == "" {
		fmt.Printf("[actions] %s not set\n", envVar)
		return false
	}
	resp, err := client.Get(url)
	if err != nil {
		fmt.Printf("[actions] %s failed: %v\n", envVar, err)
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode >= 200 && resp.StatusCode < 300
}

func ToggleLights() bool  { return toggleWebhook("HA_LIGHT_TOGGLE_URL") }
func ToggleDesk() bool    { return toggleWebhook("HA_DESK_TOGGLE_URL") }
func ToggleHanging() bool { return toggleWebhook("HA_HANGING_TOGGLE_URL") }
func LightsOff() bool     { return toggleWebhook("HA_LIGHTS_OFF_URL") }
