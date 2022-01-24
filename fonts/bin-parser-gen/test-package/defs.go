package testpackage

import "math"

type lookup struct {
	a, b, c int32
	d       uint32
	e       int64
	g, h    byte

	fl  float214
	fl2 fl32
}

type subtable struct {
	version uint16
	x, y    int16
	lookups []lookup
}

type float214 float32 // representated as 2.14 fixed point

func (f *float214) fromUint(v uint16) {
	// TOOD: implement
	*f = 0
}

func (f float214) toUint() uint16 {
	// TOOD: implement
	return uint16(f)
}

type fl32 float32

func (f *fl32) fromUint(v uint32) {
	*f = fl32(math.Float32frombits(v))
}

func (f fl32) toUint() uint32 {
	return math.Float32bits(float32(f))
}
