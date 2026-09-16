package vt

import (
	"fmt"
	uv "github.com/charmbracelet/ultraviolet"
	"github.com/charmbracelet/x/ansi"
	"io"
	"strconv"
	"strings"
)

type keyboardState struct {
	flags uint32
	stack []uint32
}

func (e *Emulator) registerKeyboardHandlers() {
	e.RegisterCsiHandler(ansi.Command('?', 0, 'u'), func(ansi.Params) bool { fmt.Fprintf(e.replies(), "\x1b[?%du", e.KeyboardFlags()); return true })
	e.RegisterCsiHandler(ansi.Command('>', 0, 'u'), func(p ansi.Params) bool {
		k := &e.keyboard[e.screenIndex()]
		n, _, _ := p.Param(0, 0)
		if len(k.stack) == e.limits.KeyboardStack {
			copy(k.stack, k.stack[1:])
			k.stack = k.stack[:len(k.stack)-1]
		}
		k.stack = append(k.stack, k.flags)
		k.flags = uint32(n) & 31
		return true
	})
	e.RegisterCsiHandler(ansi.Command('<', 0, 'u'), func(p ansi.Params) bool {
		k := &e.keyboard[e.screenIndex()]
		n, _, _ := p.Param(0, 1)
		if n < 1 {
			n = 1
		}
		for ; n > 0 && len(k.stack) > 0; n-- {
			k.flags = k.stack[len(k.stack)-1]
			k.stack = k.stack[:len(k.stack)-1]
		}
		if n > 0 {
			k.flags = 0
		}
		return true
	})
	e.RegisterCsiHandler(ansi.Command('=', 0, 'u'), func(p ansi.Params) bool {
		k := &e.keyboard[e.screenIndex()]
		n, _, _ := p.Param(0, 0)
		how, _, _ := p.Param(1, 1)
		switch how {
		case 1:
			k.flags = uint32(n) & 31
		case 2:
			k.flags |= uint32(n) & 31
		case 3:
			k.flags &^= uint32(n) & 31
		}
		return true
	})
}

// sendExtendedKey handles negotiated keys and modified legacy function keys.
func (e *Emulator) sendExtendedKey(ev uv.KeyEvent) bool {
	k := ev.Key()
	_, release := ev.(uv.KeyReleaseEvent)
	flags := e.KeyboardFlags()
	if release && flags&2 == 0 {
		return true
	}
	code, final, special := keyWireCode(k.Code)
	mod := kittyModifiers(k.Mod) + 1
	event := 1
	if release {
		event = 3
	} else if k.IsRepeat {
		event = 2
	}
	// Plain text, Enter, Tab and Backspace retain legacy encoding unless all-key
	// reporting or a modifier requires an escape representation.
	extended := flags&8 != 0 || flags&1 != 0 && (k.Mod & ^(uv.ModCapsLock|uv.ModNumLock|uv.ModShift) != 0 || special || k.Code == uv.KeyEscape || (k.Code < 32 || k.Code == 127) && k.Mod&uv.ModShift != 0) || flags&2 != 0 && special
	if k.Code >= uv.KeyLeftShift && k.Code <= uv.KeyIsoLevel5Shift && flags&8 == 0 {
		return true
	}
	if release && !extended {
		return true
	}
	if !extended {
		if special && mod > 1 {
			fmt.Fprintf(e.replies(), "\x1b[%d;%d%c", code, mod, final)
			return true
		}
		return false
	}
	if !special {
		code = int(k.Code)
		final = 'u'
	}
	if code > 0x10ffff {
		return true
	}
	codepart := strconv.Itoa(code)
	if flags&4 != 0 && final == 'u' && (k.ShiftedCode != 0 || k.BaseCode != 0) {
		codepart += ":"
		if k.ShiftedCode != 0 {
			codepart += strconv.Itoa(int(k.ShiftedCode))
		}
		if k.BaseCode != 0 {
			codepart += ":" + strconv.Itoa(int(k.BaseCode))
		}
	}
	seq := "\x1b[" + codepart
	if mod != 1 || flags&2 != 0 || flags&16 != 0 {
		seq += ";" + strconv.Itoa(mod)
		if flags&2 != 0 {
			seq += ":" + strconv.Itoa(event)
		}
	}
	if flags&16 != 0 && k.Text != "" && !release {
		var text []string
		for _, r := range k.Text {
			text = append(text, strconv.Itoa(int(r)))
		}
		seq += ";" + strings.Join(text, ":")
	}
	seq += string(final)
	io.WriteString(e.replies(), seq)
	return true
}
func kittyModifiers(m uv.KeyMod) int {
	n := 0
	for _, p := range []struct {
		in  uv.KeyMod
		out int
	}{{uv.ModShift, 1}, {uv.ModAlt, 2}, {uv.ModCtrl, 4}, {uv.ModSuper, 8}, {uv.ModHyper, 16}, {uv.ModMeta, 32}, {uv.ModCapsLock, 64}, {uv.ModNumLock, 128}} {
		if m&p.in != 0 {
			n |= p.out
		}
	}
	return n
}
func keyWireCode(c rune) (int, rune, bool) {
	switch c {
	case uv.KeyUp:
		return 1, 'A', true
	case uv.KeyDown:
		return 1, 'B', true
	case uv.KeyRight:
		return 1, 'C', true
	case uv.KeyLeft:
		return 1, 'D', true
	case uv.KeyHome:
		return 1, 'H', true
	case uv.KeyEnd:
		return 1, 'F', true
	case uv.KeyInsert:
		return 2, '~', true
	case uv.KeyDelete:
		return 3, '~', true
	case uv.KeyPgUp:
		return 5, '~', true
	case uv.KeyPgDown:
		return 6, '~', true
	case uv.KeyF1:
		return 1, 'P', true
	case uv.KeyF2:
		return 1, 'Q', true
	case uv.KeyF3:
		return 13, '~', true
	case uv.KeyF4:
		return 1, 'S', true
	case uv.KeyF5:
		return 15, '~', true
	case uv.KeyF6:
		return 17, '~', true
	case uv.KeyF7:
		return 18, '~', true
	case uv.KeyF8:
		return 19, '~', true
	case uv.KeyF9:
		return 20, '~', true
	case uv.KeyF10:
		return 21, '~', true
	case uv.KeyF11:
		return 23, '~', true
	case uv.KeyF12:
		return 24, '~', true
	}
	// Ultraviolet internal key constants are not Kitty wire numbers.
	if c >= uv.KeyF13 && c <= uv.KeyF35 {
		return 57376 + int(c-uv.KeyF13), 'u', true
	}
	if n, ok := kittyFunctional[c]; ok {
		return n, 'u', true
	}
	return int(c), 'u', false
}

// Kitty functional codes match the protocol table; UV uses internal rune IDs.
var kittyFunctional = map[rune]int{
	uv.KeyCapsLock:         57358,
	uv.KeyScrollLock:       57359,
	uv.KeyNumLock:          57360,
	uv.KeyPrintScreen:      57361,
	uv.KeyPause:            57362,
	uv.KeyMenu:             57363,
	uv.KeyKp0:              57399,
	uv.KeyKp1:              57400,
	uv.KeyKp2:              57401,
	uv.KeyKp3:              57402,
	uv.KeyKp4:              57403,
	uv.KeyKp5:              57404,
	uv.KeyKp6:              57405,
	uv.KeyKp7:              57406,
	uv.KeyKp8:              57407,
	uv.KeyKp9:              57408,
	uv.KeyKpDecimal:        57409,
	uv.KeyKpDivide:         57410,
	uv.KeyKpMultiply:       57411,
	uv.KeyKpMinus:          57412,
	uv.KeyKpPlus:           57413,
	uv.KeyKpEnter:          57414,
	uv.KeyKpEqual:          57415,
	uv.KeyKpSep:            57416,
	uv.KeyKpLeft:           57417,
	uv.KeyKpRight:          57418,
	uv.KeyKpUp:             57419,
	uv.KeyKpDown:           57420,
	uv.KeyKpPgUp:           57421,
	uv.KeyKpPgDown:         57422,
	uv.KeyKpHome:           57423,
	uv.KeyKpEnd:            57424,
	uv.KeyKpInsert:         57425,
	uv.KeyKpDelete:         57426,
	uv.KeyKpBegin:          57427,
	uv.KeyMediaPlay:        57428,
	uv.KeyMediaPause:       57429,
	uv.KeyMediaPlayPause:   57430,
	uv.KeyMediaReverse:     57431,
	uv.KeyMediaStop:        57432,
	uv.KeyMediaFastForward: 57433,
	uv.KeyMediaRewind:      57434,
	uv.KeyMediaNext:        57435,
	uv.KeyMediaPrev:        57436,
	uv.KeyMediaRecord:      57437,
	uv.KeyLowerVol:         57438,
	uv.KeyRaiseVol:         57439,
	uv.KeyMute:             57440,
	uv.KeyLeftShift:        57441,
	uv.KeyLeftCtrl:         57442,
	uv.KeyLeftAlt:          57443,
	uv.KeyLeftSuper:        57444,
	uv.KeyLeftHyper:        57445,
	uv.KeyLeftMeta:         57446,
	uv.KeyRightShift:       57447,
	uv.KeyRightCtrl:        57448,
	uv.KeyRightAlt:         57449,
	uv.KeyRightSuper:       57450,
	uv.KeyRightHyper:       57451,
	uv.KeyRightMeta:        57452,
	uv.KeyIsoLevel3Shift:   57453,
	uv.KeyIsoLevel5Shift:   57454,
}
