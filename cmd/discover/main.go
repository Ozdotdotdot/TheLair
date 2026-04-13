// discover lights up one matrix position at a time so you can map (row, col) to physical keys.
// Run on the Pi: ./discover
// It walks through every position, lighting it bright green for 1 second each.
// Note down which physical key lights up for each (row, col) printed to the console.
package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/ozdotdotdot/TheLair/internal/razer"
)

func main() {
	if err := razer.Init(); err != nil {
		log.Fatalf("razer init: %v", err)
	}

	green := [3]byte{0, 255, 0}

	reader := bufio.NewReader(os.Stdin)
	fmt.Println("Matrix discovery tool. Press Enter to advance to the next position.")
	fmt.Println("Watch which physical key lights up, and note the (row, col) shown here.")
	fmt.Println()

	for row := 0; row < razer.MatrixRows; row++ {
		for col := 0; col < razer.MatrixCols; col++ {
			var matrix [razer.MatrixRows][razer.MatrixCols][3]byte
			matrix[row][col] = green
			razer.FlushMatrix(matrix)

			fmt.Printf("  (%d, %d)  — press Enter for next", row, col)
			reader.ReadString('\n')
		}
	}

	// Clean up — set all off.
	var blank [razer.MatrixRows][razer.MatrixCols][3]byte
	razer.FlushMatrix(blank)
	fmt.Println("Done!")
	time.Sleep(100 * time.Millisecond)
}
