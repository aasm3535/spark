package terminal

import "gioui.org/io/key"

// KeyToBytes converts a Gio key event to the corresponding byte sequence
// that should be sent to the PTY.
func KeyToBytes(e key.Event, appCursorKeys bool) []byte {
	if e.State != key.Press {
		return nil
	}

	ctrl := e.Modifiers.Contain(key.ModCtrl)
	shift := e.Modifiers.Contain(key.ModShift)

	if ctrl {
		switch e.Name {
		case "A":
			return []byte{1}
		case "B":
			return []byte{2}
		case "C":
			return []byte{3}
		case "D":
			return []byte{4}
		case "E":
			return []byte{5}
		case "F":
			return []byte{6}
		case "G":
			return []byte{7}
		case "H":
			return []byte{8}
		case "I":
			return []byte{9}
		case "J":
			return []byte{10}
		case "K":
			return []byte{11}
		case "L":
			return []byte{12}
		case "M":
			return []byte{13}
		case "N":
			return []byte{14}
		case "O":
			return []byte{15}
		case "P":
			return []byte{16}
		case "Q":
			return []byte{17}
		case "R":
			return []byte{18}
		case "S":
			return []byte{19}
		case "T":
			return []byte{20}
		case "U":
			return []byte{21}
		case "V":
			return []byte{22}
		case "W":
			return []byte{23}
		case "X":
			return []byte{24}
		case "Y":
			return []byte{25}
		case "Z":
			return []byte{26}
		case "[", key.NameEscape:
			return []byte{27}
		case "\\":
			return []byte{28}
		case "]":
			return []byte{29}
		case "^", "6":
			return []byte{30}
		case "_", "-":
			return []byte{31}
		case key.NameDeleteBackward:
			return []byte{8}
		}
	}

	switch e.Name {
	case key.NameReturn, key.NameEnter:
		return []byte{'\r'}
	case key.NameDeleteBackward:
		return []byte{127}
	case key.NameDeleteForward:
		return []byte{'\x1b', '[', '3', '~'}
	case key.NameTab:
		if shift {
			return []byte{'\x1b', '[', 'Z'}
		}
		return []byte{'\t'}
	case key.NameEscape:
		return []byte{'\x1b'}
	case key.NameSpace:
		return []byte{' '}
	case key.NameUpArrow:
		if appCursorKeys {
			return []byte{'\x1b', 'O', 'A'}
		}
		return []byte{'\x1b', '[', 'A'}
	case key.NameDownArrow:
		if appCursorKeys {
			return []byte{'\x1b', 'O', 'B'}
		}
		return []byte{'\x1b', '[', 'B'}
	case key.NameRightArrow:
		if appCursorKeys {
			return []byte{'\x1b', 'O', 'C'}
		}
		return []byte{'\x1b', '[', 'C'}
	case key.NameLeftArrow:
		if appCursorKeys {
			return []byte{'\x1b', 'O', 'D'}
		}
		return []byte{'\x1b', '[', 'D'}
	case key.NameHome:
		return []byte{'\x1b', '[', 'H'}
	case key.NameEnd:
		return []byte{'\x1b', '[', 'F'}
	case key.NamePageUp:
		return []byte{'\x1b', '[', '5', '~'}
	case key.NamePageDown:
		return []byte{'\x1b', '[', '6', '~'}
	case "F1":
		return []byte{'\x1b', 'O', 'P'}
	case "F2":
		return []byte{'\x1b', 'O', 'Q'}
	case "F3":
		return []byte{'\x1b', 'O', 'R'}
	case "F4":
		return []byte{'\x1b', 'O', 'S'}
	case "F5":
		return []byte{'\x1b', '[', '1', '5', '~'}
	case "F6":
		return []byte{'\x1b', '[', '1', '7', '~'}
	case "F7":
		return []byte{'\x1b', '[', '1', '8', '~'}
	case "F8":
		return []byte{'\x1b', '[', '1', '9', '~'}
	case "F9":
		return []byte{'\x1b', '[', '2', '0', '~'}
	case "F10":
		return []byte{'\x1b', '[', '2', '1', '~'}
	case "F11":
		return []byte{'\x1b', '[', '2', '3', '~'}
	case "F12":
		return []byte{'\x1b', '[', '2', '4', '~'}
	}

	return nil
}
