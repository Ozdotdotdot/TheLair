package modes

import (
	"context"
	"fmt"

	evdev "github.com/holoplot/go-evdev"
	"github.com/ozdotdotdot/TheLair/internal/actions"
	"github.com/ozdotdotdot/TheLair/internal/config"
	"github.com/ozdotdotdot/TheLair/internal/leds"
	"github.com/ozdotdotdot/TheLair/internal/razer"
	"github.com/ozdotdotdot/TheLair/internal/sonotui"
	"github.com/ozdotdotdot/TheLair/internal/visualizer"
)

// MusicMode provides media controls and an audio visualizer.
type MusicMode struct {
	cancel context.CancelFunc
}

func (m *MusicMode) Name() string { return "Music" }

func (m *MusicMode) Macros() map[evdev.EvCode]Macro {
	macros := map[evdev.EvCode]Macro{
		evdev.KEY_F1: {Label: "Toggle Lights", Action: actions.ToggleLights},

		// Navigation keys.
		evdev.KEY_PAGEUP:   {Label: "Previous", Action: sonotui.Prev},
		evdev.KEY_PAGEDOWN: {Label: "Next", Action: sonotui.Next},

		// Numpad volume.
		evdev.KEY_KPMINUS: {Label: "Volume Down", Action: sonotui.VolumeDown},
		evdev.KEY_KPPLUS:  {Label: "Volume Up", Action: sonotui.VolumeUp},

		// Numpad numbers — all play/pause.
		evdev.KEY_KP0: {Label: "Play/Pause", Action: playPause},
		evdev.KEY_KP1: {Label: "Play/Pause", Action: playPause},
		evdev.KEY_KP2: {Label: "Play/Pause", Action: playPause},
		evdev.KEY_KP3: {Label: "Play/Pause", Action: playPause},
		evdev.KEY_KP4: {Label: "Play/Pause", Action: playPause},
		evdev.KEY_KP5: {Label: "Play/Pause", Action: playPause},
		evdev.KEY_KP6: {Label: "Play/Pause", Action: playPause},
		evdev.KEY_KP7: {Label: "Play/Pause", Action: playPause},
		evdev.KEY_KP8: {Label: "Play/Pause", Action: playPause},
		evdev.KEY_KP9: {Label: "Play/Pause", Action: playPause},
	}

	// F5-F12: progress bar seek keys.
	seekKeys := []evdev.EvCode{
		evdev.KEY_F5, evdev.KEY_F6, evdev.KEY_F7, evdev.KEY_F8,
		evdev.KEY_F9, evdev.KEY_F10, evdev.KEY_F11, evdev.KEY_F12,
	}
	for i, key := range seekKeys {
		idx := i
		macros[key] = Macro{
			Label:  fmt.Sprintf("Seek %d/8", idx),
			Action: func() bool { return seekTo(idx) },
		}
	}

	return macros
}

func (m *MusicMode) OnEnter() {
	razer.SetBrightness(config.BrightnessActive)

	// Show mode indicator strip on left edge (bottom of wall) with rest dark.
	showModeIndicator()

	if !sonotui.Available() {
		fmt.Println("[music] sonotui not configured")
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	m.cancel = cancel

	spectrumCh := sonotui.PollSpectrum(ctx, config.SpectrumBands, config.VisualizerFrameInterval)
	stateCh := sonotui.Subscribe(ctx)

	visualizer.OnTransport(SetTransport)
	visualizer.OnVolume(leds.SetVolumeMeter)
	visualizer.OnPosition(SetPosition)
	visualizer.Start(spectrumCh, stateCh)

	// Fetch initial state so volume meter, transport, and duration are correct immediately.
	if status, err := sonotui.FetchStatus(); err == nil {
		SetTransport(status.Transport)
		leds.SetVolumeMeter(status.Volume)
		SetPosition(status.Elapsed, status.Duration)
	}
}

func (m *MusicMode) OnExit() {
	visualizer.Stop()
	leds.Cleanup()
	if m.cancel != nil {
		m.cancel()
		m.cancel = nil
	}
}

func (m *MusicMode) RestoreState() {
	if visualizer.Running() {
		visualizer.Resume()
	} else {
		showModeIndicator()
	}
}

func (m *MusicMode) IdleEnabled() bool      { return false }
func (m *MusicMode) SuccessAnimation() bool { return false }

// showModeIndicator draws only the left-edge indicator strip, rest dark.
func showModeIndicator() {
	var matrix [razer.MatrixRows][razer.MatrixCols][3]byte
	for _, pos := range config.MusicModeIndicator {
		matrix[pos[0]][pos[1]] = config.ColorMusicMode
	}
	razer.FlushMatrix(matrix)
}

// playPause toggles based on last known transport state.
// If transport is ambiguous (TRANSITIONING, empty), refresh from the API.
var (
	lastTransport string
	lastDuration  int
)

func playPause() bool {
	t := lastTransport
	if t != "PLAYING" && t != "PAUSED_PLAYBACK" && t != "STOPPED" {
		// Stale or transitioning — fetch fresh state.
		if status, err := sonotui.FetchStatus(); err == nil {
			t = status.Transport
			lastTransport = t
		}
	}
	if t == "PLAYING" {
		return sonotui.Pause()
	}
	return sonotui.Play()
}

// SetTransport is called from the visualizer to keep play/pause toggle in sync.
func SetTransport(t string) {
	lastTransport = t
}

// SetPosition is called from the visualizer when playback position changes.
func SetPosition(elapsed, duration int) {
	lastDuration = duration
}

// seekTo jumps to a proportional position in the current track.
// Seeks to the point where the pressed key's LED lights up:
// F5 → 1/8, F6 → 2/8, ..., F12 → 8/8 (capped at duration-1).
func seekTo(segment int) bool {
	if lastDuration <= 0 {
		return false
	}
	numKeys := len(config.ProgressBarPositions)
	target := ((segment + 1) * lastDuration) / numKeys
	if target >= lastDuration {
		target = lastDuration - 1
	}
	return sonotui.Seek(target)
}

// PauseVisualizer temporarily pauses the visualizer for macro feedback animations.
func PauseVisualizer() {
	visualizer.Pause()
}

// ResumeVisualizer resumes the visualizer after macro feedback.
func ResumeVisualizer() {
	visualizer.Resume()
}

