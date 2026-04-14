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
	return map[evdev.EvCode]Macro{
		evdev.KEY_F1: {Label: "Toggle Lights", Action: actions.ToggleLights},
		evdev.KEY_F5: {Label: "Play/Pause", Action: playPause},
		evdev.KEY_F4: {Label: "Previous", Action: sonotui.Prev},
		evdev.KEY_F6: {Label: "Next", Action: sonotui.Next},
		evdev.KEY_F3: {Label: "Volume Down", Action: sonotui.VolumeDown},
		evdev.KEY_F7: {Label: "Volume Up", Action: sonotui.VolumeUp},

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
	visualizer.Start(spectrumCh, stateCh)
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
// If we can't determine state, try pause first (more likely playing).
var lastTransport string

func playPause() bool {
	if lastTransport == "PLAYING" {
		return sonotui.Pause()
	}
	return sonotui.Play()
}

// SetTransport is called from the main loop to keep play/pause toggle in sync.
func SetTransport(t string) {
	lastTransport = t
}

// PauseVisualizer temporarily pauses the visualizer for macro feedback animations.
func PauseVisualizer() {
	visualizer.Pause()
}

// ResumeVisualizer resumes the visualizer after macro feedback.
func ResumeVisualizer() {
	visualizer.Resume()
}

