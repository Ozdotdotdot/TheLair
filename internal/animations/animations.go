package animations

import (
	"context"
	"sync"
	"time"

	"github.com/yourusername/huntsman-panel/internal/config"
	"github.com/yourusername/huntsman-panel/internal/razer"
)

var (
	animMu     sync.Mutex
	animCancel context.CancelFunc
)

// run cancels any in-progress animation and starts fn in a new goroutine.
// Mashing a key 5 times starts 5 cancellations — only the last one runs to completion.
func run(fn func(ctx context.Context)) {
	animMu.Lock()
	if animCancel != nil {
		animCancel()
	}
	ctx, cancel := context.WithCancel(context.Background())
	animCancel = cancel
	animMu.Unlock()

	go fn(ctx)
}

// ChevronSuccess sweeps white columns left→right, trails off behind.
func ChevronSuccess() {
	run(func(ctx context.Context) {
		var matrix [razer.MatrixRows][razer.MatrixCols][3]byte

		for col := 0; col < razer.MatrixCols; col++ {
			select {
			case <-ctx.Done():
				return
			default:
			}
			for row := 0; row < razer.MatrixRows; row++ {
				matrix[row][col] = config.ColorChevron
			}
			if col >= 2 {
				for row := 0; row < razer.MatrixRows; row++ {
					matrix[row][col-2] = config.ColorOff
				}
			}
			razer.FlushMatrix(matrix)
			time.Sleep(config.ChevronFrameDelay)
		}

		select {
		case <-ctx.Done():
			return
		default:
			time.Sleep(config.ChevronHold)
			SetActive()
		}
	})
}

// ErrorFlash strobes the board red N times.
func ErrorFlash() {
	run(func(ctx context.Context) {
		for i := 0; i < config.ErrorFlashCount; i++ {
			select {
			case <-ctx.Done():
				return
			default:
			}
			razer.Static(config.ColorError[0], config.ColorError[1], config.ColorError[2])
			time.Sleep(config.ErrorFlashDelay)
			razer.Off()
			time.Sleep(config.ErrorFlashDelay)
		}
		select {
		case <-ctx.Done():
			return
		default:
			SetActive()
		}
	})
}

func SetActive() {
	razer.SetBrightness(config.BrightnessActive)
	razer.Static(config.ColorIdle[0], config.ColorIdle[1], config.ColorIdle[2])
}

func SetIdle() {
	razer.SetBrightness(config.BrightnessIdle)
	razer.Breathing(config.ColorIdle[0], config.ColorIdle[1], config.ColorIdle[2])
}
