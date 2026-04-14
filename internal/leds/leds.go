package leds

import (
	"fmt"
	"sync"

	evdev "github.com/holoplot/go-evdev"
	"github.com/ozdotdotdot/TheLair/internal/razer"
)

// The 5 indicator LEDs, ordered bottom-to-top on the wall:
//   Caps Lock (C) → Num Lock (1) → Scroll Lock (S) → Macro (M) → Game Mode (G)
// Caps/Num/Scroll are controlled via EV_LED events on the primary kbd node.
// Macro/Game Mode are controlled via OpenRazer D-Bus.

var (
	mu  sync.Mutex
	dev *evdev.InputDevice

	// Current state of each LED (to avoid redundant writes).
	capsState   bool
	numState    bool
	scrollState bool
	macroState  bool
	gameState   bool
)

// Init opens the primary keyboard event node for LED control.
func Init(d *evdev.InputDevice) {
	mu.Lock()
	defer mu.Unlock()
	// Open event0 (primary kbd) — the lock LEDs live here, not on event1 which we grab.
	primary, err := evdev.Open("/dev/input/by-id/usb-Razer_Razer_Huntsman-event-kbd")
	if err != nil {
		fmt.Printf("[leds] failed to open primary kbd node: %v\n", err)
		return
	}
	dev = primary
	fmt.Println("[leds] ready (using primary kbd node)")
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

	setLockLED(evdev.LED_CAPSL, &capsState, level >= 1)
	setLockLED(evdev.LED_NUML, &numState, level >= 2)
	setLockLED(evdev.LED_SCROLLL, &scrollState, level >= 3)
	setMacro(level >= 4)
	setGame(level >= 5)
}

// Cleanup turns off all indicator LEDs.
func Cleanup() {
	mu.Lock()
	defer mu.Unlock()

	setLockLED(evdev.LED_CAPSL, &capsState, false)
	setLockLED(evdev.LED_NUML, &numState, false)
	setLockLED(evdev.LED_SCROLLL, &scrollState, false)
	setMacro(false)
	setGame(false)
}

// setLockLED writes an EV_LED event to toggle a lock indicator.
func setLockLED(code evdev.EvCode, current *bool, want bool) {
	if *current == want || dev == nil {
		return
	}
	var val int32
	if want {
		val = 1
	}
	ev := &evdev.InputEvent{Type: evdev.EV_LED, Code: code, Value: val}
	if err := dev.WriteOne(ev); err != nil {
		fmt.Printf("[leds] EV_LED write failed (code %d): %v\n", code, err)
		return
	}
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
