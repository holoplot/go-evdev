package evdev

import (
	"reflect"
	"testing"
)

func Test_bitsToArray(t *testing.T) {
	tests := []struct {
		name string
		bits []byte
		want []int
	}{
		{
			name: "1",
			bits: []byte{0x01, 0xff},
			want: []int{0, 8, 9, 10, 11, 12, 13, 14, 15},
		},
		{
			name: "2",
			bits: []byte{},
			want: []int{},
		},
		{
			name: "3",
			bits: []byte{0x00, 0x00, 0x00, 0x00, 0x01},
			want: []int{32},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bm := newBitmap(tt.bits)
			if got := bm.setBits(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("setBits() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_bitmap_bitIsSet(t *testing.T) {
	tests := []struct {
		name string
		bits []byte
		bit  int
		want bool
	}{
		{
			name: "1",
			bits: []byte{0x00, 0x00, 0x00, 0x00, 0x01},
			bit:  32,
			want: true,
		},
		{
			name: "2",
			bits: []byte{0x00, 0x00, 0x00, 0x00, 0x01},
			bit:  31,
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bm := &bitmap{
				bits: tt.bits,
			}
			if got := bm.bitIsSet(tt.bit); got != tt.want {
				t.Errorf("bitmap.bitIsSet() = %v, want %v", got, tt.want)
			}
		})
	}
}
