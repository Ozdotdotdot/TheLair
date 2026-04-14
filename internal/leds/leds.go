package leds

import (
	"fmt"
	"sync"
	"time"

	evdev "github.com/holoplot/go-evdev"
	"github.com/ozdotdotdot/TheLair/internal/razer"
)

// The 5 indicator LEDs, ordered bottom-to-top on the wall:
//   Caps Lock (C) → Num Lock (1) → Scroll Lock (S) → Macro (M) → Game Mode (G)
// Caps/Num/Scroll are toggled via evdev key injection.
// Macro/Game Mode are toggled via OpenRazer D-Bus.

var (
	mu  sync.Mutex
	dev *evdev.InputDevice

	// Current state of each LED (to avoid redundant toggles).
	capsState   bool
	numState    bool
	scrollState bool
	macroState  bool
	gameState   bool
)

// Init stores the evdev device for key injection.
func Init(d *evdev.InputDevice) {
	mu.Lock()
	defer mu.Unlock()
	dev = d
}

// SetVolumeMeter lights up 0-5 LEDs based on volume (0-100).
func SetVolumeMeter(volume int) {
	var level int
	switch {
	case volume <= 20:
		level = 0
	case volume <= 40:
		level = 1
	case volume <= 60:
		level = 2
	case volume <= 80:
		level = 3
	case volume <= 95:
		level = 4
	default:
		level = 5
	}

	mu.Lock()
	defer mu.Unlock()

	setLockLED(evdev.KEY_CAPSLOCK, &capsState, level >= 1)
	setLockLED(evdev.KEY_NUMLOCK, &numState, level >= 2)
	setLockLED(evdev.KEY_SCROLLLOCK, &scrollState, level >= 3)
	setMacro(level >= 4)
	setGame(level >= 5)
}

// Cleanup turns off all indicator LEDs and restores lock states.
func Cleanup() {
	mu.Lock()
	defer mu.Unlock()

	setLockLED(evdev.KEY_CAPSLOCK, &capsState, false)
	setLockLED(evdev.KEY_NUMLOCK, &numState, false)
	setLockLED(evdev.KEY_SCROLLLOCK, &scrollState, false)
	setMacro(false)
	setGame(false)
}

// setLockLED toggles a lock key LED by injecting a key down+up event.
func setLockLED(code evdev.EvCode, current *bool, want bool) {
	if *current == want || dev == nil {
		return
	}

	down := &evdev.InputEvent{Type: evdev.EV_KEY, Code: code, Value: 1}
	up := &evdev.InputEvent{Type: evdev.EV_KEY, Code: code, Value: 0}
	syn := &evdev.InputEvent{Type: evdev.EV_SYN, Code: 0, Value: 0}

	if err := dev.WriteOne(down); err != nil {
		fmt.Printf("[leds] write key down failed: %v\n", err)
		return
	}
	if err := dev.WriteOne(up); err != nil {
		fmt.Printf("[leds] write key up failed: %v\n", err)
		return
	}
	if err := dev.WriteOne(syn); err != nil {
		fmt.Printf("[leds] write syn failed: %v\n", err)
		return
	}

	// Small delay to let the kernel process the state change.
	time.Sleep(2 * time.Millisecond)
	*current = want
}

func setMacro(want bool) {
	if macroState == want {
		return
	}
	razer.SetMacroLED(want)
	macroState = want
}

func setGame(want bool) {
	if gameState == want {
		return
	}
	razer.SetGameModeLED(want)
	gameState = want
}
