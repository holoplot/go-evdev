package evdev

// TypeName returns the name of an EvType as string, or "UNKNOWN" if the type is not valid
func TypeName(t EvType) string {
	name, ok := EVToString[t]
	if ok {
		return name
	}

	return "UNKNOWN"
}

// PropName returns the name of the given EvProp, or "UNKNOWN" if the property is not valid
func PropName(p EvProp) string {
	name, ok := INPUTToString[p]
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
	case EV_SYN:
		name, ok = SYNToString[c]
	case EV_KEY:
		name, ok = KEYToString[c]
	case EV_REL:
		name, ok = RELToString[c]
	case EV_ABS:
		name, ok = ABSToString[c]
	case EV_MSC:
		name, ok = MSCToString[c]
	case EV_SW:
		name, ok = SWToString[c]
	case EV_LED:
		name, ok = LEDToString[c]
	case EV_SND:
		name, ok = SNDToString[c]
	case EV_REP:
		name, ok = REPToString[c]
	case EV_FF:
		name, ok = FFToString[c]
	// case EV_PWR:
	// case EV_FF_STATUS:
	default:
		return "UNKNOWN"
	}

	if ok {
		return name
	}

	return "UNKNOWN"
}
