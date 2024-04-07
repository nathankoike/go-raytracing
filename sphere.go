package main

import "image/color"

// Implements Object interface
type Sphere struct {
	position Vec3
	radius   float64
	material Material
}

func (s Sphere) Center() Vec3 {
	return s.position
}

func (s Sphere) Color() color.RGBA {
	return s.material.color
}

func (s Sphere) Reflectivity() float64 {
	return s.material.reflectivity
}

func (s Sphere) Roughness() float64 {
	return s.material.roughness
}

func (s Sphere) Hit(r Ray) bool {
	// Get the distance vector from the origin of the ray to the center of the object
	distance := r.origin.Sub(s.position)

	// Treat the ray and the distance vector as polynomials
	// Calculating the discriminant will give us the number of intersections
	a := r.direction.Dot(r.direction)
	b := distance.Dot(r.direction) * 2
	c := distance.Dot(distance) - s.radius*s.radius

	// If the determinant is ever negative, it meand we have
	return b*b-4*a*c >= 0
}
