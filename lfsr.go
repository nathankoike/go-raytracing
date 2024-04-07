package main

type LFSR16 struct {
	seed uint16
}

func (l LFSR16) Shift() LFSR16 {
	var next uint16 = l.seed

	next ^= next >> 7
	next ^= next << 9
	next ^= next >> 13

	return LFSR16{seed: next}
}
