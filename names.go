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
	names := map[EvCode]string{}

	switch t {
	case EV_ABS:
		names = ABSName
	case EV_SYN:
		names = SYNName
	case EV_KEY:
		names = KEYName
	case EV_SW:
		names = SWName
	case EV_LED:
		names = LEDName
	case EV_SND:
		names = SNDName
	default:
		return "UNSUPPORTED"
	}

	name, ok := names[c]
	if ok {
		return name
	}

	return "UNKNOWN"
}
