package main

import (
	"fmt"
	"io/ioutil"
	"os"

	"github.com/holoplot/go-evdev"
)

func listDevices() {
	basePath := "/dev/input"

	files, err := ioutil.ReadDir(basePath)
	if err != nil {
		fmt.Printf("Cannot read /dev/input: %v\n", err)
		return
	}

	for _, fileName := range files {
		if fileName.IsDir() {
			continue
		}

		full := fmt.Sprintf("%s/%s", basePath, fileName.Name())
		d, err := evdev.Open(full)
		if err == nil {
			name, _ := d.Name()

			if err == nil {
				fmt.Printf("%s:\t%s\n", d.Path(), name)
			}
		}
	}
}

func cloneDevice(dev *evdev.InputDevice) (*evdev.InputDevice, error) {
	return evdev.CloneDevice("my-device-clone", dev)
}

func createDevice() (*evdev.InputDevice, error) {
	return evdev.CreateDevice(
		"fake-mouse",
		evdev.InputID{
			BusType: 0x03,
			Vendor:  0x4712,
			Product: 0x0816,
			Version: 1,
		},
		map[evdev.EvType][]evdev.EvCode{
			evdev.EV_KEY: {
				evdev.BTN_LEFT,
				evdev.BTN_RIGHT,
				evdev.BTN_MIDDLE,
			},
			evdev.EV_REL: {
				evdev.REL_X,
				evdev.REL_Y,
				evdev.REL_WHEEL,
				evdev.REL_HWHEEL,
			},
		},
	)
}

func main() {
	if len(os.Args) < 2 {
		fmt.Printf("Usage: %s <input device>\n\n", os.Args[0])
		fmt.Printf("Available devices:\n")

		listDevices()
		return
	}

	d, err := evdev.Open(os.Args[1])
	if err != nil {
		fmt.Printf("Cannot read %s: %v\n", os.Args[1], err)
		return
	}

	if err := d.Grab(); err != nil {
		fmt.Printf("Cannot grab device: %s", err)
		return
	} else if err := d.Ungrab(); err != nil {
		fmt.Printf("Cannot ungrab device: %s", err)
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

	fmt.Printf("Testing ... (interrupt to exit)\n")

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
		default:
			fmt.Printf("%s, %s\n", ts, e.String())
		}
	}
}
