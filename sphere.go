package main

import (
	"image/color"
	"math"
)

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

func (s Sphere) Roughness() float64 {
	return s.material.roughness
}

func (s Sphere) Transparency() float64 {
	return s.material.transparency
}

// If the ray hits the sphere, return where along the ray it does so
// If the ray does not hit the sphere, return -1
func (s Sphere) Hit(r Ray, itv Interval) float64 {
	// Get the distance vector from the origin of the ray to the center of the object
	distance := r.origin.Sub(s.position)

	// Treat the ray and the distance vector as polynomials
	// Calculating the discriminant will give us the number of intersections
	a := r.direction.LengthSquared()
	halfB := distance.Dot(r.direction)
	c := distance.LengthSquared() - s.radius*s.radius

	discriminant := halfB*halfB - a*c

	// Check for a hit
	if discriminant < 0 {
		return -1
	}

	root := (-halfB - math.Sqrt(discriminant)) / a

	// Find the nearest acceptable root
	if !itv.Contains(root) {
		root = (-halfB + math.Sqrt(discriminant)) / a
		if !itv.Contains(root) {
			return -1
		}
	}

	// Finish the quadratic formula
	return root
}

// The normal vector of the point where the ray hit the sphere
func (s Sphere) Normal(r Ray, t float64) Vec3 {
	return r.At(t).Sub(s.position)
}

// The unit normal vector of the point where the ray hit the sphere
func (s Sphere) UnitNormal(r Ray, t float64) Vec3 {
	return s.Normal(r, t).Unit()
}

// Calculate the refraction of a ray through the sphere
func (s Sphere) Refract(direction Vec3, normal Vec3, hitFront bool) Vec3 {
	// Calcluate the cosine of the angle between the two unit vectors
	cosTheta := math.Min(direction.Unit().Dot(normal.Unit()), 1)

	refractionIndex := s.material.refractionIndex

	// Do we need to flip the refraction index to exit the material?
	if hitFront {
		refractionIndex = 1 / refractionIndex
	}

	// The perpendicular direction of the exit ray
	exitPerpendicular := direction.
		Add(normal.Scale(cosTheta)).
		Scale(refractionIndex)

	// The parallel direction of the exit ray
	exitParallel := normal.Scale(
		-math.Sqrt(math.Abs(1 - exitPerpendicular.LengthSquared())),
	)

	// Add the perpendicular and parallel components of the exit ray
	return exitParallel.Add(exitPerpendicular)
}
