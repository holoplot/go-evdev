package evdev

var EvCodeNameLookup = map[EvType]map[EvCode]string{
	EV_SYN: SYNNames,
	EV_KEY: KEYNames,
	EV_REL: RELNames,
	EV_ABS: ABSNames,
	EV_MSC: MSCNames,
	EV_SW:  SWNames,
	EV_LED: LEDNames,
	EV_SND: SNDNames,
	EV_REP: REPNames,
	EV_FF:  FFNames,
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
