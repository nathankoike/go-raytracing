package main

import (
	"fmt"
	"image"
	"image/color"
	"log"
	"math"
	"math/rand/v2"
	"time"

	"golang.org/x/exp/shiny/driver"
	"golang.org/x/exp/shiny/screen"

	"golang.org/x/mobile/event/lifecycle"
	"golang.org/x/mobile/event/paint"
	"golang.org/x/mobile/event/size"
)

// Some globals to help
var (
	screenWidth, screenHeight = 1920, 1080 // Higher res for efficiency testing
	// screenWidth, screenHeight = 640, 360 // Lower res for dev testing
	// aspectRatio                       = screenWidth / screenHeight
	viewportHeight float64 = 1

	// Scene selectors
	drawMode = 2 // [noise, rainbowRectangle, rayTraced]

	// Min blue value for rainbow rectangle
	minBlue float64 = 128

	// RGB values for white and the sky
	white = Vec3{x: 255, y: 255, z: 255}
	sky   = Vec3{x: 127, y: 192, z: 255}

	// How many color samples do we take per pixel on the screen
	samplesPerPixel = 8

	// This slice will store all the obejects in out scene
	objects = make([]Object, 0)

	sizeEvent size.Event
)

// Generate a pseudo-random uint16
func randomUint16() uint16 {
	return uint16(rand.IntN(math.MaxUint16))
}

// Create a camera sized and scaled for the current window size
func createCamera() Camera {
	// Setup the camera viewport
	viewportWidth := (float64(screenWidth) / float64(screenHeight)) * viewportHeight

	// Get vec3s that traverse the viewport plane in the same direction as the
	// screen coordinate system
	viewportX := Vec3{x: viewportWidth, y: 0, z: 0}
	viewportY := Vec3{x: 0, y: -viewportHeight, z: 0}

	// Create a camera for the scene
	camera := Camera{
		position:       Vec3{x: 0, y: 0, z: 0},
		focalLength:    1,
		viewportHeight: viewportHeight,
		viewportWidth:  viewportWidth,
		viewportX:      viewportX,
		viewportY:      viewportY,

		// Get vec3s to represent the ratio difference between the viewport
		// vectors and the actual screen size
		pixelDeltaX: viewportX.Div(float64(screenWidth)),
		pixelDeltaY: viewportY.Div(float64(screenHeight)),

		// The location of the top left corner of the screen, relative to the
		// position of the camera
		pixel00: Vec3{x: 0, y: 0, z: 0}, // Fill this in later
	}

	// Set the proprt location of the top left pixel in the camera
	camera.pixel00 = camera.TopLeft().Add(camera.pixelDeltaX.Add(camera.pixelDeltaY).Div(2))

	return camera
}

// Resize the screen
func handleResize(s screen.Screen, event size.Event, screenBuffer *screen.Buffer) {
	// Capture the event
	sizeEvent = event

	// Update the screen size
	screenWidth, screenHeight = event.WidthPx, event.HeightPx

	// Release the old screen buffer and create a new one of the proper size
	(*screenBuffer).Release()
	newscreenBuffer, err := s.NewBuffer(image.Point{event.WidthPx, event.HeightPx})

	if err != nil {
		log.Fatalf("couldn't create new buffer at size.Event - %v", err)
	}

	*screenBuffer = newscreenBuffer
}

// Write pseudo-random noise to the pixel buffer
func drawNoise(pixelBuffer *image.RGBA) {
	for y := 0; y < screenHeight; y++ {
		for x := 0; x < screenWidth; x++ {
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

// Write a nice gradient to the pixel buffer
func drawRainbowRectangle(pixelBuffer *image.RGBA) {
	// Update the pixel buffer
	for y := 0; y < screenHeight; y++ {
		for x := 0; x < screenWidth; x++ {
			pixelBuffer.SetRGBA(
				x,
				y,
				color.RGBA{
					uint8(math.Floor(float64(x) / float64(screenWidth) * 256)),                               // R
					uint8(math.Floor(float64(y) / float64(screenHeight) * 256)),                              // G
					uint8(math.Max(math.Floor(float64(x*y)/float64(screenWidth*screenHeight)*256), minBlue)), // B
					0xff}) // A
		}
	}
}

// Return the color of the sky if the ray misses all objects
func raySkyColor(ray Ray) color.RGBA {
	// Get the color of the skybox at the given ray
	c := 0.5 * (ray.direction.Unit().y + 1.0)
	rgb := white.Scale(1 - c).Add(sky.Scale(c))

	return color.RGBA{
		R: uint8(rgb.x),
		G: uint8(rgb.y),
		B: uint8(rgb.z),
		A: 0xff,
	}
}

// Determine the color based on the normal vector of the object
func normalColor(normal Vec3) color.RGBA {
	// Make sure we have positive numbers, then scale the normal to get usable color values
	scaledNormal := Vec3{normal.x + 1, normal.y + 1, normal.z + 1}.Scale(255)
	return color.RGBA{
		R: uint8(scaledNormal.x / 2),
		G: uint8(scaledNormal.y / 2),
		B: uint8(scaledNormal.z / 2),
		A: 0xff,
	}
}

// What color should the pixel be at the ray?
func rayColor(ray Ray) color.RGBA {
	// Track the closest object hit
	minT := math.MaxFloat64
	var closestObj Object = nil

	// Check if the ray hits any objects
	for _, o := range objects {
		t := o.Hit(ray)

		// Check if there was a hit
		if t > 0 {
			// Check if this object is closer than another object
			if t < minT {
				minT = t
				closestObj = o
			}
		}
	}

	if closestObj != nil {
		// return o.Color() // Return the color of the object

		// The unit normal vector where the ray hits the object
		return normalColor(closestObj.UnitNormal(ray, minT))
	}

	return raySkyColor(ray)
}

// Write a raytraced frame to the pixel buffer
func raytracedScene(pixelBuffer *image.RGBA, camera Camera) {
	for y := 0; y < screenHeight; y++ {
		for x := 0; x < screenWidth; x++ {
			// Store the cumulative RGB values for the pixel color
			pixelColor := Vec3{0, 0, 0}

			// Take multiple samples for the pixel
			for i := 0; i < samplesPerPixel; i++ {
				// Generate some small random offsets for the pixel
				offsetX := rand.Float64() - 0.5
				offsetY := rand.Float64() - 0.5

				// Scale the ratio differences by the coordinates of the current
				// pixel, adding the offset to vary the direction
				xVec := camera.pixelDeltaX.Scale(float64(x) + offsetX)
				yVec := camera.pixelDeltaY.Scale(float64(y) + offsetY)

				// Calculate the offset from the camera position to the pixel on the screen
				directionToPixel := camera.pixel00.Add(xVec).Add(yVec).Sub(camera.position)

				// Cast a ray from the camera center to the pixel
				r := Ray{origin: camera.position, direction: directionToPixel}
				colorOfRay := rayColor(r)

				pixelColor.x += float64(colorOfRay.R)
				pixelColor.y += float64(colorOfRay.G)
				pixelColor.z += float64(colorOfRay.B)
			}

			// Set the final pixel color
			pixelBuffer.SetRGBA(x, y, color.RGBA{
				uint8(pixelColor.x / float64(samplesPerPixel)),
				uint8(pixelColor.y / float64(samplesPerPixel)),
				uint8(pixelColor.z / float64(samplesPerPixel)),
				0xff})
		}
	}
}

// The main render loop of the application
func render(s screen.Screen, window screen.Window, screenBuffer screen.Buffer) {
	// Clean up when the loop ends
	defer window.Release()
	defer screenBuffer.Release()
	defer func() { fmt.Println("Cleaning up in render") }()

	// We will write into this buffer to draw to the screen
	pixelBuffer := screenBuffer.RGBA()

	// We need a camera for the scene
	camera := createCamera()

	// Loop indefinitely, closing when the window is closed
	for {
		// Get the type of the next event on the window
		switch event := window.NextEvent().(type) {

		// Check for screen resize
		case size.Event:
			handleResize(s, event, &screenBuffer)
			camera = createCamera()
			pixelBuffer = screenBuffer.RGBA()

		// If the type of the event is lifecycle.Event
		case lifecycle.Event:
			// Check for window close
			if event.To == lifecycle.StageDead {
				return
			}

		// Check for draw event
		case paint.Event:
			start := time.Now()
			switch drawMode {
			case 0:
				drawNoise(pixelBuffer)
			case 1:
				drawRainbowRectangle(pixelBuffer)
			case 2:
				raytracedScene(pixelBuffer, camera)
			}

			// Upload the updated pixel buffer to the screen
			window.Upload(image.Point{0, 0}, screenBuffer, sizeEvent.Bounds())
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
			Width:  screenWidth,
			Height: screenHeight,
		})

		// Check for an error creating the window
		if err != nil {
			fmt.Printf("Failed to create Window - %v", err)
		}
		defer window.Release()

		// The size of the screen can change
		screenSize := image.Point{screenWidth, screenHeight}
		screenBuffer, err := s.NewBuffer(screenSize)

		// Check for an error creating the buffer
		if err != nil {
			fmt.Printf("Failed to create pixel buffer - %v", err)
			window.Release() // Clean up the window
		}
		defer screenBuffer.Release()

		// Add a ground sphere
		objects = append(objects, Sphere{
			position: Vec3{0, -100.5, -1},
			radius:   100,
			material: Material{
				color:        color.RGBA{0, 255, 0, 255},
				reflectivity: 0,
				roughness:    0,
			},
		})

		// Fill the scene with objects
		objects = append(objects, Sphere{
			position: Vec3{0, 0, -2},
			radius:   0.5,
			material: Material{
				color:        color.RGBA{255, 0, 0, 255},
				reflectivity: 0,
				roughness:    0,
			},
		})

		render(s, window, screenBuffer)
	})
}
