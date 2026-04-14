package modes

import (
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
		evdev.KEY_F1: {Label: "Toggle Lights", Action: actions.ToggleLights},
		evdev.KEY_1:  {Label: "Toggle Desk Lamp", Action: actions.ToggleDesk},
		evdev.KEY_2:  {Label: "Toggle Hanging Lamp", Action: actions.ToggleHanging},

		// Scenes.
		evdev.KEY_F3: {Label: "Focus Mood", Action: activateFocus},
		evdev.KEY_F4: {Label: "Sexy Time", Action: activateSexyTime},
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

func activateFocus() bool {
	return sonotui.PlayMood("focus")
}

func activateSexyTime() bool {
	return scenes.Activate(scenes.SexyTime)
}
