package testpackage

type lookup struct {
	a, b, c int32
	d       uint32
	e       int64
	g, h    byte
}

type subtable struct {
	version uint16
	x, y    int16
	lookups []lookup
}
