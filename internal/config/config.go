package config

import "time"

const (
	KeyboardDevice = "/dev/input/by-id/usb-Razer_Razer_Huntsman-if01-event-kbd"

	IdleTimeout       = 10 * time.Minute
	HealthCheckInterval = 60 * time.Second
	BrightnessActive = byte(255)
	BrightnessIdle   = byte(38) // ~15%

	ChevronFrameDelay = 30 * time.Millisecond
	ChevronHold       = 80 * time.Millisecond
	ErrorFlashCount   = 3
	ErrorFlashDelay   = 100 * time.Millisecond
)

// Colors as [R, G, B]
var (
	ColorChevron  = [3]byte{255, 255, 255}
	ColorError    = [3]byte{255, 0, 0}
	ColorIdle     = [3]byte{255, 255, 255}
	ColorWarning  = [3]byte{255, 160, 0} // amber — HA unreachable
	ColorOff      = [3]byte{0, 0, 0}
)
