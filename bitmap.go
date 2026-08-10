package evdev

type bitmap struct {
	// TODO: Consider making this a []uintptr given that the kernels'
	// bits_from_user requires bitmasks to be long-sized. That would also avoid
	// alignment problems, if any.
	bits []byte
}

func (bm bitmap) set(bit int) {
	if bit < 0 || bit >= len(bm.bits)*8 {
		return
	}
	bm.bits[bit/8] |= (1 << (bit % 8))
}

func (bm bitmap) clear(bit int) {
	if bit < 0 || bit >= len(bm.bits)*8 {
		return
	}
	bm.bits[bit/8] &^= (1 << (bit % 8))
}

func (bm bitmap) setAll() {
	for i := range bm.bits {
		bm.bits[i] = 0xFF
	}
}

func (bm bitmap) bitIsSet(bit int) bool {
	if bit > len(bm.bits)*8 {
		return false
	}

	return bm.bits[bit/8]&(1<<(bit%8)) != 0
}

func (bm bitmap) setBits() []int {
	var a []int

	for i, by := range bm.bits {
		for bit := 0; bit < 8; bit++ {
			if by&byte(1<<bit) != 0 {
				a = append(a, (i*8)+bit)
			}
		}
	}

	return a
}

func newBitmap(bits []byte) bitmap {
	return bitmap{
		bits: bits,
	}
}
