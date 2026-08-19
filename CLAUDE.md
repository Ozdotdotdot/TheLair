# CLAUDE.md

Guidance for working in **TheLair** — the `huntsman-panel` daemon.

## What this is

A Razer Huntsman keyboard mounted on a wall, driven by a **Raspberry Pi 3B+** as a
headless physical macro panel. **No display, no terminal, no login.** The daemon
grabs the keyboard via evdev (keys never reach the OS), maps keys to actions
(Home Assistant webhooks, sonotui media control, shell), and drives the keyboard
RGB via OpenRazer D-Bus as both action feedback and an ambient display.

Single Go binary, cross-compiled for `arm64`, run as a **systemd user service**
with `loginctl enable-linger` so it starts at boot without a login session.

## Layout

| Path | Role |
|------|------|
| `cmd/daemon/main.go` | Entry point: env load, init, the **main event loop**, and the background watcher goroutines (idle, HA health). This is where global state lives. |
| `cmd/discover/main.go` | Standalone tool: lights one matrix position at a time to map `(row,col)` → physical key. Run on the Pi. |
| `internal/razer/` | OpenRazer D-Bus client. `Static`/`Breathing`/`Off`/`SetBrightness` + `FlushMatrix` (diff-based per-row custom lighting). |
| `internal/animations/` | High-level LED states & transient animations: `SetActive`, `SetIdle`, `SetWarning`, `ChevronSuccess`, `ErrorFlash`. One animation at a time, cancellable via `run()`. |
| `internal/leds/` | The 5 indicator LEDs (Caps/Num/Scroll via EV_LED, Macro/Game via D-Bus). |
| `internal/modes/` | Mode system. `HomeMode` (default) and `MusicMode`. First registered = default. Cycle with Pause/Break. |
| `internal/actions/` | HA webhook calls + the HA **health check** URL/client helpers. |
| `internal/scenes/` | Composable automation scenes (setup/teardown/visual + optional SSE auto-return). |
| `internal/sonotui/` | HTTP/SSE client for the external **sonotui** music server (`SONOTUI_URL`). |
| `internal/visualizer/` | FFT spectrum → matrix bars for MusicMode. |
| `internal/config/` | All tunables: timings, colors, matrix positions, keyboard device path. |
| `KEYBOARD_MAP.md` | The Huntsman matrix map (6×22). Partially confirmed; fill in with the discover tool. |

## Build / deploy

Driven by the `Makefile` (targets the Pi at `pi@raspberrypi.local`):

```bash
make build    # GOOS=linux GOARCH=arm64 go build -o bin/huntsman-panel ./cmd/daemon
make deploy    # build, stop service, scp binary, start service
make logs     # journalctl --user -u huntsman-panel -f
make ship     # deploy + logs (the full inner loop)
```

There is **no local run** — the daemon needs the physical keyboard, OpenRazer, and
D-Bus on the Pi. Test by deploying. The committed `huntsman-panel` binary at repo
root is a stale artifact; the real build output is `bin/`.

## Environment (the Pi's `/opt/huntsman-panel/.env`)

Loaded by `loadEnv` at startup (also via the systemd `EnvironmentFile`). Not in the repo.

| Var | Used by |
|-----|---------|
| `HA_LIGHT_TOGGLE_URL` | F1 / Space toggle. **Also the source of the health-check URL** (path swapped to `/api/`). |
| `HA_DESK_TOGGLE_URL` | Key `1` |
| `HA_HANGING_TOGGLE_URL` | Key `2` |
| `HA_FAN_POWER_TOGGLE_URL` | Keys `3` / `F3` — HA webhook that SSHes into musicpi and runs `fan_relay.py power` |
| `HA_FAN_LIGHT_TOGGLE_URL` | F2 — HA webhook that SSHes into musicpi and runs `fan_relay.py light` |
| `HA_AC_TOGGLE_URL` | F4 — HA webhook that toggles the Midea climate entity |
| `HA_AC_OFF_URL` | F5 — HA webhook that idempotently turns off the Midea climate entity |
| `HA_LIGHTS_OFF_URL` | "Leaving" scene (F5), alongside AC off |
| `SONOTUI_URL` | MusicMode + scene music |

## State model — read this before touching the event loop

The keyboard's visual state is the product of **four overlapping, independent
state sources**, tracked as atomics in `cmd/daemon/main.go`:

- `lightsKilled` — ESC kill switch. Highest precedence; when set, all LEDs are off
  and watchers must not draw over it (note the `!lightsKilled.Load()` guards).
- `haUnreachable` — set by the HA health goroutine. Drives **amber**.
- `isIdle` — set by the idle watcher after `IdleTimeout` (1 min) of no keypress.
- `scenes.IsActive()` — an active scene owns the visual and suppresses idle/chevron.

Precedence when restoring after an animation (`restore` closure in the loop):
`lightsKilled` → `haUnreachable` (amber) → `mode.RestoreState()`. Any new watcher
or key handler that repaints the board **must respect these guards** or it will
fight the other sources (this is the cause of "flickering" between states).

## LED states

| State | Visual | Trigger |
|-------|--------|---------|
| Active | static white, full brightness | default resting state |
| Idle | breathing white, ~15% | 1 min no keypress (Home mode only) |
| Macro success | white `>>>` chevrons sweep | action returned true |
| Macro failure | red flash ×3 | action returned false |
| HA unreachable | **static amber** `{255,160,0}` | health check failed (see below) |
| Kill switch | all off | ESC |

## The HA health check (amber) — known source of confusion

Goroutine in `cmd/daemon/main.go`. Mechanics:

- Polls **every 60s** (`HealthCheckInterval`).
- URL = `HealthCheckURL()`: takes `HA_LIGHT_TOGGLE_URL` and replaces the path with
  `/api/`. So health and the F1 toggle hit the **same host**.
- Dedicated client, **3s timeout** (`HealthClient()`).
- `nowUnreachable := err != nil || resp.StatusCode >= 500`.
  → amber on connection error (DNS failure, refused, no route, **3s timeout**) or 5xx.
  → 401/403/404 count as **reachable** (the check only asks "is the box answering").
- On the unreachable→reachable edge it clears `isIdle` and calls `RestoreState()`,
  which is why you see **amber → Active → Idle → amber** cycle as polls flip.

**Gotcha (LAN vs internet):** if `HA_LIGHT_TOGGLE_URL` uses a **hostname** rather
than a raw LAN IP, a downed router/WAN breaks even LAN-local reachability —
`*.local` (mDNS) resolution gets flaky, and DNS names resolve via the router. That
produces intermittent 3s timeouts → flickering amber, even though HA itself is up
on the LAN. A raw LAN IP (or an `/etc/hosts` pin on the Pi) makes it immune. The
same flakiness makes F1 intermittently chevron/red-flash, since it hits the same host.

## Hardware / RGB notes

- OpenRazer D-Bus: service `org.razer`, device path `/org/razer/device/{SERIAL}`
  (serial is **uppercase** and case-sensitive). Lighting via `razer.device.lighting.*`.
- Matrix is **6 rows × 22 cols**. `FlushMatrix` diffs against the previous frame and
  only re-sends dirty rows; it calls `setCustom` once on mode entry (`customMode` flag).
  Any non-custom call (`Static`/`Breathing`/`Off`) clears that flag.
- **Matrix position (for lighting) and evdev key code (for input) are independent.**
  When binding a macro you need both: the evdev code in the mode's `Macros()` map and,
  if you light the key, the `(row,col)` from `KEYBOARD_MAP.md`.
- On the wall the keyboard is mounted vertically (left side down), so matrix rows run
  bottom→top. Color/position constants in `config.go` are written with that in mind.
- OpenRazer may not be ready when the daemon starts at boot → `initRazerWithRetry`
  (10 attempts × 2s).

## Conventions

- Actions return `bool` (success → chevron, failure → red flash). Fire-and-forget
  long work (sonotui mood, scene setup) should `go` so it doesn't block the event loop.
- Colors are `[3]byte{R,G,B}`; brightness is a `byte` 0–255 mapped to 0–100% in `razer`.
- Keep tunables in `config.go`, not inline.
