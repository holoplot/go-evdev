package kbext

import (
	"errors"
	"log"

	evdev "github.com/holoplot/go-evdev"
)

type KbState struct {
	layout LayoutID
	shift  bool
	altgr  bool
}

var (
	ErrNotACharKey    = errors.New("last event was not a printable char key down")
	ErrNotHandled     = errors.New("event not handled")
	ErrUnknwownLayout = errors.New("unknown layout")
)

func NewKbState(layout LayoutID) KbState {
	return KbState{
		layout: layout,
		shift:  false,
		altgr:  false,
	}
}

func (kb *KbState) KeyEvent(kbEvt *evdev.KeyEvent) (string, error) {
	switch kbEvt.State {
	case evdev.KeyUp:
		_, err := kb.handleKey(kbEvt, false)
		if err == nil {
			err = ErrNotACharKey
		}
		return "", err
	case evdev.KeyDown:
		return kb.handleKey(kbEvt, true)
	}
	return "", ErrNotHandled
}

func (kb *KbState) handleKey(kbEvt *evdev.KeyEvent, down bool) (string, error) {
	switch kbEvt.Scancode {
	case evdev.KEY_LEFTSHIFT:
		fallthrough
	case evdev.KEY_RIGHTSHIFT:
		kb.shift = down
		return "", ErrNotACharKey

	case evdev.KEY_ENTER:
		fallthrough
	case evdev.KEY_KPENTER:
		return "\n", nil
	}

	layout, ok := layouts[kb.layout]
	if !ok {
		return "", ErrUnknwownLayout
	}

	keyinfo, ok := layout[kbEvt.Scancode]
	if !ok {
		return "", errors.New("KeyInfo not found")
	}

	if kb.altgr {
		if keyinfo.plain != keyinfo.altgr {
			return keyinfo.altgr, nil
		}
		keyName := evdev.KEYToString[kbEvt.Scancode]
		log.Printf("TODO : altgr mapping for %v", keyName)
		return keyinfo.altgr, nil
	}
	if kb.shift {
		if keyinfo.plain != keyinfo.shift {
			return keyinfo.shift, nil
		}
		keyName := evdev.KEYToString[kbEvt.Scancode]
		log.Printf("TODO : shift mapping for %v", keyName)
		return keyinfo.shift, nil
	}
	return keyinfo.plain, nil
}
