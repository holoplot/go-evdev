package main

import (
	"fmt"
	"os"
	"syscall"
	"time"

	"github.com/holoplot/go-evdev"
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

func cloneDevice(devicePath string) {
	targetDev, err := evdev.Open(devicePath)
	if err != nil {
		fmt.Printf("failed to open target device for cloning: %s", err.Error())
		return

	}
	defer targetDev.Close()

	clonedDev, err := evdev.CloneDevice("my-device-clone", targetDev)
	if err != nil {
		fmt.Printf("failed to clone device: %s", err.Error())
		return
	}
	defer clonedDev.Close()

	moveMouse(clonedDev)
}

func createDevice() {
	dev, err := evdev.CreateDevice(
		"fake-device",
		evdev.InputID{
			BusType: 0x03,
			Vendor:  0x4711,
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
	if err != nil {
		fmt.Printf("failed to create device: %s", err.Error())
		return
	}
	// defer dev.Close()

	moveMouse(dev)
}

func moveMouse(dev *evdev.InputDevice) {
	fmt.Println("Moving the mouse...")
	for i := 0; i < 400; i++ {
		time.Sleep(10 * time.Millisecond)

		evTime := syscall.NsecToTimeval(int64(time.Now().Nanosecond()))

		if i < 100 {
			dev.WriteOne(&evdev.InputEvent{
				Time:  evTime,
				Type:  evdev.EV_REL,
				Code:  evdev.REL_X,
				Value: 2,
			})
		} else if i < 200 {
			dev.WriteOne(&evdev.InputEvent{
				Time:  evTime,
				Type:  evdev.EV_REL,
				Code:  evdev.REL_Y,
				Value: 2,
			})
		} else if i < 300 {
			dev.WriteOne(&evdev.InputEvent{
				Time:  evTime,
				Type:  evdev.EV_REL,
				Code:  evdev.REL_X,
				Value: -2,
			})
		} else {
			dev.WriteOne(&evdev.InputEvent{
				Time:  evTime,
				Type:  evdev.EV_REL,
				Code:  evdev.REL_Y,
				Value: -2,
			})
		}

		dev.WriteOne(&evdev.InputEvent{
			Time:  evTime,
			Type:  evdev.EV_SYN,
			Code:  evdev.SYN_REPORT,
			Value: 0,
		})
	}

	fmt.Println("Done!")
}

func usage() {
	fmt.Print("Create a new input device, or clone capabiliies from an existing one\n\n")
	fmt.Printf("Usage: %s clone [input device]\n", os.Args[0])
	fmt.Printf("       %s create\n\n", os.Args[0])
	fmt.Printf("Available devices:\n")

	listDevices()
}

func main() {
	if len(os.Args) < 2 {
		usage()
		return
	}

	switch os.Args[1] {
	case "clone":
		if len(os.Args) < 3 {
			usage()
			return
		}

		cloneDevice(os.Args[2])

	case "create":
		createDevice()

	default:
		usage()
	}
}
