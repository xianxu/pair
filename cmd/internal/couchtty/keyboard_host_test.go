package couchtty

import (
	"bytes"
	"context"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/xianxu/pair/cmd/internal/ansi"
	"github.com/xianxu/pair/cmd/internal/hostty"
	"github.com/xianxu/pair/cmd/internal/ptychild"
)

// keyboardHost models the keyboard protocol independently of Couch's policy.
// It consumes actual terminal output, then encodes a physical key according to
// the resulting mode. Strings are skipped with constant memory; only CSI
// parameters are retained, up to keyboardCSILimit bytes.
type keyboardHost struct {
	*hostty.FakeHost
	kmu   sync.Mutex
	model keyboardModel
}

func newKeyboardHost(rows, cols uint16) *keyboardHost {
	return &keyboardHost{FakeHost: hostty.NewFakeHost(ptychild.Size{Rows: rows, Cols: cols})}
}

func (h *keyboardHost) Write(p []byte) (int, error) {
	h.kmu.Lock()
	defer h.kmu.Unlock()
	h.model.feed(p)
	return h.FakeHost.Write(p)
}

func (h *keyboardHost) flags() uint32 {
	h.kmu.Lock()
	defer h.kmu.Unlock()
	return h.model.buffers[h.model.active].current
}

// depth counts saved entries, excluding the currently effective flags.
func (h *keyboardHost) depth() int {
	h.kmu.Lock()
	defer h.kmu.Unlock()
	return len(h.model.buffers[h.model.active].saved)
}

func (h *keyboardHost) ctrlReturn() []byte {
	f := h.flags()
	if f&(1|8) == 0 {
		return []byte("\r")
	}
	if f&2 != 0 {
		return []byte("\x1b[13;5:1u")
	}
	return []byte("\x1b[13;5u")
}

const keyboardCSILimit = 256
const keyboardStackLimit = 16

type keyboardBuffer struct {
	current uint32
	saved   []uint32
}

type keyboardModel struct {
	buffers   [2]keyboardBuffer
	active    int
	state     byte // zero, ESC, CSI, OSC, or ST-terminated string
	stringESC bool
	csi       []byte
	oversize  bool
}

func (m *keyboardModel) feed(p []byte) {
	for _, b := range p {
		switch m.state {
		case ']', 'P':
			if (m.state == ']' && b == 7) || (m.stringESC && b == '\\') {
				m.state, m.stringESC = 0, false
			} else {
				m.stringESC = b == 0x1b
			}
		case '[':
			if b == 0x1b {
				m.state, m.csi, m.oversize = 0x1b, nil, false
			} else if ansi.IsFinalByte(b) {
				if !m.oversize {
					m.command(m.csi, b)
				}
				m.state, m.csi, m.oversize = 0, nil, false
			} else if len(m.csi) < keyboardCSILimit {
				m.csi = append(m.csi, b)
			} else {
				m.oversize = true
			}
		case 0x1b:
			m.state = 0
			switch b {
			case '[':
				m.state = '['
			case ']':
				m.state = ']'
			case 'P', '_', '^', 'X':
				m.state = 'P'
			case 0x1b:
				m.state = 0x1b
			case 'c':
				*m = keyboardModel{}
			}
		default:
			if b == 0x1b {
				m.state = 0x1b
			}
		}
	}
}

func (m *keyboardModel) command(params []byte, final byte) {
	if len(params) == 0 {
		return
	}
	if params[0] == '?' && (final == 'h' || final == 'l') {
		for _, p := range strings.Split(string(params[1:]), ";") {
			if p == "47" || p == "1047" || p == "1049" {
				m.active = 0
				if final == 'h' {
					m.active = 1
				}
			}
		}
		return
	}
	if final != 'u' || !bytes.ContainsRune([]byte("=><"), rune(params[0])) {
		return
	}
	parts := strings.Split(string(params[1:]), ";")
	value := uint64(0)
	if parts[0] != "" {
		var err error
		value, err = strconv.ParseUint(parts[0], 10, 32)
		if err != nil {
			return
		}
	}
	b := &m.buffers[m.active]
	switch params[0] {
	case '=':
		mode := "1"
		if len(parts) > 1 && parts[1] != "" {
			mode = parts[1]
		}
		if len(parts) > 2 {
			return
		}
		switch mode {
		case "1":
			b.current = uint32(value)
		case "2":
			b.current |= uint32(value)
		case "3":
			b.current &^= uint32(value)
		}
	case '>':
		if len(parts) != 1 {
			return
		}
		if len(b.saved) == keyboardStackLimit {
			copy(b.saved, b.saved[1:])
			b.saved = b.saved[:keyboardStackLimit-1]
		}
		b.saved = append(b.saved, b.current)
		b.current = uint32(value)
	case '<':
		if len(parts) != 1 {
			return
		}
		if parts[0] == "" {
			value = 1
		}
		if value == 0 {
			return
		}
		if value > uint64(len(b.saved)) {
			b.current, b.saved = 0, nil
		} else {
			index := len(b.saved) - int(value)
			b.current = b.saved[index]
			b.saved = b.saved[:index]
		}
	}
}

func TestKeyboardHostProtocol(t *testing.T) {
	h := newKeyboardHost(24, 80)
	steps := []struct {
		output string
		flags  uint32
		depth  int
		key    string
	}{
		{"", 0, 0, "\r"},
		{"\x1b[=4u\x1b[=1;2u", 5, 0, "\x1b[13;5u"},
		{"\x1b[=1;3u", 4, 0, "\r"},
		{"\x1b[>11u", 11, 1, "\x1b[13;5:1u"},
		{"\x1b[>u", 0, 2, "\r"},
		{"\x1b[<u", 11, 1, "\x1b[13;5:1u"},
		{"\x1b[<0u", 11, 1, "\x1b[13;5:1u"},
		{"\x1b[<2u", 0, 0, "\r"},
		{"\x1b[=8u", 8, 0, "\x1b[13;5u"},
		{"\x1b[?1049h", 0, 0, "\r"},
		{"\x1b[>3u", 3, 1, "\x1b[13;5:1u"},
		{"\x1b[?1049l", 8, 0, "\x1b[13;5u"},
		{"\x1b[?1047h", 3, 1, "\x1b[13;5:1u"},
		{"\x1bc", 0, 0, "\r"},
		{"\x1b[?47h", 0, 0, "\r"},
	}
	for i, s := range steps {
		_, _ = h.Write([]byte(s.output))
		if h.flags() != s.flags || h.depth() != s.depth || string(h.ctrlReturn()) != s.key {
			t.Fatalf("step %d: flags=%d depth=%d key=%q; want %d %d %q", i, h.flags(), h.depth(), h.ctrlReturn(), s.flags, s.depth, s.key)
		}
	}
}

func TestKeyboardHostBoundsAndControlStrings(t *testing.T) {
	h := newKeyboardHost(24, 80)
	for i := 1; i <= 100; i++ {
		_, _ = h.Write([]byte(fmt.Sprintf("\x1b[>%du", i)))
	}
	if h.depth() != keyboardStackLimit {
		t.Fatalf("unbounded depth: %d", h.depth())
	}
	_, _ = h.Write([]byte("\x1b[<16u"))
	if h.flags() != 84 || h.depth() != 0 {
		t.Fatalf("oldest entry not evicted: flags %d depth %d", h.flags(), h.depth())
	}
	_, _ = h.Write([]byte("\x1b[<u"))
	if h.flags() != 0 {
		t.Fatal("underflow did not reset")
	}
	for _, intro := range []string{"\x1b]", "\x1bP", "\x1b_"} {
		payload := intro + strings.Repeat("x", 70000) + "\x1b[=1u\x1bc\x1b\\"
		for _, b := range []byte(payload) {
			h.model.feed([]byte{b})
		}
		if h.flags() != 0 || len(h.model.csi) > keyboardCSILimit {
			t.Fatal("string payload interpreted or retained")
		}
	}
	_, _ = h.Write([]byte("\x1b[=" + strings.Repeat("0", 70000) + "1u\x1b[=8u"))
	if h.flags() != 8 || len(h.model.csi) != 0 {
		t.Fatal("oversized CSI did not recover")
	}
}

func TestKeyboardHostEveryPartition(t *testing.T) {
	streams := []string{
		"\x1b[=4u\x1b[>3u\x1b[=1;2u\x1b[<u\x1b[?1049h\x1b[>8u\x1b[?1049l",
		"\x1b]title\x1b[=1u\x07\x1b[=8u",
		"\x1bPpayload\x07\x1b[=1u\x1b\\\x1b[=4u",
		"\x1b_payload\x1bc\x1b\\\x1b[>3u",
	}
	for _, stream := range streams {
		var whole keyboardModel
		whole.feed([]byte(stream))
		for cut := 0; cut <= len(stream); cut++ {
			var split keyboardModel
			split.feed([]byte(stream[:cut]))
			split.feed([]byte(stream[cut:]))
			if !reflect.DeepEqual(whole, split) {
				t.Fatalf("partition %d of %q differs", cut, stream)
			}
		}
	}
}

func FuzzKeyboardHostPartitions(f *testing.F) {
	f.Add([]byte("\x1b[>3u\x1b]title\x1b[=0u\x07\x1b[<u"), uint8(3))
	f.Add([]byte("\x1bP\x1b[=1u\x07\x1b\\\x1b[?1049h"), uint8(1))
	f.Fuzz(func(t *testing.T, p []byte, width uint8) {
		var whole, split keyboardModel
		whole.feed(p)
		step := int(width) + 1
		for start := 0; start < len(p); start += step {
			end := start + step
			if end > len(p) {
				end = len(p)
			}
			split.feed(p[start:end])
		}
		if !reflect.DeepEqual(whole, split) {
			t.Fatal("partition changed keyboard state")
		}
		if len(split.csi) > keyboardCSILimit || len(split.buffers[0].saved) > keyboardStackLimit || len(split.buffers[1].saved) > keyboardStackLimit {
			t.Fatal("unbounded retained state")
		}
	})
}

func (h *keyboardHost) WriteContext(ctx context.Context, p []byte) (int, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	return h.Write(p)
}
