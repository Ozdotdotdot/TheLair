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
	"github.com/ozdotdotdot/TheLair/internal/razer"
)

var (
	lightsKilled  atomic.Bool
	isIdle        atomic.Bool
	haUnreachable atomic.Bool
	lastActivity  atomic.Int64 // UnixNano
)

type macro struct {
	label  string
	action func() bool
}

var macros = map[evdev.EvCode]macro{
	evdev.KEY_ESC: {
		label: "Kill Switch",
		action: func() bool {
			if lightsKilled.Load() {
				lightsKilled.Store(false)
				// Restore to the correct state rather than always going active.
				if time.Since(time.Unix(0, lastActivity.Load())) > config.IdleTimeout {
					animations.SetIdle()
				} else {
					animations.SetActive()
				}
			} else {
				lightsKilled.Store(true)
				razer.Off()
			}
			return true
		},
	},
	evdev.KEY_F1: {
		label:  "Toggle Lights",
		action: actions.ToggleLights,
	},
	// evdev.KEY_F2: { label: "Movie Mode", action: actions.MovieMode },
}

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

	// Retry razer init — openrazer-daemon may still be starting up at boot.
	if err := initRazerWithRetry(10, 2*time.Second); err != nil {
		log.Fatalf("[main] razer init: %v", err)
	}
	animations.SetActive()

	dev, err := evdev.Open(config.KeyboardDevice)
	if err != nil {
		log.Fatalf("[main] evdev open: %v", err)
	}
	if err := dev.Grab(); err != nil {
		log.Fatalf("[main] evdev grab: %v", err)
	}

	// Graceful shutdown — restore keyboard state before exit.
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGTERM, syscall.SIGINT)
	go func() {
		<-sigs
		fmt.Println("[main] shutting down")
		dev.Ungrab()
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
			if !isIdle.Load() && elapsed > config.IdleTimeout {
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
		url := os.Getenv("HA_HEALTH_CHECK_URL")
		if url == "" {
			url = os.Getenv("HA_LIGHT_TOGGLE_URL") // fall back to toggle URL as a proxy
		}
		if url == "" {
			log.Println("[health] no URL configured, skipping health check")
			return
		}
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
					if isIdle.Load() {
						animations.SetIdle()
					} else {
						animations.SetActive()
					}
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
				animations.SetActive()
			}
		}

		if modifierCodes[event.Code] {
			continue
		}
		if event.Value != 1 { // key-down only
			continue
		}

		m, ok := macros[event.Code]
		if !ok {
			continue
		}

		fmt.Printf("[main] macro: %s\n", m.label)
		success := m.action()

		// Kill switch manages its own visual state.
		if event.Code == evdev.KEY_ESC {
			continue
		}
		if success {
			animations.ChevronSuccess()
		} else {
			animations.ErrorFlash()
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
