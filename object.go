package main

import "image/color"

type Object interface {
	Center() Vec3
	Color() color.RGBA
	Roughness() float64
	Transparency() float64
	Hit(Ray, Interval) float64
	Normal(r Ray, t float64) Vec3
	UnitNormal(r Ray, t float64) Vec3
	Refract(direction Vec3, normal Vec3, hitFront bool) Vec3
}
