# TheLair
> Making my keyboard the brains to control my room.

![Huntsman Panel](docs/Huntsman%20Panel.jpeg)

A Razer Huntsman keyboard mounted on the wall, connected to a Raspberry Pi 3B+, functioning as a physical macro panel. No display. No terminal. Keys map to actions. RGB provides feedback. The whole thing boots headlessly and runs as a systemd user service.

---

## What it does

- Intercepts raw keyboard input via **evdev** (keys don't reach the OS — the daemon owns them)
- Maps keys to actions (HTTP webhooks, shell commands, whatever you want)
- Drives keyboard RGB via **OpenRazer D-Bus** as a feedback and ambient display
- Runs at boot with no login required via `loginctl enable-linger`

### Key bindings

| Key | Action |
|-----|--------|
| `F1` | Toggle lights via Home Assistant webhook |
| `ESC` | Kill switch — toggle all LEDs off/on |

### LED states

| State | Behavior |
|-------|----------|
| Active | Static white, full brightness |
| Idle (1min no activity) | Breathing white, ~15% brightness |
| Macro success | Three white `>>>` chevrons sweep right |
| Macro failure | Red flash × 3 |
| HA unreachable | Static amber (auto-restores when HA returns) |
| Kill switch ON | All LEDs off |

---

## Hardware

- Raspberry Pi 3B+ (arm64, running Raspberry Pi OS Trixie Lite)
- Razer Huntsman (USB)

---

## Stack

| Layer | Tool |
|-------|------|
| Language | Go — single binary, zero runtime deps on the Pi |
| Key capture | `github.com/holoplot/go-evdev` |
| RGB control | `github.com/godbus/dbus/v5` → OpenRazer daemon |
| Light toggle | `net/http` GET to Home Assistant webhook |
| Service | systemd user unit + `loginctl enable-linger` |
| Cross-compile | `GOOS=linux GOARCH=arm64 go build` |

---

## Pi setup

### 1. Install OpenRazer

```bash
echo 'deb http://download.opensuse.org/repositories/hardware:/razer/Debian_13/ /' \
  | sudo tee /etc/apt/sources.list.d/hardware:razer.list
curl -fsSL https://download.opensuse.org/repositories/hardware:razer/Debian_13/Release.key \
  | gpg --dearmor | sudo tee /etc/apt/trusted.gpg.d/hardware_razer.gpg > /dev/null
sudo apt update
sudo apt install -y openrazer-daemon openrazer-driver-dkms linux-headers-$(uname -r)
```

### 2. Groups, linger, deploy directory

```bash
sudo gpasswd -a pi plugdev
sudo gpasswd -a pi input
loginctl enable-linger pi          # user units start at boot without login
sudo mkdir -p /opt/huntsman-panel
sudo chown pi:pi /opt/huntsman-panel
```

### 3. Load the kernel module at boot

```bash
echo 'razerkbd' | sudo tee /etc/modules-load.d/razerkbd.conf
```

### 4. Reload udev and replug the keyboard

```bash
sudo udevadm control --reload-rules && sudo udevadm trigger
# unplug and replug the Huntsman — udev needs to apply plugdev ownership to sysfs nodes
```

### 5. Install the systemd user unit

```bash
mkdir -p ~/.config/systemd/user
cp huntsman-panel.service ~/.config/systemd/user/
systemctl --user daemon-reload
systemctl --user enable --now huntsman-panel
```

### 6. Create the .env file

```bash
echo 'HA_LIGHT_TOGGLE_URL=http://your-ha-ip:8123/api/webhook/yourwebhook' \
  > /opt/huntsman-panel/.env
```

---

## Development workflow

Code lives on your main machine. The Pi just runs the binary.

```bash
# Build for Pi
make build

# Build + push + restart + tail logs (inner loop)
make ship

# Just tail logs
make logs

# SSH into the Pi
make ssh
```

---

## Adapting for a different Razer keyboard

Three things to change:

**1. Find your event node** — plug in the keyboard, then:
```bash
sudo python3 /tmp/findkeys.py   # see bringup notes below
```
Razer keyboards expose multiple event nodes. The one that produces key events is often `if01`, not the default `event-kbd`. Update `KeyboardDevice` in `internal/config/config.go`.

**2. Get your matrix dimensions** — after openrazer-daemon detects your device:
```bash
python3 -c "
import dbus
bus = dbus.SessionBus()
obj = bus.get_object('org.razer', '/org/razer')
serial = str(obj.getDevices(dbus_interface='razer.devices')[0])
dev = bus.get_object('org.razer', '/org/razer/device/' + serial)
print(dev.getMatrixDimensions(dbus_interface='razer.device.misc'))
"
```
Update `MatrixRows` and `MatrixCols` in `internal/razer/razer.go`.

**3. Add your macros** — edit the `macros` map in `cmd/daemon/main.go`.

---

## Bringup notes (hard-won)

Things that weren't obvious and took debugging to discover:

- **The event node is not what you think.** Razer keyboards expose 3–5 `/dev/input/eventX` nodes. On the Huntsman, key events come through `if01-event-kbd` (event1), not `event-kbd` (event0). Use evtest or the Python script above to find yours.

- **The D-Bus serial is case-sensitive.** `openrazer-daemon` returns the serial in uppercase (e.g. `PM1839F24710643`) and the object path must match exactly — do not lowercase it.

- **openrazer-daemon finds no devices until udev rules apply.** After installing, you must run `udevadm control --reload-rules && udevadm trigger` and then physically replug the keyboard. The `plugdev` group must own the sysfs device nodes.

- **The razerkbd kernel module doesn't load automatically** until you add it to `/etc/modules-load.d/`. Without it, openrazer-daemon starts but detects nothing.

- **User units need `loginctl enable-linger`.** Without it, the systemd user session (and your service) only starts on interactive login, not at boot.

- **Don't use a system unit.** openrazer-daemon runs on the session D-Bus, which a system unit cannot reach. Use a user unit.

---

## Project structure

```
TheLair/
├── cmd/daemon/main.go          # entry point, evdev loop, macro dispatch
├── internal/
│   ├── config/config.go        # constants, colors, timing
│   ├── razer/razer.go          # D-Bus client for OpenRazer
│   ├── animations/animations.go # chevron, error flash, idle
│   └── actions/actions.go      # HTTP webhook calls
├── huntsman-panel.service      # systemd user unit
├── Makefile
└── .env                        # on Pi only, not committed — HA_LIGHT_TOGGLE_URL
```
