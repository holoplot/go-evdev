package evdev

var EvCodeNameLookup = map[EvType]map[EvCode]string{
	EV_SYN: SYNToString,
	EV_KEY: KEYToString,
	EV_REL: RELToString,
	EV_ABS: ABSToString,
	EV_MSC: MSCToString,
	EV_SW:  SWToString,
	EV_LED: LEDToString,
	EV_SND: SNDToString,
	EV_REP: REPToString,
	EV_FF:  FFToString,
	// EV_PWR:
	// EV_FF_STATUS:
}

// TypeName returns the name of an EvType as string, or "UNKNOWN" if the type is not valid
func TypeName(t EvType) string {
	name, ok := EVToString[t]
	if ok {
		return name
	}
	return "unknown"
}

// PropName returns the name of the given EvProp, or "UNKNOWN" if the property is not valid
func PropName(p EvProp) string {
	name, ok := INPUTToString[p]
	if ok {
		return name
	}
	return "unknown"
}

// CodeName returns the name of an EvfCode in the given EvType, or "UNKNOWN" of the code is not valid.
func CodeName(t EvType, c EvCode) string {
	name, ok := EvCodeNameLookup[t][c]
	if !ok {
		return "unknown"
	}
	return name
}
