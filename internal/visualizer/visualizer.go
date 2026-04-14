package visualizer

import (
	"context"
	"fmt"
	"math"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ozdotdotdot/TheLair/internal/config"
	"github.com/ozdotdotdot/TheLair/internal/razer"
	"github.com/ozdotdotdot/TheLair/internal/sonotui"
)

// Bar layout on wall (keyboard vertical, left side down):
//   Row 5 = leftmost bar (bass)
//   Row 1 = rightmost bar (treble)
//   Row 0 = function row (exempt)
//   Columns run vertically: low cols = bottom of wall, high cols = top.

const numBars = 5 // rows 1-5

// barRows maps bar index (0=bass, 4=treble) to matrix row.
// Bass on the left (row 5), treble on the right (row 1).
var barRows = [numBars]int{5, 4, 3, 2, 1}

var (
	mu              sync.Mutex
	cancel          context.CancelFunc
	paused          atomic.Bool
	running         atomic.Bool
	onTransportFunc func(string)    // called when transport state changes
	onVolumeFunc    func(int)       // called when volume changes
	onPositionFunc  func(int, int)  // called when position changes (elapsed, duration)
)

// OnTransport registers a callback invoked when transport state changes.
func OnTransport(fn func(string)) {
	mu.Lock()
	onTransportFunc = fn
	mu.Unlock()
}

// OnVolume registers a callback invoked when volume changes.
func OnVolume(fn func(int)) {
	mu.Lock()
	onVolumeFunc = fn
	mu.Unlock()
}

// OnPosition registers a callback invoked when playback position changes.
func OnPosition(fn func(elapsed, duration int)) {
	mu.Lock()
	onPositionFunc = fn
	mu.Unlock()
}

// Start begins the visualizer render loop. Call Stop to terminate.
func Start(spectrumCh <-chan sonotui.SpectrumFrame, stateCh <-chan sonotui.PlayerState) {
	mu.Lock()
	if cancel != nil {
		cancel()
	}
	ctx, c := context.WithCancel(context.Background())
	cancel = c
	running.Store(true)
	mu.Unlock()

	go renderLoop(ctx, spectrumCh, stateCh)
}

// Stop terminates the visualizer.
func Stop() {
	mu.Lock()
	if cancel != nil {
		cancel()
		cancel = nil
	}
	running.Store(false)
	mu.Unlock()
}

// Pause temporarily stops rendering (for macro feedback animations).
func Pause()  { paused.Store(true) }
func Resume() { paused.Store(false) }

// Running reports whether the visualizer is active.
func Running() bool { return running.Load() }

func renderLoop(ctx context.Context, spectrumCh <-chan sonotui.SpectrumFrame, stateCh <-chan sonotui.PlayerState) {
	defer running.Store(false)

	var (
		bands      = make([]float64, numBars)
		smoothed   = make([]float64, numBars)
		transport  = "STOPPED"
		lastFrame  sonotui.SpectrumFrame
		lastVolume int
		elapsed    int
		duration   int
		// Frame rate tracking.
		frameCount int
		fpsStart   = time.Now()
	)

	ticker := time.NewTicker(config.VisualizerFrameInterval)
	defer ticker.Stop()

	// Render an initial frame immediately so the mode indicator shows on entry.
	renderFrame(smoothed, transport, elapsed, duration)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}

		if paused.Load() {
			continue
		}

		// Drain latest data from channels non-blockingly.
		for {
			select {
			case frame := <-spectrumCh:
				lastFrame = frame
			case state := <-stateCh:
				if state.Transport != transport {
					transport = state.Transport
					if fn := onTransportFunc; fn != nil {
						fn(transport)
					}
				}
				if state.Volume != lastVolume {
					lastVolume = state.Volume
					if fn := onVolumeFunc; fn != nil {
						fn(lastVolume)
					}
				}
				if state.Elapsed != elapsed || state.Duration != duration {
					elapsed = state.Elapsed
					duration = state.Duration
					if fn := onPositionFunc; fn != nil {
						fn(elapsed, duration)
					}
				}
			default:
				goto render
			}
		}
	render:

		// Update bands from latest spectrum frame.
		if lastFrame.Playing && len(lastFrame.Bands) >= numBars {
			for i := 0; i < numBars; i++ {
				bands[i] = lastFrame.Bands[i]
			}
		} else {
			// Paused, stopped, or no data — decay toward zero.
			for i := range bands {
				bands[i] *= 0.85
				if bands[i] < 0.01 {
					bands[i] = 0
				}
			}
		}

		// Smooth the bars for fluid motion.
		for i := range smoothed {
			smoothed[i] += (bands[i] - smoothed[i]) * config.VisualizerSmoothing
		}

		renderStart := time.Now()
		renderFrame(smoothed, transport, elapsed, duration)
		renderDur := time.Since(renderStart)

		frameCount++
		if elapsed := time.Since(fpsStart); elapsed >= 5*time.Second {
			fmt.Printf("[visualizer] render: %.1f fps (avg flush: %dms)\n",
				float64(frameCount)/elapsed.Seconds(),
				int(renderDur.Milliseconds()))
			frameCount = 0
			fpsStart = time.Now()
		}
	}
}

func renderFrame(bars []float64, transport string, elapsed, duration int) {
	var matrix [razer.MatrixRows][razer.MatrixCols][3]byte

	// Draw visualizer bars on the main section.
	for i := 0; i < numBars; i++ {
		row := barRows[i]
		height := int(math.Round(bars[i] * float64(config.VisualizerMaxHeight)))
		if height > config.VisualizerMaxHeight {
			height = config.VisualizerMaxHeight
		}

		for col := config.VisualizerColStart; col < config.VisualizerColStart+height && col <= config.VisualizerColEnd; col++ {
			// Gradient: integer lerp from base color (bottom) to peak color (top).
			// tFixed: 0-256, avoids float64 conversions.
			tFixed := (col - config.VisualizerColStart) * 256 / config.VisualizerMaxHeight
			matrix[row][col] = lerpColor(config.ColorBarBase, config.ColorBarPeak, tFixed)
		}
	}

	// Draw mode indicator strip (bottom of wall = column 0, left-edge keys).
	for _, pos := range config.MusicModeIndicator {
		matrix[pos[0]][pos[1]] = config.ColorMusicMode
	}

	// Draw action keys in green (PgUp, PgDn, +, -).
	for _, pos := range config.ActionKeyPositions {
		matrix[pos[0]][pos[1]] = config.ColorActionKey
	}

	// Draw numpad icon based on transport state.
	drawNumpadIcon(&matrix, transport)

	// Draw progress bar on function row (F5-F12).
	if duration > 0 {
		progress := float64(elapsed) / float64(duration)
		litCount := int(progress * float64(len(config.ProgressBarPositions)))
		if elapsed > 0 && litCount == 0 {
			litCount = 1
		}
		for i := 0; i < litCount && i < len(config.ProgressBarPositions); i++ {
			pos := config.ProgressBarPositions[i]
			matrix[pos[0]][pos[1]] = config.ColorProgressBar
		}
	}

	razer.FlushMatrix(matrix)
}

// drawNumpadIcon overlays pause/play icon on the numpad area.
func drawNumpadIcon(matrix *[razer.MatrixRows][razer.MatrixCols][3]byte, transport string) {
	color := config.ColorNumpadIcon
	if transport == "PLAYING" {
		for _, pos := range config.NumpadPlay {
			matrix[pos[0]][pos[1]] = color
		}
	} else {
		// Paused, stopped, or initial — show pause icon.
		for _, pos := range config.NumpadPause {
			matrix[pos[0]][pos[1]] = color
		}
	}
}

// lerpColor interpolates between two colors using fixed-point math.
// tFixed: 0 = color a, 256 = color b.
func lerpColor(a, b [3]byte, tFixed int) [3]byte {
	if tFixed < 0 {
		tFixed = 0
	}
	if tFixed > 256 {
		tFixed = 256
	}
	return [3]byte{
		byte(int(a[0]) + (int(b[0])-int(a[0]))*tFixed/256),
		byte(int(a[1]) + (int(b[1])-int(a[1]))*tFixed/256),
		byte(int(a[2]) + (int(b[2])-int(a[2]))*tFixed/256),
	}
}
