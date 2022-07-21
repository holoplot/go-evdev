package evdev

import (
	"errors"
	"fmt"
	"strings"
	"syscall"
	"unsafe"
)

const (
	ioctlDirNone  = 0x0
	ioctlDirWrite = 0x1
	ioctlDirRead  = 0x2
)

func trimNull(s string) string {
	return strings.Trim(s, "\x00")
}

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

func ioctlEVIOCGREP(fd uintptr) ([2]uint32, error) {
	rep := [2]uint32{}
	code := ioctlMakeCode(ioctlDirRead, 'E', 0x03, unsafe.Sizeof(rep))
	err := doIoctl(fd, code, unsafe.Pointer(&rep))
	return rep, err
}

func ioctlEVIOCSREP(fd uintptr, rep [2]uint32) error {
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
	return trimNull(string(str[:])), err
}

func ioctlEVIOCGPHYS(fd uintptr) (string, error) {
	str := [256]byte{}
	code := ioctlMakeCode(ioctlDirRead, 'E', 0x07, unsafe.Sizeof(str))
	err := doIoctl(fd, code, unsafe.Pointer(&str))
	return trimNull(string(str[:])), err
}

func ioctlEVIOCGUNIQ(fd uintptr) (string, error) {
	str := [256]byte{}
	code := ioctlMakeCode(ioctlDirRead, 'E', 0x08, unsafe.Sizeof(str))
	err := doIoctl(fd, code, unsafe.Pointer(&str))
	return trimNull(string(str[:])), err
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
	var cnt int

	switch evtype {
	case 0:
		// special case, indicating the list of all feature types supported should be returned,
		// rather than the list of particular features for that type
		cnt = EV_CNT
	case EV_KEY:
		cnt = KEY_CNT
	case EV_REL:
		cnt = REL_CNT
	case EV_ABS:
		cnt = ABS_CNT
	case EV_MSC:
		cnt = MSC_CNT
	case EV_SW:
		cnt = SW_CNT
	case EV_LED:
		cnt = LED_CNT
	case EV_SND:
		cnt = SND_CNT
	case EV_REP:
		cnt = REP_CNT
	case EV_FF:
		cnt = FF_CNT
	default: // EV_PWR, EV_FF_STATUS ??
		cnt = KEY_MAX
	}

	bytesNumber := (cnt + 7) / 8

	bits := [KEY_MAX]byte{}
	code := ioctlMakeCode(ioctlDirRead, 'E', 0x20+evtype, unsafe.Sizeof(bits))
	err := doIoctl(fd, code, unsafe.Pointer(&bits))
	return bits[:bytesNumber], err
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

func ioctlEVIOCGRAB(fd uintptr, p int32) error {
	code := ioctlMakeCode(ioctlDirWrite, 'E', 0x90, unsafe.Sizeof(p))
	if p != 0 {
		return doIoctl(fd, code, unsafe.Pointer(&p))
	}
	return doIoctl(fd, code, nil)
}

func ioctlEVIOCREVOKE(fd uintptr) error {
	var p int32
	code := ioctlMakeCode(ioctlDirWrite, 'E', 0x91, unsafe.Sizeof(p))
	return doIoctl(fd, code, nil)
}

func ioctlUISETEVBIT(fd uintptr, ev uintptr) error {
	var p int32
	code := ioctlMakeCode(ioctlDirWrite, 'U', 100, unsafe.Sizeof(p))
	return doIoctl(fd, code, unsafe.Pointer(ev))
}

func ioctlUISETKEYBIT(fd uintptr, key uintptr) error {
	var p int32
	code := ioctlMakeCode(ioctlDirWrite, 'U', 101, unsafe.Sizeof(p))
	return doIoctl(fd, code, unsafe.Pointer(key))
}

func ioctlUISETRELBIT(fd uintptr, rel uintptr) error {
	var p int32
	code := ioctlMakeCode(ioctlDirWrite, 'U', 102, unsafe.Sizeof(p))
	return doIoctl(fd, code, unsafe.Pointer(rel))
}

func ioctlUISETABSBIT(fd uintptr, abs uintptr) error {
	var p int32
	code := ioctlMakeCode(ioctlDirWrite, 'U', 103, unsafe.Sizeof(p))
	return doIoctl(fd, code, unsafe.Pointer(abs))
}

func ioctlUISETMSCBIT(fd uintptr, msc uintptr) error {
	var p int32
	code := ioctlMakeCode(ioctlDirWrite, 'U', 104, unsafe.Sizeof(p))
	return doIoctl(fd, code, unsafe.Pointer(msc))
}

func ioctlUISETLEDBIT(fd uintptr, led uintptr) error {
	var p int32
	code := ioctlMakeCode(ioctlDirWrite, 'U', 105, unsafe.Sizeof(p))
	return doIoctl(fd, code, unsafe.Pointer(led))
}

func ioctlUISETSNDBIT(fd uintptr, snd uintptr) error {
	var p int32
	code := ioctlMakeCode(ioctlDirWrite, 'U', 106, unsafe.Sizeof(p))
	return doIoctl(fd, code, unsafe.Pointer(snd))
}

func ioctlUISETFFBIT(fd uintptr, fe uintptr) error {
	var p int32
	code := ioctlMakeCode(ioctlDirWrite, 'U', 107, unsafe.Sizeof(p))
	return doIoctl(fd, code, unsafe.Pointer(fe))
}

func ioctlUISETSWBIT(fd uintptr, sw uintptr) error {
	var p int32
	code := ioctlMakeCode(ioctlDirWrite, 'U', 109, unsafe.Sizeof(p))
	return doIoctl(fd, code, unsafe.Pointer(sw))
}

func ioctlUISETPROPBIT(fd uintptr, prop uintptr) error {
	var p int32
	code := ioctlMakeCode(ioctlDirWrite, 'U', 110, unsafe.Sizeof(p))
	return doIoctl(fd, code, unsafe.Pointer(prop))
}

func ioctlUIDEVCREATE(fd uintptr) error {
	code := ioctlMakeCode(ioctlDirNone, 'U', 1, 0)
	return doIoctl(fd, code, nil)
}

func ioctlUIDEVDESTROY(fd uintptr) error {
	code := ioctlMakeCode(ioctlDirNone, 'U', 2, 0)
	return doIoctl(fd, code, nil)
}
