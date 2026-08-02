package modes

import (
	"time"

	evdev "github.com/holoplot/go-evdev"
	"github.com/ozdotdotdot/TheLair/internal/actions"
	"github.com/ozdotdotdot/TheLair/internal/animations"
	"github.com/ozdotdotdot/TheLair/internal/scenes"
	"github.com/ozdotdotdot/TheLair/internal/sonotui"
)

// HomeMode is the default home-automation mode.
type HomeMode struct{}

func (h *HomeMode) Name() string { return "Home" }

func (h *HomeMode) Macros() map[evdev.EvCode]Macro {
	return map[evdev.EvCode]Macro{
		evdev.KEY_F1:    {Label: "Toggle Lights", Action: actions.ToggleLights},
		evdev.KEY_SPACE: {Label: "Toggle Lights", Action: actions.ToggleLights},
		evdev.KEY_1:  {Label: "Toggle Desk Lamp", Action: actions.ToggleDesk},
		evdev.KEY_2:  {Label: "Toggle Hanging Lamp", Action: actions.ToggleHanging},
		evdev.KEY_3:  {Label: "Toggle Fan Power", Action: actions.ToggleFanPower},

		evdev.KEY_F2: {Label: "Toggle Fan Light", Action: actions.ToggleFanLight},
		evdev.KEY_F3: {Label: "Toggle Fan Power", Action: actions.ToggleFanPower},

		// Scenes.
		evdev.KEY_F4: {Label: "Leaving", Action: activateLeaving},
		evdev.KEY_F5: {Label: "Focus Mood", Action: activateFocus},
		evdev.KEY_F6: {Label: "Sexy Time", Action: activateSexyTime},
	}
}

func (h *HomeMode) OnEnter() {
	animations.SetActive()
}

func (h *HomeMode) OnExit() {
	scenes.Deactivate()
}

func (h *HomeMode) RestoreState() {
	if scenes.IsActive() {
		scenes.Active().Visual()
		return
	}
	animations.SetActive()
}

func (h *HomeMode) IdleEnabled() bool      { return !scenes.IsActive() }
func (h *HomeMode) SuccessAnimation() bool { return !scenes.IsActive() }

func activateLeaving() bool {
	// Fire-and-forget: lights off + pause music, no persistent scene state.
	actions.LightsOff()
	time.Sleep(300 * time.Millisecond)
	sonotui.Pause()
	return true
}

func activateFocus() bool {
	// Fire-and-forget: don't block the event loop while sonotui clears
	// the queue and adds tracks (can take several seconds).
	go sonotui.PlayMood("focus")
	return true
}

func activateSexyTime() bool {
	return scenes.Activate(scenes.SexyTime)
}
