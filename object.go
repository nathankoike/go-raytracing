package main

import "image/color"

type Object interface {
	// Center() Vec3
	Color() color.RGBA
	Reflectivity() float64
	Roughness() float64
	Hit(Ray) bool
}
