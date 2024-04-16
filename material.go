package main

import "image/color"

type Material struct {
	color           color.RGBA
	roughness       float64
	transparency    float64
	refractionIndex float64
}
