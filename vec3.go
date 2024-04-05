package main

import (
	"fmt"
	"math"
)

type Vec3 struct {
	x float64
	y float64
	z float64
}

func (v Vec3) String() string {
	return fmt.Sprintf("<%3.2[1]f, %3.2[2]f, %3.2[3]f>", v.x, v.y, v.z)
}

func (v Vec3) Length() float64 {
	return math.Sqrt(v.x*v.x + v.y*v.y + v.z*v.z)
}

func (v Vec3) Scale(sc float64) Vec3 {
	return Vec3{
		x: v.x * sc,
		y: v.y * sc,
		z: v.z * sc,
	}
}

func (v Vec3) Mul(c float64) Vec3 {
	return v.Scale(c)
}

func (v Vec3) Div(denom float64) Vec3 {
	return v.Scale(1 / denom)
}

func (v Vec3) Add(v2 Vec3) Vec3 {
	return Vec3{
		x: v.x + v2.x,
		y: v.y + v2.y,
		z: v.z + v2.z,
	}
}

func (v Vec3) Sub(v2 Vec3) Vec3 {
	return Vec3{
		x: v.x - v2.x,
		y: v.y - v2.y,
		z: v.z - v2.z,
	}
}

func (v Vec3) Dot(v2 Vec3) float64 {
	return v.x*v2.x + v.y*v2.y + v.z*v2.z
}

func (v Vec3) Cross(v2 Vec3) Vec3 {
	return Vec3{
		x: v.y*v2.z - v.z*v2.y,
		y: v.z*v2.x - v.x*v2.z,
		z: v.x*v2.y - v.y*v2.x,
	}
}

func (v Vec3) Unit() Vec3 {
	return v.Div(v.Length())
}
