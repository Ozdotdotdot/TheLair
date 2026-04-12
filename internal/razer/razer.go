package razer

// OpenRazer D-Bus interface:
//   Service:   org.razer
//   Manager:   /org/razer
//     razer.devices.getDevices() → []string (device serials)
//   Per-device: /org/razer/device/{serial}
//     razer.device.lighting.brightness  setBrightness(float64 0-100)
//     razer.device.lighting.chroma      setStatic(r,g,b byte)
//                                       setBreathing(r,g,b byte)
//                                       setCustom()
//                                       setNone()
//     razer.device.lighting.custom      setKeyRow(row, startCol byte, colors []byte)
//       colors = packed RGB: [r0,g0,b0, r1,g1,b1, ...]
//
// Verify matrix dimensions and exact D-Bus path after install:
//   gdbus introspect --session --dest org.razer --object-path /org/razer/device/<serial>

import (
	"fmt"
	"strings"
	"sync"

	"github.com/godbus/dbus/v5"
)

// MatrixRows and MatrixCols for the Huntsman.
// VERIFY these against your specific model via gdbus introspect before running animations.
const (
	MatrixRows = 6
	MatrixCols = 22
)

const (
	razerService    = "org.razer"
	managerPath     = dbus.ObjectPath("/org/razer")
	ifaceDevices    = "razer.devices"
	ifaceChroma     = "razer.device.lighting.chroma"
	ifaceCustom     = "razer.device.lighting.custom"
	ifaceBrightness = "razer.device.lighting.brightness"
)

type Device struct {
	conn *dbus.Conn
	path dbus.ObjectPath
	mu   sync.Mutex
}

var instance *Device

func Init() error {
	conn, err := dbus.ConnectSessionBus()
	if err != nil {
		return fmt.Errorf("dbus connect: %w", err)
	}

	var serials []string
	obj := conn.Object(razerService, managerPath)
	if err := obj.Call(ifaceDevices+".getDevices", 0).Store(&serials); err != nil {
		return fmt.Errorf("getDevices: %w", err)
	}
	if len(serials) == 0 {
		return fmt.Errorf("no Razer devices found via D-Bus")
	}

	// NOTE: verify the path casing against gdbus introspect output —
	// openrazer may or may not lowercase the serial in the object path.
	serial := serials[0]
	path := dbus.ObjectPath("/org/razer/device/" + strings.ToLower(serial))
	instance = &Device{conn: conn, path: path}
	fmt.Printf("[razer] connected: %s\n", serial)
	return nil
}

func (d *Device) obj() dbus.BusObject {
	return d.conn.Object(razerService, d.path)
}

func SetBrightness(level byte) {
	if instance == nil {
		return
	}
	instance.mu.Lock()
	defer instance.mu.Unlock()
	pct := float64(level) / 255.0 * 100.0
	instance.obj().Call(ifaceBrightness+".setBrightness", 0, pct)
}

func Static(r, g, b byte) {
	if instance == nil {
		return
	}
	instance.mu.Lock()
	defer instance.mu.Unlock()
	instance.obj().Call(ifaceChroma+".setStatic", 0, r, g, b)
}

func Breathing(r, g, b byte) {
	if instance == nil {
		return
	}
	instance.mu.Lock()
	defer instance.mu.Unlock()
	instance.obj().Call(ifaceChroma+".setBreathing", 0, r, g, b)
}

func Off() {
	if instance == nil {
		return
	}
	instance.mu.Lock()
	defer instance.mu.Unlock()
	instance.obj().Call(ifaceChroma+".setNone", 0)
}

func FlushMatrix(matrix [MatrixRows][MatrixCols][3]byte) {
	if instance == nil {
		return
	}
	instance.mu.Lock()
	defer instance.mu.Unlock()
	instance.obj().Call(ifaceChroma+".setCustom", 0)
	for row := 0; row < MatrixRows; row++ {
		rowBytes := make([]byte, MatrixCols*3)
		for col := 0; col < MatrixCols; col++ {
			rowBytes[col*3] = matrix[row][col][0]
			rowBytes[col*3+1] = matrix[row][col][1]
			rowBytes[col*3+2] = matrix[row][col][2]
		}
		instance.obj().Call(ifaceCustom+".setKeyRow", 0, byte(row), byte(0), rowBytes)
	}
}
