package main

import (
	"fmt"
	"os"

	"github.com/holoplot/go-evdev"
)

type evCodeToString = map[evdev.EvCode]string
type evPropToString = map[evdev.EvProp]string
type evTypeToString = map[evdev.EvType]string

type evCodeFromString = map[string]evdev.EvCode
type evPropFromString = map[string]evdev.EvProp
type evTypeFromString = map[string]evdev.EvType

type testPair struct {
	name       string
	toString   interface{}
	fromString interface{}
}

var testingSet = []testPair{
	{"INPUT", evdev.INPUTToString, evdev.INPUTFromString},
	{"EV", evdev.EVToString, evdev.EVFromString},
	{"SYN", evdev.SYNToString, evdev.SYNFromString},
	{"KEY", evdev.KEYToString, evdev.KEYFromString},
	{"REL", evdev.RELToString, evdev.RELFromString},
	{"ABS", evdev.ABSToString, evdev.ABSFromString},
	{"SW", evdev.SWToString, evdev.SWFromString},
	{"MSC", evdev.MSCToString, evdev.MSCFromString},
	{"LED", evdev.LEDToString, evdev.LEDFromString},
	{"REP", evdev.REPToString, evdev.REPFromString},
	{"SND", evdev.SNDToString, evdev.SNDFromString},
	{"ID", evdev.IDToString, evdev.IDFromString},
	{"BUS", evdev.BUSToString, evdev.BUSFromString},
	{"MT", evdev.MTToString, evdev.MTFromString},
	{"FF", evdev.FFToString, evdev.FFFromString},
}

var validMapping, invalidMapping, duplicationMapping int

func contains[T comparable](s []T, e T) bool {
	for _, v := range s {
		if v == e {
			return true
		}
	}
	return false
}

func compareGroups[V evdev.EvCode | evdev.EvType | evdev.EvProp](from map[string]V, to map[V]string) {
	var duplicatedVal = make(map[V][]string)

	for str1, val1 := range from {
		for val2, str2 := range to {
			if val1 == val2 && str1 != str2 {
				if !contains(duplicatedVal[val1], str1) {
					duplicatedVal[val1] = append(duplicatedVal[val1], str1)
				}
				if !contains(duplicatedVal[val1], str2) {
					duplicatedVal[val1] = append(duplicatedVal[val1], str2)
				}
			}
		}
	}

	// test mapping in to>from direction
	for k1, v := range to {
		k2, ok := from[v]
		if !ok {
			fmt.Printf("- missing fromString item! key: %v (expected value: %4d / 0x%04x)\n", v, k1, k1)
			invalidMapping++
			continue
		}
		if k1 != k2 {
			fmt.Printf("- different fromString value! key: %v, expected: %v, got: %v\n", v, k1, k2)
			invalidMapping++
			continue
		}
		validMapping++
	}

	// test mapping in from>to direction
	for k1, v := range from {
		k2, ok := to[v]
		if !ok {
			fmt.Printf("- missing toString item! key: %4d / 0x%04x (expected value: %v)\n", v, v, k1)
			invalidMapping++
			continue
		}
		if k1 != k2 {
			// may be different due to value duplication
			if contains(duplicatedVal[v], k1) {
				duplicationMapping++
				continue
			}
			fmt.Printf("- different toString value! key: %v, expected: %v, got: %v\n", v, k1, k2)
			invalidMapping++
			continue
		}

		validMapping++
	}
}

func main() {
	for _, p := range testingSet {
		fmt.Printf("Analyzing \"%s\" group...\n", p.name)
		switch to := p.toString.(type) {
		case evCodeToString:
			switch from := p.fromString.(type) {
			case evCodeFromString:
				compareGroups(from, to)
			case evPropFromString, evTypeFromString:
				panic("type mismatch")
			default:
				panic("unexpected type")
			}
		case evPropToString:
			switch from := p.fromString.(type) {
			case evPropFromString:
				compareGroups(from, to)
			case evCodeFromString, evTypeFromString:
				panic("type mismatch")
			default:
				panic("unexpected type")
			}
		case evTypeToString:
			switch from := p.fromString.(type) {
			case evTypeFromString:
				compareGroups(from, to)
			case evCodeFromString, evPropFromString:
				panic("type mismatch")
			default:
				panic("unexpected type")
			}
		default:
			panic("unexpected type")
		}
	}

	fmt.Printf(
		"\nvalid mappings: %d,\nignored due to key value duplication: %d,\ninvalid mappings: %d\n\n",
		validMapping, duplicationMapping, invalidMapping,
	)
	if invalidMapping > 0 {
		fmt.Printf("detected invalid mappings!\n")
		os.Exit(1)
	}

	fmt.Printf("all good\n")
	os.Exit(0)
}
