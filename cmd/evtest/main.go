package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/holoplot/go-evdev"
	"github.com/holoplot/go-evdev/kbext"
)

func listDevices() {
	devicePaths, err := evdev.ListDevicePaths()
	if err != nil {
		fmt.Printf("Cannot list device paths: %s", err)
		return
	}
	for _, d := range devicePaths {
		fmt.Printf("%s:\t%s\n", d.Path, d.Name)
	}
}

func listLayouts() {
	ids := kbext.LayoutKeys()
	for _, d := range ids {
		fmt.Println(d)
	}
}

var (
	kbLayoutStr string
)

func main() {
	flag.StringVar(&kbLayoutStr, "kblayout", string(kbext.LayoutQuertyEnUs), "Keyboard layout")
	flag.Parse()

	args := flag.Args()
	if len(args) < 1 {
		fmt.Printf("Usage: %s <input device>\n\n", os.Args[0])
		fmt.Printf("Available devices:\n")

		listDevices()

		fmt.Printf("\nAvailable keyboard layouts:\n")
		listLayouts()
		return
	}

	d, err := evdev.Open(args[0])
	if err != nil {
		fmt.Printf("Cannot read %s: %v\n", args[0], err)
		return
	}

	vMajor, vMinor, vMicro := d.DriverVersion()
	fmt.Printf("Input driver version is %d.%d.%d\n", vMajor, vMinor, vMicro)

	inputID, err := d.InputID()
	if err == nil {
		fmt.Printf("Input device ID: bus 0x%x vendor 0x%x product 0x%x version 0x%x\n",
			inputID.BusType, inputID.Vendor, inputID.Product, inputID.Version)
	}

	name, err := d.Name()
	if err == nil {
		fmt.Printf("Input device name: \"%s\"\n", name)
	}

	phys, err := d.PhysicalLocation()
	if err == nil {
		fmt.Printf("Input device physical location: %s\n", phys)
	}

	uniq, err := d.UniqueID()
	if err == nil {
		fmt.Printf("Input device unique ID: %s\n", uniq)
	}

	fmt.Printf("Supported events:\n")

	for _, t := range d.CapableTypes() {
		fmt.Printf("  Event type %d (%s)\n", t, evdev.TypeName(t))

		state, err := d.State(t)
		if err == nil {
			for code, value := range state {
				fmt.Printf("    Event code %d (%s) state %v\n", code, evdev.CodeName(t, code), value)
			}
		}

		if t != evdev.EV_ABS {
			continue
		}

		absInfos, err := d.AbsInfos()
		if err != nil {
			continue
		}

		for code, absInfo := range absInfos {
			fmt.Printf("    Event code %d (%s)\n", code, evdev.CodeName(t, code))
			fmt.Printf("      Value: %d\n", absInfo.Value)
			fmt.Printf("      Min: %d\n", absInfo.Minimum)
			fmt.Printf("      Max: %d\n", absInfo.Maximum)

			if absInfo.Fuzz != 0 {
				fmt.Printf("      Fuzz: %d\n", absInfo.Fuzz)
			}
			if absInfo.Flat != 0 {
				fmt.Printf("      Flat: %d\n", absInfo.Flat)
			}
			if absInfo.Resolution != 0 {
				fmt.Printf("      Resolution: %d\n", absInfo.Resolution)
			}
		}
	}

	fmt.Printf("Properties:\n")

	props := d.Properties()

	for _, p := range props {
		fmt.Printf("  Property type %d (%s)\n", p, evdev.PropName(p))
	}

	fmt.Printf("Testing (kb layout=%s)... (interrupt to exit)\n", kbLayoutStr)
	kbState := kbext.NewKbState(kbext.LayoutID(kbLayoutStr))

	for {
		e, err := d.ReadOne()
		if err != nil {
			fmt.Printf("Error reading from device: %v\n", err)
			return
		}

		ts := fmt.Sprintf("Event: time %d.%06d", e.Time.Sec, e.Time.Usec)

		switch e.Type {
		case evdev.EV_SYN:
			switch e.Code {
			case evdev.SYN_MT_REPORT:
				fmt.Printf("%s, ++++++++++++++ %s ++++++++++++\n", ts, e.CodeName())
			case evdev.SYN_DROPPED:
				fmt.Printf("%s, >>>>>>>>>>>>>> %s <<<<<<<<<<<<\n", ts, e.CodeName())
			default:
				fmt.Printf("%s, -------------- %s ------------\n", ts, e.CodeName())
			}
		case evdev.EV_KEY:
			kbevt := evdev.NewKeyEvent(e)
			fmt.Printf("%s, %s\n", ts, kbevt.String())

			pressed, err := kbState.KeyEvent(kbevt)
			if err == nil {
				fmt.Printf("  kb char: %s\n", pressed)
			}
		default:
			fmt.Printf("%s, %s\n", ts, e.String())
		}
	}
}
