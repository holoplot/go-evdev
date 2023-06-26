package evdev

import (
	"encoding/binary"
	"fmt"
	"os"
	"syscall"
)

// InputDevice represent a Linux kernel input device in userspace.
// It can be used to query and write device properties, read input events,
// or grab it for exclusive access.
type InputDevice struct {
	file          *os.File
	driverVersion int32
}

// Open creates a new InputDevice from the given path. Returns an error if
// the device node could not be opened or its properties failed to read.
func Open(path string) (*InputDevice, error) {
	d := &InputDevice{}

	var err error
	d.file, err = os.OpenFile(path, os.O_RDWR, 0)
	if err != nil {
		return nil, err
	}

	d.driverVersion, err = ioctlEVIOCGVERSION(d.file.Fd())
	if err != nil {
		return nil, fmt.Errorf("cannot get driver version: %v", err)
	}

	return d, nil
}

// OpenByName creates a new InputDevice from the device name as reported by the kernel.
// Returns an error if the name does not exist, or the device node could
// not be opened or its properties failed to read.
func OpenByName(name string) (*InputDevice, error) {
	devices, err := ListDevicePaths()
	if err != nil {
		return nil, err
	}
	for _, d := range devices {
		if d.Name == name {
			return Open(d.Path)
		}
	}
	return nil, fmt.Errorf("could not find input device with name %q", name)
}

// Close releases the resources held by an InputDevice. After calling this
// function, the InputDevice is no longer operational.
func (d *InputDevice) Close() error {
	return d.file.Close()
}

// Path returns the device's node path it was opened under.
func (d *InputDevice) Path() string {
	return d.file.Name()
}

// DriverVersion returns the version of the Linux Evdev driver.
// The three ints returned by this function describe the major, minor and
// micro parts of the version code.
func (d *InputDevice) DriverVersion() (int, int, int) {
	return int(d.driverVersion >> 16),
		int((d.driverVersion >> 8) & 0xff),
		int((d.driverVersion >> 0) & 0xff)
}

// Name returns the device's name as reported by the kernel.
func (d *InputDevice) Name() (string, error) {
	return ioctlEVIOCGNAME(d.file.Fd())
}

// PhysicalLocation returns the device's physical location as reported by the kernel.
func (d *InputDevice) PhysicalLocation() (string, error) {
	return ioctlEVIOCGPHYS(d.file.Fd())
}

// UniqueID returns the device's unique identifier as reported by the kernel.
func (d *InputDevice) UniqueID() (string, error) {
	return ioctlEVIOCGUNIQ(d.file.Fd())
}

// InputID returns the device's vendor/product/busType/version information as reported by the kernel.
func (d *InputDevice) InputID() (InputID, error) {
	return ioctlEVIOCGID(d.file.Fd())
}

// CapableTypes returns a slice of EvType that are the device supports
func (d *InputDevice) CapableTypes() []EvType {
	var types []EvType

	evBits, err := ioctlEVIOCGBIT(d.file.Fd(), 0)
	if err != nil {
		return []EvType{}
	}

	evBitmap := newBitmap(evBits)

	for _, t := range evBitmap.setBits() {
		types = append(types, EvType(t))
	}

	return types
}

// CapableEvents returns a slice of EvCode that are the device supports for given EvType
func (d *InputDevice) CapableEvents(t EvType) []EvCode {
	var codes []EvCode

	evBits, err := ioctlEVIOCGBIT(d.file.Fd(), int(t))
	if err != nil {
		return []EvCode{}
	}

	evBitmap := newBitmap(evBits)

	for _, t := range evBitmap.setBits() {
		codes = append(codes, EvCode(t))
	}

	return codes
}

// Properties returns a slice of EvProp that are the device supports
func (d *InputDevice) Properties() []EvProp {
	var props []EvProp

	propBits, err := ioctlEVIOCGPROP(d.file.Fd())
	if err != nil {
		return []EvProp{}
	}

	propBitmap := newBitmap(propBits)

	for _, p := range propBitmap.setBits() {
		props = append(props, EvProp(p))
	}

	return props
}

// State return a StateMap for the given type. The map will be empty if the requested type
// is not supported by the device.
func (d *InputDevice) State(t EvType) (StateMap, error) {
	fd := d.file.Fd()

	evBits, err := ioctlEVIOCGBIT(fd, 0)
	if err != nil {
		return nil, fmt.Errorf("cannot get evBits: %v", err)
	}

	evBitmap := newBitmap(evBits)

	if !evBitmap.bitIsSet(int(t)) {
		return StateMap{}, nil
	}

	codeBits, err := ioctlEVIOCGBIT(fd, int(t))
	if err != nil {
		return nil, fmt.Errorf("cannot get evBits: %v", err)
	}

	codeBitmap := newBitmap(codeBits)

	var stateBits []byte

	switch t {
	case EV_KEY:
		stateBits, err = ioctlEVIOCGKEY(fd)
	case EV_SW:
		stateBits, err = ioctlEVIOCGSW(fd)
	case EV_LED:
		stateBits, err = ioctlEVIOCGLED(fd)
	case EV_SND:
		stateBits, err = ioctlEVIOCGSND(fd)
	default:
		err = fmt.Errorf("unsupported evType %d", t)
	}

	if err != nil {
		return nil, err
	}

	stateBitmap := newBitmap(stateBits)
	st := StateMap{}

	for _, code := range codeBitmap.setBits() {
		st[EvCode(code)] = stateBitmap.bitIsSet(code)
	}

	return st, nil
}

// AbsInfos returns the AbsInfo struct for all axis the device supports.
func (d *InputDevice) AbsInfos() (map[EvCode]AbsInfo, error) {
	a := make(map[EvCode]AbsInfo)

	absBits, err := ioctlEVIOCGBIT(d.file.Fd(), EV_ABS)
	if err != nil {
		return nil, fmt.Errorf("cannot get absBits: %v", err)
	}

	absBitmap := newBitmap(absBits)

	for _, abs := range absBitmap.setBits() {
		absInfo, err := ioctlEVIOCGABS(d.file.Fd(), abs)
		if err == nil {
			a[EvCode(abs)] = absInfo
		}
	}

	return a, nil
}

// Grab grabs the device for exclusive access. No other process will receive
// input events until the device instance is active.
func (d *InputDevice) Grab() error {
	return ioctlEVIOCGRAB(d.file.Fd(), 1)
}

// Ungrab releases a previously taken exclusive use with Grab().
func (d *InputDevice) Ungrab() error {
	return ioctlEVIOCGRAB(d.file.Fd(), 0)
}

// Revoke revokes device access
func (d *InputDevice) Revoke() error {
	return ioctlEVIOCREVOKE(d.file.Fd())
}

// NonBlock sets file descriptor into nonblocking mode.
// This way it is possible to interrupt ReadOne call by closing the device.
// Note: file.Fd() call will set file descriptor back to blocking mode so make sure your program
// is not using any other method than ReadOne after NonBlock call.
func (d *InputDevice) NonBlock() error {
	return syscall.SetNonblock(int(d.file.Fd()), true)
}

// ReadOne reads one InputEvent from the device. It blocks until an event has
// been received or an error has occurred.
func (d *InputDevice) ReadOne() (*InputEvent, error) {
	event := InputEvent{}

	err := binary.Read(d.file, binary.LittleEndian, &event)
	if err != nil {
		return nil, err
	}

	return &event, nil
}

// WriteOne writes one InputEvent to the device.
// Useful for controlling LEDs of the device
func (d *InputDevice) WriteOne(event *InputEvent) error {
	return binary.Write(d.file, binary.LittleEndian, event)
}
