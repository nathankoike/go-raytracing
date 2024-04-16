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
	// screenWidth, screenHeight = 1920, 1080 // Higher res for efficiency testing
	// screenWidth, screenHeight = 1280, 720 // Medium-high res
	screenWidth, screenHeight = 640, 360 // Lower res for dev testing
	// aspectRatio                       = screenWidth / screenHeight
	viewportHeight float64 = 1

	// Scene selectors
	drawMode = 2 // [noise, rainbowRectangle, rayTraced]

	maxColorVal uint8 = 255 // The maximum value a single color channel can hold

	// Min blue value for rainbow rectangle
	minBlue float64 = 128

	// RGB values for white and the sky
	white = Vec3{x: float64(maxColorVal), y: float64(maxColorVal), z: float64(maxColorVal)}
	sky   = Vec3{x: 127, y: 192, z: float64(maxColorVal)}

	// The number of color samples taken per pixel
	// samplesPerPixel = 128 // Higher value for quality
	samplesPerPixel = 8 // Lower value for testing

	// The number of times a ray can bounce before returning 0
	maxBounces = 16

	// This slice will store all the obejects in out scene
	objects = make([]Object, 0)

	sizeEvent size.Event
)

// Generate a pseudo-random uint16
func randomUint16() uint16 {
	return uint16(rand.IntN(math.MaxUint16))
}

// Generate a random vec3
func randomVec3() Vec3 {
	return Vec3{rand.Float64() - 0.5, rand.Float64() - 0.5, rand.Float64() - 0.5}
}

func randomRangeVec3(min float64, max float64) Vec3 {
	// Gracefully handle bounds error
	if min >= max {
		return Vec3{}
	}

	// How far apart are the min and max values
	offset := float64(max - min)

	// Get some random values on the interval [0, offset]
	x := rand.Float64() * offset
	y := rand.Float64() * offset
	z := rand.Float64() * offset

	// Increase the floor such that every value is on the interval [min, max]
	return Vec3{x + min, y + min, z + min}
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
					maxColorVal})                    // A
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
					maxColorVal}) // A
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
		A: maxColorVal,
	}
}

// Determine the color based on the normal vector of the object
func normalColor(normal Vec3) color.RGBA {
	// Make sure we have positive numbers, then scale the normal to get usable color values
	scaledNormal := Vec3{normal.x + 1, normal.y + 1, normal.z + 1}.Scale(float64(maxColorVal))
	return color.RGBA{
		R: uint8(scaledNormal.x / 2),
		G: uint8(scaledNormal.y / 2),
		B: uint8(scaledNormal.z / 2),
		A: maxColorVal,
	}
}

// Take a linear component and transform it into a gamma value
func linearToGamma(linear float64) float64 {
	if linear > 0 {
		return math.Sqrt(linear)
	}

	return 0
}

// Determine the color based on the color of the object and its surface rougness
func rayObjectColor(object Object, ray Ray, t float64, maxDepth int) color.RGBA {
	// Find the normal of the hit object
	hitNormal := object.UnitNormal(ray, t)

	// Get the object color as a simple vec3 of RGB
	objColor := object.Color()
	objRGB := Vec3{float64(objColor.R), float64(objColor.G), float64(objColor.B)}

	var castColor color.RGBA // This will store the color of a cast ray

	// Store the RGB values of the ray cast into the scene
	refractedRayCastColor := Vec3{0, 0, 0}

	// Check for transparency
	if object.Transparency() > 0 {
		// Did the ray hit the front of the object?
		hitFront := ray.HitFront(hitNormal)

		newRayDir := object.Refract(ray.direction, hitNormal, hitFront).Unit()

		castColor = rayColor(Ray{ray.At(t), newRayDir}, maxDepth-1)

		refractedRayCastColor = Vec3{
			float64(castColor.R),
			float64(castColor.G),
			float64(castColor.B)}

		// Scale the returned values appropriately for their color channels
		refractedRayCastColor = Vec3{
			refractedRayCastColor.x * float64(objRGB.x) / float64(maxColorVal),
			refractedRayCastColor.y * float64(objRGB.y) / float64(maxColorVal),
			refractedRayCastColor.z * float64(objRGB.z) / float64(maxColorVal),
		}
	}

	// Determine the average reflectivity of the object
	var reflectivity float64 = objRGB.x / float64(maxColorVal)
	reflectivity += objRGB.y / float64(maxColorVal)
	reflectivity += objRGB.z / float64(maxColorVal)

	reflectivity /= 3

	// Check against transparency
	reflectivity *= 1 - object.Transparency()

	// Store the RGB values of the ray cast into the scene
	reflectedRayCastColor := Vec3{0, 0, 0}

	// Don't send off reflected rays unnecessarily
	if reflectivity > 0 {
		newRayDir := ray.direction.Reflect(hitNormal)

		// Don't calculate any random vectors unless there's a need to
		if object.Roughness() > 0 {
			// Generate a random unit vector
			randomUnit := randomVec3().Unit()

			// Make sure the random unit vector is on the same hemisphere as the
			// normal vector at the point where the initial ray hit the object
			if !randomUnit.OnPlane(hitNormal) {
				randomUnit = randomUnit.Scale(-1)
			}

			newRayDir = newRayDir.Add(randomUnit.Scale(object.Roughness()))
		}

		newRayDir = newRayDir.Add(hitNormal.Scale(1 - object.Roughness()))

		// Cast a ray and extract its color values
		castColor = rayColor(Ray{ray.At(t), newRayDir}, maxDepth-1)
		reflectedRayCastColor = Vec3{
			float64(castColor.R),
			float64(castColor.G),
			float64(castColor.B)}

		// Scale the returned values appropriately for their color channels
		reflectedRayCastColor = Vec3{
			reflectedRayCastColor.x * float64(objRGB.x) / float64(maxColorVal),
			reflectedRayCastColor.y * float64(objRGB.y) / float64(maxColorVal),
			reflectedRayCastColor.z * float64(objRGB.z) / float64(maxColorVal),
		}
	}

	// Compose the color of the ray
	scaledObjectRGB := objRGB.Scale(1 - reflectivity - object.Transparency())
	scaledReflectedRayRGB := reflectedRayCastColor.Scale(reflectivity)
	scaledRefractedRayRGB := refractedRayCastColor.Scale(object.Transparency())

	composedRGB := Vec3{
		(scaledObjectRGB.x + scaledReflectedRayRGB.x + scaledRefractedRayRGB.x),
		(scaledObjectRGB.y + scaledReflectedRayRGB.y + scaledRefractedRayRGB.y),
		(scaledObjectRGB.z + scaledReflectedRayRGB.z + scaledRefractedRayRGB.z),
	}

	composedRGB = composedRGB.Scale(reflectivity + object.Transparency())

	return color.RGBA{
		uint8(composedRGB.x),
		uint8(composedRGB.y),
		uint8(composedRGB.z),
		maxColorVal}
}

// What color should the pixel be at the ray?
func rayColor(ray Ray, maxDepth int) color.RGBA {
	// If we have run out of depth, return blackness
	if maxDepth < 1 {
		return color.RGBA{0, 0, 0, 0}
	}

	// Track the closest object hit and create an interval for the hit range
	hitInterval := Interval{0.0001, math.MaxFloat64}
	var closestObj Object = nil

	// Check if the ray hits any objects
	for _, o := range objects {
		// Get the distance to the object
		t := o.Hit(ray, hitInterval)

		// Check if there was a closer hit
		if t > 0 {
			hitInterval.max = t
			closestObj = o
		}
	}

	if closestObj != nil {
		// return closestObj.Color() // Return the color of the object

		// // The unit normal vector where the ray hits the object
		// return normalColor(closestObj.UnitNormal(ray, minT))

		// Return the color of the object, accounting for roughness
		return rayObjectColor(closestObj, ray, hitInterval.max, maxDepth)
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
				// Potentially, generate one for each direction, but this ends
				// up adding significant processing time
				var offset float64 = 0
				if samplesPerPixel > 1 {
					offset = rand.Float64() - 0.5
				}

				// Scale the ratio differences by the coordinates of the current
				// pixel, adding the offset to vary the direction
				xVec := camera.pixelDeltaX.Scale(float64(x) + offset)
				yVec := camera.pixelDeltaY.Scale(float64(y) + offset)

				// Calculate the offset from the camera position to the pixel on the screen
				directionToPixel := camera.pixel00.Add(xVec).Add(yVec).Sub(camera.position)

				// Cast a ray from the camera center to the pixel
				r := Ray{origin: camera.position, direction: directionToPixel}
				colorOfRay := rayColor(r, maxBounces)

				pixelColor.x += float64(colorOfRay.R)
				pixelColor.y += float64(colorOfRay.G)
				pixelColor.z += float64(colorOfRay.B)
			}

			// Average the pixel colors
			pixelColor.x = pixelColor.x / float64(samplesPerPixel)
			pixelColor.y = pixelColor.y / float64(samplesPerPixel)
			pixelColor.z = pixelColor.z / float64(samplesPerPixel)

			// // Gamma correction
			// intensity := Interval{0, 1}
			// pixelColor.x = intensity.Clamp(linearToGamma(pixelColor.x/float64(maxColorVal))) * float64(maxColorVal)
			// pixelColor.y = intensity.Clamp(linearToGamma(pixelColor.y/float64(maxColorVal))) * float64(maxColorVal)
			// pixelColor.z = intensity.Clamp(linearToGamma(pixelColor.z/float64(maxColorVal))) * float64(maxColorVal)

			// Set the final pixel color
			pixelBuffer.SetRGBA(x, y, color.RGBA{
				uint8(pixelColor.x),
				uint8(pixelColor.y),
				uint8(pixelColor.z),
				maxColorVal})
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

		// Create some materials
		groundMaterial := Material{
			color:           color.RGBA{128, 128, 128, maxColorVal},
			roughness:       1,
			transparency:    0,
			refractionIndex: 0,
		}

		defaultSphereMaterial := Material{
			color:           color.RGBA{128, 128, 128, maxColorVal},
			roughness:       1,
			transparency:    0,
			refractionIndex: 0,
		}

		metalMaterial := Material{
			color:           color.RGBA{maxColorVal - 0xf, maxColorVal - 0xf, maxColorVal - 0xf, maxColorVal},
			roughness:       0,
			transparency:    0,
			refractionIndex: 0,
		}

		yellowMetalMaterial := Material{
			color:           color.RGBA{maxColorVal, maxColorVal, 128, maxColorVal},
			roughness:       0.1,
			transparency:    0,
			refractionIndex: 0,
		}

		darkMetalMaterial := Material{
			color:           color.RGBA{96, 96, 128, maxColorVal},
			roughness:       0,
			transparency:    0,
			refractionIndex: 0,
		}

		diffuseWhiteMaterial := Material{
			color:           color.RGBA{maxColorVal, maxColorVal, maxColorVal, maxColorVal},
			roughness:       1,
			transparency:    0,
			refractionIndex: 0,
		}

		glassMaterial := Material{
			color:           color.RGBA{maxColorVal, maxColorVal, maxColorVal, maxColorVal},
			roughness:       0,
			transparency:    1,
			refractionIndex: 1.5,
		}

		// Add a ground sphere
		objects = append(objects, Sphere{
			position: Vec3{0, -100.5, -1},
			radius:   100,
			material: groundMaterial,
		})

		// Fill the scene with objects
		objects = append(objects, Sphere{
			position: Vec3{0, 0, -2},
			radius:   0.5,
			material: glassMaterial,
		})

		objects = append(objects, Sphere{
			position: Vec3{-2, 0.5, -3.5},
			radius:   1,
			material: metalMaterial,
		})

		objects = append(objects, Sphere{
			position: Vec3{1.5, 0, -2.5},
			radius:   0.5,
			material: yellowMetalMaterial,
		})

		objects = append(objects, Sphere{
			position: Vec3{1.5, 3.5, -4},
			radius:   3,
			material: darkMetalMaterial,
		})

		objects = append(objects, Sphere{
			position: Vec3{0, -0.4, -1.45},
			radius:   0.1,
			material: diffuseWhiteMaterial,
		})

		objects = append(objects, Sphere{
			position: Vec3{0, 0.25, -5},
			radius:   0.7,
			material: defaultSphereMaterial,
		})

		render(s, window, screenBuffer)
	})
}
