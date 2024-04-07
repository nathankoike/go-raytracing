package main

type Ray struct {
	origin    Vec3
	direction Vec3
}

func (r Ray) At(t float64) Vec3 {
	return r.origin.Add(r.direction.Scale(t))
}
