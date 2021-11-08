package evdev

// TypeName returns the name of an EvType as string, or "UNKNOWN" if the type is not valid
func TypeName(t EvType) string {
	name, ok := EVName[t]
	if ok {
		return name
	}

	return "UNKNOWN"
}

// PropName returns the name of the given EvProp, or "UNKNOWN" if the property is not valid
func PropName(p EvProp) string {
	name, ok := PROPName[p]
	if ok {
		return name
	}

	return "UNKNOWN"
}

// CodeName returns the name of an EvfCode in the given EvType, or "UNKNOWN" of the code is not valid.
func CodeName(t EvType, c EvCode) string {
	var name string
	var ok bool

	switch t {
	case EV_ABS:
		name, ok = ABSName[c]
	case EV_SYN:
		name, ok = SYNName[c]
	case EV_KEY:
		name, ok = KEYName[c]
		if !ok {
			name, ok = BTNName[c]
		}
	case EV_SW:
		name, ok = SWName[c]
	case EV_LED:
		name, ok = LEDName[c]
	case EV_SND:
		name, ok = SNDName[c]
	default:
		return "UNSUPPORTED"
	}

	if ok {
		return name
	}

	return "UNKNOWN"
}
