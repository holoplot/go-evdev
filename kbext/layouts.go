package kbext

import evdev "github.com/holoplot/go-evdev"

type LayoutID string

var (
	LayoutQuertyEnUs LayoutID = "querty-en-US"
)

type layout map[evdev.EvCode]keymap

var layouts = map[LayoutID]layout{
	LayoutQuertyEnUs: quertyEnUs,
}

func LayoutKeys() []LayoutID {
	r := make([]LayoutID, 0, len(layouts))
	for k := range layouts {
		r = append(r, k)
	}
	return r
}
