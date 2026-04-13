package razer

// OpenRazer D-Bus interface (verified via introspection):
//   Service:   org.razer
//   Manager:   /org/razer
//     razer.devices.getDevices() → []string (device serials)
//   Per-device: /org/razer/device/{SERIAL} — serial is UPPERCASE, as returned by getDevices
//     razer.device.lighting.brightness  setBrightness(d: float64 0-100)
//     razer.device.lighting.chroma      setStatic(y,y,y: r,g,b)
//                                       setBreathSingle(y,y,y: r,g,b)
//                                       setCustom()
//                                       setNone()
//     razer.device.lighting.custom      setKeyRow(ay: payload)
//       payload = [row, startCol, endCol, r0,g0,b0, r1,g1,b1, ...]

import (
	"fmt"
	"sync"

	"github.com/godbus/dbus/v5"
)

// MatrixRows and MatrixCols for the Huntsman.
// Verified via introspection: getMatrixDimensions returns [rows, cols].
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
	conn       *dbus.Conn
	path       dbus.ObjectPath
	mu         sync.Mutex
	customMode bool // true when setCustom has been called and no other mode has replaced it
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

	// Serial must be kept as-is (uppercase) — the D-Bus path is case-sensitive.
	serial := serials[0]
	path := dbus.ObjectPath("/org/razer/device/" + serial)
	instance = &Device{conn: conn, path: path}
	fmt.Printf("[razer] connected: %s\n", serial)
	return nil
}

func (d *Device) obj() dbus.BusObject {
	return d.conn.Object(razerService, d.path)
}

func call(method string, args ...interface{}) {
	if instance == nil {
		return
	}
	instance.mu.Lock()
	defer instance.mu.Unlock()
	if err := instance.obj().Call(method, 0, args...).Err; err != nil {
		fmt.Printf("[razer] %s failed: %v\n", method, err)
	}
}

func SetBrightness(level byte) {
	pct := float64(level) / 255.0 * 100.0
	call(ifaceBrightness+".setBrightness", pct)
}

func Static(r, g, b byte) {
	clearCustomMode()
	call(ifaceChroma+".setStatic", r, g, b)
}

func Breathing(r, g, b byte) {
	clearCustomMode()
	call(ifaceChroma+".setBreathSingle", r, g, b)
}

func Off() {
	clearCustomMode()
	call(ifaceChroma+".setNone")
}

func clearCustomMode() {
	if instance != nil {
		instance.customMode = false
	}
}

var (
	// prevMatrix tracks the last flushed state for diff-based updates.
	prevMatrix [MatrixRows][MatrixCols][3]byte
	// payloadBufs are pre-allocated per-row buffers to avoid heap allocs.
	payloadBufs [MatrixRows][3 + MatrixCols*3]byte
)

func init() {
	// Write static header bytes into each payload buffer once.
	for row := 0; row < MatrixRows; row++ {
		payloadBufs[row][0] = byte(row)
		payloadBufs[row][1] = 0
		payloadBufs[row][2] = byte(MatrixCols - 1)
	}
}

func FlushMatrix(matrix [MatrixRows][MatrixCols][3]byte) {
	if instance == nil {
		return
	}
	instance.mu.Lock()
	defer instance.mu.Unlock()

	// Only call setCustom if we're not already in custom mode.
	if !instance.customMode {
		if err := instance.obj().Call(ifaceChroma+".setCustom", 0).Err; err != nil {
			fmt.Printf("[razer] setCustom failed: %v\n", err)
			return
		}
		instance.customMode = true
		// Force full flush on mode entry by zeroing prevMatrix.
		prevMatrix = [MatrixRows][MatrixCols][3]byte{}
	}

	// Collect which rows need updating.
	var dirty [MatrixRows]bool
	dirtyCount := 0
	for row := 0; row < MatrixRows; row++ {
		if matrix[row] != prevMatrix[row] {
			dirty[row] = true
			dirtyCount++
			// Write RGB data into pre-allocated buffer.
			for col := 0; col < MatrixCols; col++ {
				payloadBufs[row][3+col*3] = matrix[row][col][0]
				payloadBufs[row][3+col*3+1] = matrix[row][col][1]
				payloadBufs[row][3+col*3+2] = matrix[row][col][2]
			}
		}
	}

	if dirtyCount == 0 {
		return
	}

	// Send all dirty rows concurrently. dbus.Conn is safe for concurrent calls.
	var wg sync.WaitGroup
	for row := 0; row < MatrixRows; row++ {
		if !dirty[row] {
			continue
		}
		wg.Add(1)
		go func(r int) {
			defer wg.Done()
			payload := payloadBufs[r][:]
			if err := instance.obj().Call(ifaceChroma+".setKeyRow", 0, payload).Err; err != nil {
				fmt.Printf("[razer] setKeyRow row %d failed: %v\n", r, err)
			}
		}(row)
	}
	wg.Wait()

	prevMatrix = matrix
}
