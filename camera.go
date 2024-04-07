package main

type Camera struct {
	position       Vec3
	focalLength    float64
	viewportHeight float64
	viewportWidth  float64
	viewportX      Vec3
	viewportY      Vec3
	pixelDeltaX    Vec3
	pixelDeltaY    Vec3
	pixel00        Vec3
}

func (c Camera) TopLeft() Vec3 {
	return c.position.Sub(Vec3{0, 0, c.focalLength}).Sub(c.viewportX.Div(2)).Sub(c.viewportY.Div(2))
}
