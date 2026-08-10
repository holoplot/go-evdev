package evdev

type Mask struct {
	typ   EvType // EV_KEY, ...
	allow bool   // Is an allowlist or a denylist?
	codes []EvCode
}

// DenyMask returns a mask that will deny [codes] when passed to
// [InputDevice.SetMask]. If codes is empty, all events of type typ are
// returned.
func DenyMask(typ EvType, codes []EvCode) Mask {
	return Mask{
		typ:   typ,
		codes: codes,
	}
}

// AllowMask returns a mask that will allow [codes] when passed to
// [InputDevice.SetMask]. If codes is empty, no events of type typ are returned.
func AllowMask(typ EvType, codes []EvCode) Mask {
	m := DenyMask(typ, codes)
	m.allow = true
	return m
}
