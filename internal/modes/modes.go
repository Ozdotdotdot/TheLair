package modes

import (
	"fmt"
	"sync"

	evdev "github.com/holoplot/go-evdev"
)

// Macro represents a single key binding within a mode.
type Macro struct {
	Label  string
	Action func() bool
}

// Mode defines a keyboard operating mode.
type Mode interface {
	Name() string
	Macros() map[evdev.EvCode]Macro
	OnEnter()      // called when switching TO this mode
	OnExit()       // called when leaving this mode
	RestoreState() // return keyboard to this mode's resting visual state
}

var (
	mu       sync.Mutex
	modes    []Mode
	curIndex int
)

// Register adds a mode. The first registered mode is active by default.
func Register(m Mode) {
	mu.Lock()
	defer mu.Unlock()
	modes = append(modes, m)
}

// Current returns the active mode.
func Current() Mode {
	mu.Lock()
	defer mu.Unlock()
	return modes[curIndex]
}

// Cycle advances to the next mode, calling OnExit/OnEnter.
func Cycle() {
	mu.Lock()
	prev := modes[curIndex]
	curIndex = (curIndex + 1) % len(modes)
	next := modes[curIndex]
	mu.Unlock()

	prev.OnExit()
	fmt.Printf("[modes] %s → %s\n", prev.Name(), next.Name())
	next.OnEnter()
}
