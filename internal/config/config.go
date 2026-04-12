package config

import "time"

const (
	KeyboardDevice = "/dev/input/event0"

	IdleTimeout      = 10 * time.Minute
	BrightnessActive = byte(255)
	BrightnessIdle   = byte(38) // ~15%

	ChevronFrameDelay = 30 * time.Millisecond
	ChevronHold       = 80 * time.Millisecond
	ErrorFlashCount   = 3
	ErrorFlashDelay   = 100 * time.Millisecond
)

// Colors as [R, G, B]
var (
	ColorChevron = [3]byte{255, 255, 255}
	ColorError   = [3]byte{255, 0, 0}
	ColorIdle    = [3]byte{20, 20, 60}
	ColorOff     = [3]byte{0, 0, 0}
)
