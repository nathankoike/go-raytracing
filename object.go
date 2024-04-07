package main

import "image/color"

type Object interface {
	Center() Vec3
	Color() color.RGBA
	Reflectivity() float64
	Roughness() float64
	Hit(Ray) float64
	Normal(r Ray, t float64) Vec3
	UnitNormal(r Ray, t float64) Vec3
}
