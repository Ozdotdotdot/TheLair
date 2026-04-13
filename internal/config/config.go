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
	// On the wall, rows are horizontal strips. Two lit rows with a gap = pause ‖.
	// Pause ‖: NumLock,/,* (row 1 numpad) + 4,5,6 (row 3 numpad)
	NumpadPause = [][2]int{
		{1, 18}, {1, 19}, {1, 20}, // NumLock, /, *
		{3, 18}, {3, 19}, {3, 20}, // 4, 5, 6
	}
	// Play ▶: NumLock,/,* (row 1 numpad) + 8 (row 2, center)
	NumpadPlay = [][2]int{
		{1, 18}, {1, 19}, {1, 20}, // NumLock, /, *
		{2, 19},                    // 8
	}

	// Mode indicator: left-edge keys (bottom of wall when mounted vertically).
	// `, Tab, CapsLock, LShift, LCtrl — column 1, rows 1-5.
	MusicModeIndicator = [][2]int{
		{1, 1}, // `
		{2, 1}, // Tab
		{3, 1}, // CapsLock
		{4, 1}, // LShift
		{5, 1}, // LCtrl
	}
)
