package evdev

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"os"
	"syscall"
	"time"
)

const (
	uinputMaxNameSize = 80
	absSize           = 64
)

// CreateDevice from scratch with the given capabilities
func CreateDevice(uinputDev *UinputUserDevice, capabilities map[EvType][]EvCode, name string) (*InputDevice, error) {
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
			return nil, fmt.Errorf("failed to set ev bit: %d", ev)
		}

		if err := setEventCodes(newDev, ev, codes); err != nil {
			DestroyDevice(newDev)
			return nil, fmt.Errorf("failed to set ev code")
		}
	}

	_, err = createUsbDevice(newDev.file, UinputUserDevice{
		Name: toUinputName([]byte(name)),
		ID: InputID{
			BusType: 0x03,
			Vendor:  0x4712,
			Product: 0x0816,
			Version: 1,
		},
	})
	if err != nil {
		return nil, err
	}

	return newDev, nil
}

// CloneDevice from existing device
func CloneDevice(dev *InputDevice) (*InputDevice, error) {
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
			return nil, fmt.Errorf("failed to set ev bit: %d - %s", ev, err)
		}

		eventCodes := dev.CapableEvents(ev)
		if err := setEventCodes(newDev, ev, eventCodes); err != nil {
			DestroyDevice(newDev)
			return nil, fmt.Errorf("failed to set ev code %s", err)
		}
	}

	name, err := dev.Name()
	if err != nil {
		DestroyDevice(newDev)
		return nil, errors.New("failed to get original device name")
	}

	id, err := dev.InputID()
	if err != nil {
		DestroyDevice(newDev)
		return nil, errors.New("failed to get original device id")
	}

	_, err = createUsbDevice(newDev.file, UinputUserDevice{
		Name: toUinputName([]byte(name + "(clone)")),
		ID:   id,
	})
	if err != nil {
		return nil, err
	}

	return newDev, nil
}

func setEventCodes(dev *InputDevice, ev EvType, codes []EvCode) error {
	for _, code := range codes {
		switch ev {
		case EV_ABS:
			if err := ioctlUISETABSBIT(dev.file.Fd(), uintptr(code)); err != nil {
				return err
			}
		case EV_FF:
			if err := ioctlUISETFFBIT(dev.file.Fd(), uintptr(code)); err != nil {
				return err
			}
		case EV_KEY:
			if err := ioctlUISETKEYBIT(dev.file.Fd(), uintptr(code)); err != nil {
				return err
			}
		case EV_LED:
			if err := ioctlUISETLEDBIT(dev.file.Fd(), uintptr(code)); err != nil {
				return err
			}
		case EV_MSC:
			if err := ioctlUISETMSCBIT(dev.file.Fd(), uintptr(code)); err != nil {
				return err
			}
		case EV_REL:
			if err := ioctlUISETRELBIT(dev.file.Fd(), uintptr(code)); err != nil {
				return err
			}
		case EV_SND:
			if err := ioctlUISETSNDBIT(dev.file.Fd(), uintptr(code)); err != nil {
				return err
			}
		case EV_SW:
			if err := ioctlUISETSWBIT(dev.file.Fd(), uintptr(code)); err != nil {
				return err
			}
		}
	}

	return nil
}

func DestroyDevice(dev *InputDevice) error {
	return ioctlUIDEVDESTROY(dev.file.Fd())
}

func toUinputName(name []byte) (uinputName [uinputMaxNameSize]byte) {
	var fixedSizeName [uinputMaxNameSize]byte
	copy(fixedSizeName[:], name)
	return fixedSizeName
}

func createUsbDevice(file *os.File, dev UinputUserDevice) (fd *os.File, err error) {
	buf := new(bytes.Buffer)
	err = binary.Write(buf, binary.LittleEndian, dev)
	if err != nil {
		file.Close()
		return nil, fmt.Errorf("failed to write user device buffer: %v", err)
	}
	_, err = file.Write(buf.Bytes())
	if err != nil {
		file.Close()
		return nil, fmt.Errorf("failed to write uidev struct to device file: %v", err)
	}

	err = ioctlUIDEVCREATE(file.Fd())
	if err != nil {
		file.Close()
		return nil, fmt.Errorf("failed to create device: %v", err)
	}

	time.Sleep(time.Millisecond * 200)

	return file, err
}
