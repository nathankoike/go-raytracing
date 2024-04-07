package main

import "image/color"

type Material struct {
	color        color.RGBA
	reflectivity float64
	roughness    float64
}
