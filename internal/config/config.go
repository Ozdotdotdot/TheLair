package config

import "time"

const (
	KeyboardDevice = "/dev/input/by-id/usb-Razer_Razer_Huntsman-if01-event-kbd"

	IdleTimeout         = 1 * time.Minute
	HealthCheckInterval = 60 * time.Second
	BrightnessActive    = byte(255)
	BrightnessIdle      = byte(38) // ~15%

	ChevronFrameDelay = 30 * time.Millisecond
	ChevronHold       = 80 * time.Millisecond
	ErrorFlashCount   = 3
	ErrorFlashDelay   = 100 * time.Millisecond

	// Visualizer settings.
	// On the wall (keyboard vertical, left side down), columns run bottom→top.
	VisualizerFrameInterval = 50 * time.Millisecond // 20 Hz, matches spectrum endpoint
	VisualizerSmoothing     = 0.35                  // lerp factor per frame (0=frozen, 1=instant)
	VisualizerColStart      = 1                     // first column for bars (skip col 0 / edge keys)
	VisualizerColEnd        = 13                    // last column for bars (main alpha section)
	VisualizerMaxHeight     = VisualizerColEnd - VisualizerColStart + 1 // 13 levels per bar
	SpectrumBands           = 5                     // one FFT band per bar (rows 1-5)
)

// Colors as [R, G, B]
var (
	ColorChevron  = [3]byte{255, 255, 255}
	ColorError    = [3]byte{255, 0, 0}
	ColorIdle     = [3]byte{255, 255, 255}
	ColorWarning  = [3]byte{255, 160, 0} // amber — HA unreachable
	ColorOff      = [3]byte{0, 0, 0}

	// Visualizer gradient: base (bottom of bar) → peak (top of bar).
	ColorBarBase    = [3]byte{0, 40, 120}   // deep blue
	ColorBarPeak    = [3]byte{0, 255, 255}  // cyan
	ColorNumpadIcon = [3]byte{255, 255, 255} // white
	ColorMusicMode  = [3]byte{0, 180, 255}  // accent blue for mode indicator

	// Numpad icon positions as [row, col] pairs.
	// These are placeholders — run `cmd/discover` on the Pi to find real positions.
	// Pause ‖: numpad keys 9,8,7 + 3,2,1 (two vertical bars on wall)
	NumpadPause = [][2]int{
		{1, 20}, {2, 20}, {3, 20}, // 9,8,7 column
		{1, 18}, {2, 18}, {3, 18}, // 3,2,1 column
	}
	// Play ▶: numpad keys 9,8,7 + 5 (bar + offset point on wall)
	NumpadPlay = [][2]int{
		{1, 20}, {2, 20}, {3, 20}, // 9,8,7 column
		{2, 19}, // 5 (offset point)
	}
)
