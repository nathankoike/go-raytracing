package main

type Ray struct {
	origin    Vec3
	direction Vec3
}

func (r Ray) At(t float64) Vec3 {
	return r.origin.Add(r.direction.Scale(t))
}

// Determine if the ray hit the front of an object
func (r Ray) HitFront(normal Vec3) bool {
	// A negative dot product means the normal points against the ray
	return r.direction.Dot(normal.Unit()) < 0
}
