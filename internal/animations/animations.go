package animations

import (
	"context"
	"sync"
	"time"

	"github.com/ozdotdotdot/TheLair/internal/config"
	"github.com/ozdotdotdot/TheLair/internal/razer"
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

// ChevronSuccess sweeps multiple > chevrons across the keyboard left→right.
//
// The chevron shape for 6 rows (tip at center, spreading outward):
//
//	row 0: ░░░░X░░   (2 cols behind tip)
//	row 1: ░░░░░X░   (1 col behind tip)
//	row 2: ░░░░░░X   (tip)
//	row 3: ░░░░░░X   (tip)
//	row 4: ░░░░░X░   (1 col behind tip)
//	row 5: ░░░░X░░   (2 cols behind tip)
func ChevronSuccess() {
	run(func(ctx context.Context) {
		const (
			numChevrons = 3
			spacing     = 6 // cols between chevron tips
		)
		// rowOffset[row] = how many cols behind the tip that row's pixel sits.
		// 6 rows: tip is at rows 2 & 3, spreading to rows 1 & 4, then 0 & 5.
		rowOffset := [razer.MatrixRows]int{2, 1, 0, 0, 1, 2}
		maxOffset := rowOffset[0] // = 2

		// Animation runs until the last chevron's trailing edge exits the right side.
		totalFrames := razer.MatrixCols + (numChevrons-1)*spacing + maxOffset

		for frame := 0; frame < totalFrames; frame++ {
			select {
			case <-ctx.Done():
				return
			default:
			}

			var matrix [razer.MatrixRows][razer.MatrixCols][3]byte

			for ch := 0; ch < numChevrons; ch++ {
				tipCol := frame - ch*spacing
				for row := 0; row < razer.MatrixRows; row++ {
					col := tipCol - rowOffset[row]
					if col >= 0 && col < razer.MatrixCols {
						matrix[row][col] = config.ColorChevron
					}
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
			razer.Static(config.ColorOff[0], config.ColorOff[1], config.ColorOff[2])
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

func SetWarning() {
	razer.SetBrightness(config.BrightnessActive)
	razer.Static(config.ColorWarning[0], config.ColorWarning[1], config.ColorWarning[2])
}
