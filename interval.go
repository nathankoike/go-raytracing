package main

type Interval struct {
	min float64
	max float64
}

func (itv Interval) Contains(val float64) bool {
	return itv.min <= val && val <= itv.max
}

func (itv Interval) Surrounds(val float64) bool {
	return itv.min < val && val < itv.max
}

func (itv Interval) Clamp(val float64) float64 {
	if val < itv.min {
		return itv.min
	}
	if val > itv.max {
		return itv.max
	}

	return val
}
