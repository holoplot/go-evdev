package evdev

type bitmap struct {
	bits []byte
}

func (bm *bitmap) bitIsSet(bit int) bool {
	if bit > len(bm.bits)*8 {
		return false
	}

	return bm.bits[bit/8]&(1<<(bit%8)) != 0
}

func (bm *bitmap) setBits() []int {
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

func newBitmap(bits []byte) *bitmap {
	return &bitmap{
		bits: bits,
	}
}
