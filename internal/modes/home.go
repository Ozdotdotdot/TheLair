package modes

import (
	evdev "github.com/holoplot/go-evdev"
	"github.com/ozdotdotdot/TheLair/internal/actions"
	"github.com/ozdotdotdot/TheLair/internal/animations"
)

// HomeMode is the default home-automation mode.
type HomeMode struct{}

func (h *HomeMode) Name() string { return "Home" }

func (h *HomeMode) Macros() map[evdev.EvCode]Macro {
	return map[evdev.EvCode]Macro{
		evdev.KEY_F1: {Label: "Toggle Lights", Action: actions.ToggleLights},
		evdev.KEY_1:  {Label: "Toggle Desk Lamp", Action: actions.ToggleDesk},
		evdev.KEY_2:  {Label: "Toggle Hanging Lamp", Action: actions.ToggleHanging},
	}
}

func (h *HomeMode) OnEnter() {
	animations.SetActive()
}

func (h *HomeMode) OnExit() {}

func (h *HomeMode) RestoreState() {
	animations.SetActive()
}
