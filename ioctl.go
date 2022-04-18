package evdev

import (
	"errors"
	"fmt"
	"syscall"
	"unsafe"
)

const (
	ioctlDirNone  = 0x0
	ioctlDirWrite = 0x1
	ioctlDirRead  = 0x2
)

func ioctlMakeCode(dir, typ, nr int, size uintptr) uint32 {
	var code uint32
	if dir > ioctlDirWrite|ioctlDirRead {
		panic(fmt.Errorf("invalid ioctl dir value: %d", dir))
	}

	if size > 1<<14 {
		panic(fmt.Errorf("invalid ioctl size value: %d", size))
	}

	code |= uint32(dir) << 30
	code |= uint32(size) << 16
	code |= uint32(typ) << 8
	code |= uint32(nr)

	return code
}

func doIoctl(fd uintptr, code uint32, ptr unsafe.Pointer) error {
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, fd, uintptr(code), uintptr(ptr))
	if errno != 0 {
		return errors.New(errno.Error())
	}

	return nil
}

func ioctlEVIOCGVERSION(fd uintptr) (int32, error) {
	version := int32(0)
	code := ioctlMakeCode(ioctlDirRead, 'E', 0x01, unsafe.Sizeof(version))
	err := doIoctl(fd, code, unsafe.Pointer(&version))
	return version, err
}

func ioctlEVIOCGID(fd uintptr) (InputID, error) {
	id := InputID{}
	code := ioctlMakeCode(ioctlDirRead, 'E', 0x02, unsafe.Sizeof(id))
	err := doIoctl(fd, code, unsafe.Pointer(&id))
	return id, err
}

func ioctlEVIOCGREP(fd uintptr) ([2]uint, error) {
	rep := [2]uint{}
	code := ioctlMakeCode(ioctlDirRead, 'E', 0x03, unsafe.Sizeof(rep))
	err := doIoctl(fd, code, unsafe.Pointer(&rep))
	return rep, err
}

func ioctlEVIOCSREP(fd uintptr, rep [2]uint) error {
	code := ioctlMakeCode(ioctlDirWrite, 'E', 0x03, unsafe.Sizeof(rep))
	return doIoctl(fd, code, unsafe.Pointer(&rep))
}

func ioctlEVIOCGKEYCODE(fd uintptr) (InputKeymapEntry, error) {
	entry := InputKeymapEntry{}
	code := ioctlMakeCode(ioctlDirRead, 'E', 0x04, unsafe.Sizeof(entry))
	err := doIoctl(fd, code, unsafe.Pointer(&entry))
	return entry, err
}

func ioctlEVIOCSKEYCODE(fd uintptr, entry InputKeymapEntry) error {
	code := ioctlMakeCode(ioctlDirWrite, 'E', 0x04, unsafe.Sizeof(entry))
	return doIoctl(fd, code, unsafe.Pointer(&entry))
}

func ioctlEVIOCGNAME(fd uintptr) (string, error) {
	str := [256]byte{}
	code := ioctlMakeCode(ioctlDirRead, 'E', 0x06, unsafe.Sizeof(str))
	err := doIoctl(fd, code, unsafe.Pointer(&str))
	return string(str[:]), err
}

func ioctlEVIOCGPHYS(fd uintptr) (string, error) {
	str := [256]byte{}
	code := ioctlMakeCode(ioctlDirRead, 'E', 0x07, unsafe.Sizeof(str))
	err := doIoctl(fd, code, unsafe.Pointer(&str))
	return string(str[:]), err
}

func ioctlEVIOCGUNIQ(fd uintptr) (string, error) {
	str := [256]byte{}
	code := ioctlMakeCode(ioctlDirRead, 'E', 0x08, unsafe.Sizeof(str))
	err := doIoctl(fd, code, unsafe.Pointer(&str))
	return string(str[:]), err
}

func ioctlEVIOCGPROP(fd uintptr) ([]byte, error) {
	bits := [256]byte{}
	code := ioctlMakeCode(ioctlDirRead, 'E', 0x09, unsafe.Sizeof(bits))
	err := doIoctl(fd, code, unsafe.Pointer(&bits))
	return bits[:], err
}

func ioctlEVIOCGKEY(fd uintptr) ([]byte, error) {
	bits := [KEY_MAX]byte{}
	code := ioctlMakeCode(ioctlDirRead, 'E', 0x18, unsafe.Sizeof(bits))
	err := doIoctl(fd, code, unsafe.Pointer(&bits))
	return bits[:], err
}

func ioctlEVIOCGLED(fd uintptr) ([]byte, error) {
	bits := [LED_MAX]byte{}
	code := ioctlMakeCode(ioctlDirRead, 'E', 0x19, unsafe.Sizeof(bits))
	err := doIoctl(fd, code, unsafe.Pointer(&bits))
	return bits[:], err
}

func ioctlEVIOCGSND(fd uintptr) ([]byte, error) {
	bits := [SND_MAX]byte{}
	code := ioctlMakeCode(ioctlDirRead, 'E', 0x1a, unsafe.Sizeof(bits))
	err := doIoctl(fd, code, unsafe.Pointer(&bits))
	return bits[:], err
}

func ioctlEVIOCGSW(fd uintptr) ([]byte, error) {
	bits := [SW_MAX]byte{}
	code := ioctlMakeCode(ioctlDirRead, 'E', 0x1b, unsafe.Sizeof(bits))
	err := doIoctl(fd, code, unsafe.Pointer(&bits))
	return bits[:], err
}

func ioctlEVIOCGBIT(fd uintptr, evtype int) ([]byte, error) {
	bits := [KEY_MAX]byte{}
	code := ioctlMakeCode(ioctlDirRead, 'E', 0x20+evtype, unsafe.Sizeof(bits))
	err := doIoctl(fd, code, unsafe.Pointer(&bits))
	return bits[:], err
}

func ioctlEVIOCGABS(fd uintptr, abs int) (AbsInfo, error) {
	info := AbsInfo{}
	code := ioctlMakeCode(ioctlDirRead, 'E', 0x40+abs, unsafe.Sizeof(info))
	err := doIoctl(fd, code, unsafe.Pointer(&info))
	return info, err
}

func ioctlEVIOCSABS(fd uintptr, abs int, info AbsInfo) error {
	code := ioctlMakeCode(ioctlDirWrite, 'E', 0xc0+abs, unsafe.Sizeof(info))
	return doIoctl(fd, code, unsafe.Pointer(&info))
}

func ioctlEVIOCGRAB(fd uintptr, p int) error {
	code := ioctlMakeCode(ioctlDirWrite, 'E', 0x90, unsafe.Sizeof(p))
	if p != 0 {
		return doIoctl(fd, code, unsafe.Pointer(&p))
	}
	return doIoctl(fd, code, nil)
}

func ioctlEVIOCREVOKE(fd uintptr) error {
	var p int
	code := ioctlMakeCode(ioctlDirWrite, 'E', 0x91, unsafe.Sizeof(p))
	return doIoctl(fd, code, nil)
}
