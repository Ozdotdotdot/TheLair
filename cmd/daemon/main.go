package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"sync/atomic"
	"syscall"
	"time"

	evdev "github.com/holoplot/go-evdev"
	"github.com/ozdotdotdot/TheLair/internal/actions"
	"github.com/ozdotdotdot/TheLair/internal/animations"
	"github.com/ozdotdotdot/TheLair/internal/config"
	"github.com/ozdotdotdot/TheLair/internal/leds"
	"github.com/ozdotdotdot/TheLair/internal/modes"
	"github.com/ozdotdotdot/TheLair/internal/razer"
	"github.com/ozdotdotdot/TheLair/internal/sonotui"
)

var (
	lightsKilled  atomic.Bool
	isIdle        atomic.Bool
	haUnreachable atomic.Bool
	lastActivity  atomic.Int64 // UnixNano
)

var modifierCodes = map[evdev.EvCode]bool{
	evdev.KEY_LEFTCTRL:   true,
	evdev.KEY_RIGHTCTRL:  true,
	evdev.KEY_LEFTSHIFT:  true,
	evdev.KEY_RIGHTSHIFT: true,
	evdev.KEY_LEFTALT:    true,
	evdev.KEY_RIGHTALT:   true,
}

func main() {
	loadEnv("/opt/huntsman-panel/.env")

	// Init sonotui client (reads SONOTUI_URL env).
	sonotui.Init()

	// Register modes. First registered = default.
	modes.Register(&modes.HomeMode{})
	modes.Register(&modes.MusicMode{})

	// Retry razer init — openrazer-daemon may still be starting up at boot.
	if err := initRazerWithRetry(10, 2*time.Second); err != nil {
		log.Fatalf("[main] razer init: %v", err)
	}
	modes.Current().OnEnter()

	dev, err := evdev.Open(config.KeyboardDevice)
	if err != nil {
		log.Fatalf("[main] evdev open: %v", err)
	}
	if err := dev.Grab(); err != nil {
		log.Fatalf("[main] evdev grab: %v", err)
	}
	leds.Init(dev)

	// Graceful shutdown — restore keyboard state before exit.
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGTERM, syscall.SIGINT)
	go func() {
		<-sigs
		fmt.Println("[main] shutting down")
		leds.Cleanup()
		dev.Ungrab()
		modes.Current().OnExit()
		animations.SetActive()
		os.Exit(0)
	}()

	fmt.Println("[main] listening for macros...")
	lastActivity.Store(time.Now().UnixNano())

	// Idle watcher — checks every 30s, transitions to rest state after IdleTimeout.
	go func() {
		for {
			time.Sleep(30 * time.Second)
			elapsed := time.Since(time.Unix(0, lastActivity.Load()))
			if !isIdle.Load() && elapsed > config.IdleTimeout && modes.Current().IdleEnabled() {
				isIdle.Store(true)
				if !lightsKilled.Load() && !haUnreachable.Load() {
					animations.SetIdle()
				}
			}
		}
	}()

	// HA health check — polls every HealthCheckInterval.
	// Amber keyboard = HA is down. Restores previous state when it comes back.
	go func() {
		client := actions.HealthClient()
		url := actions.HealthCheckURL()
		if url == "" {
			log.Println("[health] no HA URL configured, skipping health check")
			return
		}
		log.Printf("[health] checking %s every %s", url, config.HealthCheckInterval)
		for {
			time.Sleep(config.HealthCheckInterval)
			resp, err := client.Get(url)
			if err == nil {
				resp.Body.Close()
			}
			wasUnreachable := haUnreachable.Load()
			nowUnreachable := err != nil || resp.StatusCode >= 500

			if nowUnreachable && !wasUnreachable {
				haUnreachable.Store(true)
				log.Println("[health] HA unreachable — switching to warning color")
				if !lightsKilled.Load() {
					animations.SetWarning()
				}
			} else if !nowUnreachable && wasUnreachable {
				haUnreachable.Store(false)
				log.Println("[health] HA reachable again — restoring state")
				if !lightsKilled.Load() {
					modes.Current().RestoreState()
				}
			}
		}
	}()

	for {
		event, err := dev.ReadOne()
		if err != nil {
			log.Printf("[main] read error: %v", err)
			continue
		}

		if event.Type != evdev.EV_KEY {
			continue
		}

		// Wake from idle on any keypress.
		lastActivity.Store(time.Now().UnixNano())
		if isIdle.Swap(false) {
			if !lightsKilled.Load() {
				modes.Current().RestoreState()
			}
		}

		if modifierCodes[event.Code] {
			continue
		}
		if event.Value != 1 { // key-down only
			continue
		}

		// --- Global keys (mode-independent) ---

		// ESC: kill switch
		if event.Code == evdev.KEY_ESC {
			fmt.Println("[main] macro: Kill Switch")
			if lightsKilled.Load() {
				lightsKilled.Store(false)
				if time.Since(time.Unix(0, lastActivity.Load())) > config.IdleTimeout {
					animations.SetIdle()
				} else {
					modes.Current().RestoreState()
				}
			} else {
				lightsKilled.Store(true)
				razer.Off()
			}
			continue
		}

		// Pause/Break: cycle mode
		if event.Code == evdev.KEY_PAUSE {
			modes.Cycle()
			continue
		}

		// --- Mode-specific dispatch ---
		m, ok := modes.Current().Macros()[event.Code]
		if !ok {
			continue
		}

		fmt.Printf("[main] macro: %s\n", m.Label)
		success := m.Action()

		// restore returns the keyboard to the correct post-animation state.
		restore := func() {
			switch {
			case lightsKilled.Load():
				razer.Off()
			case haUnreachable.Load():
				animations.SetWarning()
			default:
				modes.Current().RestoreState()
			}
		}

		if success {
			if modes.Current().SuccessAnimation() {
				animations.ChevronSuccess(restore)
			}
		} else {
			animations.ErrorFlash(restore)
		}
	}
}

// initRazerWithRetry attempts razer.Init up to maxAttempts times, waiting
// wait between each. Handles the race between our service and openrazer-daemon at boot.
func initRazerWithRetry(maxAttempts int, wait time.Duration) error {
	var err error
	for i := 0; i < maxAttempts; i++ {
		err = razer.Init()
		if err == nil {
			return nil
		}
		log.Printf("[main] razer init attempt %d/%d failed: %v", i+1, maxAttempts, err)
		time.Sleep(wait)
	}
	return err
}

func loadEnv(path string) {
	f, err := os.Open(path)
	if err != nil {
		log.Printf("[main] no .env at %s", path)
		return
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			os.Setenv(parts[0], parts[1])
		}
	}
}
