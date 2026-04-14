package scenes

import (
	"context"
	"fmt"
	"sync"

	"github.com/ozdotdotdot/TheLair/internal/animations"
	"github.com/ozdotdotdot/TheLair/internal/sonotui"
)

// Scene defines a composable automation scene triggered from the keyboard.
type Scene struct {
	Name       string
	Setup      func() bool // run on activation — returns success
	Teardown   func()      // run on deactivation (nil = no-op)
	Visual     func()      // set keyboard visual state (nil = no persistent visual)
	AutoReturn bool        // watch SSE for STOPPED to auto-deactivate
}

var (
	mu     sync.Mutex
	active *Scene
	cancel context.CancelFunc
)

// Activate runs a scene's setup, applies its visual, and starts the SSE
// auto-return watcher if configured. Returns false if setup fails.
func Activate(s *Scene) bool {
	mu.Lock()
	defer mu.Unlock()

	// Deactivate any existing scene first.
	deactivateLocked()

	fmt.Printf("[scenes] activating: %s\n", s.Name)
	if !s.Setup() {
		fmt.Printf("[scenes] %s setup failed\n", s.Name)
		return false
	}

	active = s
	if s.Visual != nil {
		s.Visual()
	}

	if s.AutoReturn && sonotui.Available() {
		ctx, c := context.WithCancel(context.Background())
		cancel = c
		go watchForStop(ctx)
	}

	return true
}

// Deactivate exits the active scene, stopping music and restoring the keyboard.
func Deactivate() {
	mu.Lock()
	defer mu.Unlock()
	deactivateLocked()
}

func deactivateLocked() {
	if active == nil {
		return
	}
	fmt.Printf("[scenes] deactivating: %s\n", active.Name)
	if active.Teardown != nil {
		active.Teardown()
	}
	if cancel != nil {
		cancel()
		cancel = nil
	}
	active = nil
	animations.SetActive()
}

// IsActive reports whether a scene is currently running.
func IsActive() bool {
	mu.Lock()
	defer mu.Unlock()
	return active != nil
}

// Active returns the currently running scene, or nil.
func Active() *Scene {
	mu.Lock()
	defer mu.Unlock()
	return active
}

// watchForStop subscribes to SSE events and deactivates the scene when
// transport goes STOPPED (i.e., the mood queue finished playing).
func watchForStop(ctx context.Context) {
	ch := sonotui.Subscribe(ctx)
	for {
		select {
		case <-ctx.Done():
			return
		case state, ok := <-ch:
			if !ok {
				return
			}
			if state.Transport == "STOPPED" {
				fmt.Println("[scenes] transport STOPPED — auto-deactivating")
				Deactivate()
				return
			}
		}
	}
}
