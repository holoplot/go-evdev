package evdev

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"os"
	"syscall"
)

const (
	uinputMaxNameSize = 80
	absSize           = 64
)

// CreateDevice creates a device from scratch with the provided capabilities and name
// If set up fails the device will be removed from the system,
// once set up it can be removed by calling dev.Close
func CreateDevice(name string, id InputID, capabilities map[EvType][]EvCode) (*InputDevice, error) {
	deviceFile, err := os.OpenFile("/dev/uinput", syscall.O_WRONLY|syscall.O_NONBLOCK, 0660)
	if err != nil {
		return nil, err
	}

	newDev := &InputDevice{
		file: deviceFile,
	}

	for ev, codes := range capabilities {
		if err := ioctlUISETEVBIT(newDev.file.Fd(), uintptr(ev)); err != nil {
			DestroyDevice(newDev)
			return nil, fmt.Errorf("failed to set ev bit: %d - %w", ev, err)
		}

		if err := setEventCodes(newDev, ev, codes); err != nil {
			DestroyDevice(newDev)
			return nil, fmt.Errorf("failed to set ev code: %w", err)
		}
	}

	if _, err = createInputDevice(newDev.file, UinputUserDevice{
		Name: toUinputName([]byte(name)),
		ID:   id,
	}); err != nil {
		DestroyDevice(newDev)
		return nil, fmt.Errorf("failed to create device: %w", err)
	}

	return newDev, nil
}

// CloneDevice creates a new device from an existing one
// all capabilites will be coppied over to the new virtual device
// If set up fails the device will be removed from the system,
// once set up it can be removed by calling dev.Close
func CloneDevice(name string, dev *InputDevice) (*InputDevice, error) {
	deviceFile, err := os.OpenFile("/dev/uinput", syscall.O_WRONLY|syscall.O_NONBLOCK, 0660)
	if err != nil {
		return nil, err
	}

	newDev := &InputDevice{
		file:          deviceFile,
		driverVersion: dev.driverVersion,
	}

	for _, ev := range dev.CapableTypes() {
		if err := ioctlUISETEVBIT(newDev.file.Fd(), uintptr(ev)); err != nil {
			DestroyDevice(newDev)
			return nil, fmt.Errorf("failed to set ev bit: %d - %w", ev, err)
		}

		eventCodes := dev.CapableEvents(ev)
		if err := setEventCodes(newDev, ev, eventCodes); err != nil {
			DestroyDevice(newDev)
			return nil, fmt.Errorf("failed to set ev code: %w", err)
		}
	}

	id, err := dev.InputID()
	if err != nil {
		DestroyDevice(newDev)
		return nil, fmt.Errorf("failed to get original device id: %w", err)
	}

	if _, err = createInputDevice(newDev.file, UinputUserDevice{
		Name: toUinputName([]byte(name)),
		ID:   id,
	}); err != nil {
		return nil, fmt.Errorf("failed to create device: %w", err)
	}

	return newDev, nil
}

// Destroy destroys an input device, removing it from the system
// This is designed to be called on self created virtual devices and may fail if called
// on real devices attached to the system
func DestroyDevice(dev *InputDevice) error {
	return ioctlUIDEVDESTROY(dev.file.Fd())
}

func setEventCodes(dev *InputDevice, ev EvType, codes []EvCode) error {
	for _, code := range codes {
		var err error

		switch ev {
		case EV_ABS:
			err = ioctlUISETABSBIT(dev.file.Fd(), uintptr(code))
		case EV_FF:
			err = ioctlUISETFFBIT(dev.file.Fd(), uintptr(code))
		case EV_KEY:
			err = ioctlUISETKEYBIT(dev.file.Fd(), uintptr(code))
		case EV_LED:
			err = ioctlUISETLEDBIT(dev.file.Fd(), uintptr(code))
		case EV_MSC:
			err = ioctlUISETMSCBIT(dev.file.Fd(), uintptr(code))
		case EV_REL:
			err = ioctlUISETRELBIT(dev.file.Fd(), uintptr(code))
		case EV_SND:
			err = ioctlUISETSNDBIT(dev.file.Fd(), uintptr(code))
		case EV_SW:
			err = ioctlUISETSWBIT(dev.file.Fd(), uintptr(code))
		}

		if err != nil {
			return err
		}
	}

	return nil
}

func toUinputName(name []byte) (uinputName [uinputMaxNameSize]byte) {
	var fixedSizeName [uinputMaxNameSize]byte
	copy(fixedSizeName[:], name)

	return fixedSizeName
}

func createInputDevice(file *os.File, dev UinputUserDevice) (fd *os.File, err error) {
	buf := new(bytes.Buffer)

	if err = binary.Write(buf, binary.LittleEndian, dev); err != nil {
		file.Close()
		return nil, fmt.Errorf("failed to write user device buffer: %w", err)
	}

	if _, err = file.Write(buf.Bytes()); err != nil {
		file.Close()
		return nil, fmt.Errorf("failed to write uidev struct to device file: %w", err)
	}

	if err = ioctlUIDEVCREATE(file.Fd()); err != nil {
		file.Close()
		return nil, fmt.Errorf("failed to create device: %w", err)
	}

	return file, nil
}
