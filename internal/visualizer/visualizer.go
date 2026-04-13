package visualizer

import (
	"context"
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
	mu      sync.Mutex
	cancel  context.CancelFunc
	paused  atomic.Bool
	running atomic.Bool
)

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
		bands     = make([]float64, numBars)
		smoothed  = make([]float64, numBars)
		transport = "STOPPED"
		lastFrame sonotui.SpectrumFrame
	)

	ticker := time.NewTicker(config.VisualizerFrameInterval)
	defer ticker.Stop()

	// Render an initial frame immediately so the mode indicator shows on entry.
	renderFrame(smoothed, transport)

	for {
		select {
		case <-ctx.Done():
			return
		case frame, ok := <-spectrumCh:
			if ok {
				lastFrame = frame
			}
		case state, ok := <-stateCh:
			if ok {
				transport = state.Transport
			}
		case <-ticker.C:
			if paused.Load() {
				continue
			}

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

			renderFrame(smoothed, transport)
		}
	}
}

func renderFrame(bars []float64, transport string) {
	var matrix [razer.MatrixRows][razer.MatrixCols][3]byte

	// Draw visualizer bars on the main section.
	for i := 0; i < numBars; i++ {
		row := barRows[i]
		height := int(math.Round(bars[i] * float64(config.VisualizerMaxHeight)))
		if height > config.VisualizerMaxHeight {
			height = config.VisualizerMaxHeight
		}

		for col := config.VisualizerColStart; col < config.VisualizerColStart+height && col <= config.VisualizerColEnd; col++ {
			// Gradient: lerp from base color (bottom) to peak color (top).
			t := float64(col-config.VisualizerColStart) / float64(config.VisualizerMaxHeight)
			matrix[row][col] = lerpColor(config.ColorBarBase, config.ColorBarPeak, t)
		}
	}

	// Draw mode indicator strip (bottom of wall = column 0, left-edge keys).
	for _, pos := range config.MusicModeIndicator {
		matrix[pos[0]][pos[1]] = config.ColorMusicMode
	}

	// Draw numpad icon based on transport state.
	drawNumpadIcon(&matrix, transport)

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

func lerpColor(a, b [3]byte, t float64) [3]byte {
	if t < 0 {
		t = 0
	}
	if t > 1 {
		t = 1
	}
	return [3]byte{
		byte(float64(a[0]) + (float64(b[0])-float64(a[0]))*t),
		byte(float64(a[1]) + (float64(b[1])-float64(a[1]))*t),
		byte(float64(a[2]) + (float64(b[2])-float64(a[2]))*t),
	}
}
