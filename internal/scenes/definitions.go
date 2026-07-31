package scenes

import (
	"fmt"
	"time"

	"github.com/ozdotdotdot/TheLair/internal/actions"
	"github.com/ozdotdotdot/TheLair/internal/config"
	"github.com/ozdotdotdot/TheLair/internal/razer"
	"github.com/ozdotdotdot/TheLair/internal/sonotui"
)

// SexyTime is currently unbound from any key (F4 was reassigned to the
// "Leaving" scene to make room for fan controls) — kept defined in case it
// gets rebound to a free key later.

var SexyTime = &Scene{
	Name:       "Sexy Time",
	AutoReturn: true,
	Setup: func() bool {
		// Best-effort: turn off all lights, then turn on the hanging lamp.
		actions.LightsOff()
		time.Sleep(300 * time.Millisecond)
		actions.ToggleHanging()
		time.Sleep(300 * time.Millisecond)

		// Core: start the mood. Failure here = scene activation fails.
		if !sonotui.PlayMood("sexy-time") {
			fmt.Println("[scenes] sexy-time mood failed to start")
			return false
		}
		return true
	},
	Teardown: func() {
		sonotui.Pause()
	},
	Visual: func() {
		c := config.ColorSceneSexyTime
		razer.SetBrightness(config.BrightnessIdle)
		razer.Breathing(c[0], c[1], c[2])
	},
}
