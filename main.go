package main

import (
	"fmt"
	"image"
	"image/color"
	"log"
	"math"
	"time"

	"golang.org/x/exp/shiny/driver"
	"golang.org/x/exp/shiny/screen"

	"golang.org/x/mobile/event/lifecycle"
	"golang.org/x/mobile/event/paint"
	"golang.org/x/mobile/event/size"
)

// Some globals to help
var (
	viewportWidth, viewportHeight         = 1280, 720
	rngSeed                       uint16  = 37 // Veritasium: 37
	minBlue                       float64 = 128
	mode                                  = 1 // Which scene do we draw?

	sizeEvent size.Event
)

// A 16-bit LFSR
func lfsr() {
	rngSeed ^= rngSeed >> 7
	rngSeed ^= rngSeed << 9
	rngSeed ^= rngSeed >> 13
}

// Generate a pseudo-random number
func randomUint16() uint16 {
	lfsr()
	return rngSeed
}

// Resize the viewport
func handleResize(s screen.Screen, event size.Event, viewportBuffer *screen.Buffer) {
	// Capture the event
	sizeEvent = event

	// Update the viewport size
	viewportWidth, viewportHeight = event.WidthPx, event.HeightPx

	// Release the old viewport buffer and create a new one of the
	// proper size
	(*viewportBuffer).Release()
	newViewportBuffer, err := s.NewBuffer(image.Point{event.WidthPx, event.HeightPx})

	if err != nil {
		log.Fatalf("couldn't create new buffer at size.Event - %v", err)
	}

	*viewportBuffer = newViewportBuffer
}

// Write a nice gradient to the pixel buffer
func drawRainbowRectangle(pixelBuffer *image.RGBA) {
	// Update the pixel buffer
	for x := 0; x < viewportWidth; x++ {
		for y := 0; y < viewportHeight; y++ {
			pixelBuffer.SetRGBA(
				x,
				y,
				color.RGBA{
					uint8(math.Floor(float64(x) / float64(viewportWidth) * 256)),                                 // R
					uint8(math.Floor(float64(y) / float64(viewportHeight) * 256)),                                // G
					uint8(math.Max(math.Floor(float64(x*y)/float64(viewportWidth*viewportHeight)*256), minBlue)), // B
					0xff}) // A
		}
	}
}

// Write pseudo-random noise to the pixel buffer
func drawNoise(pixelBuffer *image.RGBA) {
	for x := 0; x < viewportWidth; x++ {
		for y := 0; y < viewportHeight; y++ {
			offset := randomUint16() & 7 // Increase randomness, reduce patterns
			pixelBuffer.SetRGBA(
				x,
				y,
				color.RGBA{
					uint8(randomUint16() >> offset), // R
					uint8(randomUint16() >> offset), // G
					uint8(randomUint16() >> offset), // B
					0xff})                           // A
		}
	}
}

// The main render loop of the application
func render(s screen.Screen, window screen.Window, viewportBuffer screen.Buffer) {
	// Clean up when the loop ends
	defer window.Release()
	defer viewportBuffer.Release()
	defer func() { fmt.Println("Cleaning up in render") }()

	// We will write into this buffer to draw to the screen
	pixelBuffer := viewportBuffer.RGBA()

	// Loop over switch statement to listen for window events
	for {
		// Get the type of the next event on the window
		switch event := window.NextEvent().(type) {

		// Check for screen resize
		case size.Event:
			handleResize(s, event, &viewportBuffer)
			pixelBuffer = viewportBuffer.RGBA()

		// If the type of the event is lifecycle.Event
		case lifecycle.Event:
			// Check for window close
			if event.To == lifecycle.StageDead {
				return
			}

		// Check for draw event
		case paint.Event:
			start := time.Now()
			switch mode {
			case 0:
				drawNoise(pixelBuffer)
			case 1:
				drawRainbowRectangle(pixelBuffer)
			}

			// Upload the updated pixel buffer to the viewport
			window.Upload(image.Point{0, 0}, viewportBuffer, sizeEvent.Bounds())
			window.Publish() // Draw the updated buffer to the screen
			fmt.Printf("Render took %dms\n", time.Since(start).Milliseconds())

		}
	}
}

func main() {
	// fmt.Printf("Hello World!" + " Look at me!")
	defer func() { fmt.Println("All Done!") }() // Good cleanup!

	// Run the provided anonymous function on the screen
	driver.Main(func(s screen.Screen) {
		// Create a new window with the screen
		window, err := s.NewWindow(&screen.NewWindowOptions{
			Title:  "Window",
			Width:  viewportWidth,
			Height: viewportHeight,
		})

		// Check for an error creating the window
		if err != nil {
			fmt.Printf("Failed to create Window - %v", err)
		}
		defer window.Release()

		// The size of the viewport can change
		viewportSize := image.Point{viewportWidth, viewportHeight}
		viewportBuffer, err := s.NewBuffer(viewportSize)

		// Check for an error creating the buffer
		if err != nil {
			fmt.Printf("Failed to create pixel buffer - %v", err)
			window.Release() // Clean up the window
		}
		defer viewportBuffer.Release()

		render(s, window, viewportBuffer)
	})
}
