package actions

import "testing"

func TestHealthCheckURLUsesUnauthenticatedRoot(t *testing.T) {
	t.Setenv(
		"HA_LIGHT_TOGGLE_URL",
		"http://homeassistant.local:8123/api/webhook/togglelights?legacy=true#fragment",
	)

	got := HealthCheckURL()
	want := "http://homeassistant.local:8123/"
	if got != want {
		t.Fatalf("HealthCheckURL() = %q, want %q", got, want)
	}
}

func TestHealthCheckURLWithoutConfiguration(t *testing.T) {
	t.Setenv("HA_LIGHT_TOGGLE_URL", "")

	if got := HealthCheckURL(); got != "" {
		t.Fatalf("HealthCheckURL() = %q, want empty string", got)
	}
}
