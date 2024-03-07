package evdev

import (
	"fmt"
)

type KeyEventState uint8

const (
	KeyUp   KeyEventState = 0x0
	KeyDown KeyEventState = 0x1
	KeyHold KeyEventState = 0x2
)

// KeyEvents are used to describe state changes of keyboards, buttons,
// or other key-like devices.
type KeyEvent struct {
	Event    *InputEvent
	Scancode EvCode
	Keycode  uint16
	State    KeyEventState
}

func (kev *KeyEvent) New(ev *InputEvent) {
	kev.Event = ev
	kev.Keycode = 0 // :todo
	kev.Scancode = ev.Code

	switch ev.Value {
	case 0:
		kev.State = KeyUp
	case 2:
		kev.State = KeyHold
	case 1:
		kev.State = KeyDown
	}
}

func NewKeyEvent(ev *InputEvent) *KeyEvent {
	kev := &KeyEvent{}
	kev.New(ev)
	return kev
}

func (ev *KeyEvent) String() string {
	state := "unknown"

	switch ev.State {
	case KeyUp:
		state = "up"
	case KeyHold:
		state = "hold"
	case KeyDown:
		state = "down"
	}

	return fmt.Sprintf("key event %d (%d), (%s)", ev.Scancode, ev.Event.Code, state)
}
