package notifyosc

import (
	"bytes"
	"strings"
	"testing"
	"unicode/utf8"
)

func TestSanitizeRepairsStripsAndBounds(t *testing.T) {
	raw := append([]byte{0xff, 0x00, 'a', 0x7f}, []byte("\u0080世")...)
	got := Sanitize(raw)
	if got != "�a世" {
		t.Fatalf("Sanitize() = %q, want %q", got, "�a世")
	}
	if !utf8.ValidString(got) || len(got) > MaxMessageBytes {
		t.Fatalf("invalid sanitized result: valid=%v bytes=%d", utf8.ValidString(got), len(got))
	}

	bounded := Sanitize([]byte(strings.Repeat("a", MaxMessageBytes-1) + "世"))
	if len(bounded) != MaxMessageBytes-1 || !utf8.ValidString(bounded) {
		t.Fatalf("rune-boundary result has %d bytes, valid=%v", len(bounded), utf8.ValidString(bounded))
	}
}

func TestEncodeDecodeCanonicalEnvelope(t *testing.T) {
	want := "build; finished"
	encoded := Encode(want)
	if !bytes.Equal(encoded, []byte("\x1b]777;notify;pair;build; finished\x07")) {
		t.Fatalf("Encode() = %q", encoded)
	}
	got, ok := DecodeOSC(encoded)
	if !ok || got.Message != want {
		t.Fatalf("DecodeOSC() = %+v, %v", got, ok)
	}

	st := append(append([]byte(nil), encoded[:len(encoded)-1]...), 0x1b, '\\')
	got, ok = DecodeOSC(st)
	if !ok || got.Message != want {
		t.Fatalf("DecodeOSC(ST) = %+v, %v", got, ok)
	}
}

func TestDecodeOSCRejectsNearCanonicalInput(t *testing.T) {
	inputs := [][]byte{
		[]byte("\x1b]9;hello\x07"),
		[]byte("\x1b]777;notify;other;hello\x07"),
		[]byte("\x1b]777;notify;pair;hello"),
		[]byte("prefix\x1b]777;notify;pair;hello\x07"),
		[]byte("\x1b]777;notify;pair;hello\x07suffix"),
	}
	for _, input := range inputs {
		if got, ok := DecodeOSC(input); ok {
			t.Fatalf("DecodeOSC(%q) = %+v, true", input, got)
		}
	}
}

func FuzzCodec(f *testing.F) {
	f.Add([]byte("hello"))
	f.Add([]byte{0xff, 0x1b, 0x07, 'x'})
	f.Fuzz(func(t *testing.T, raw []byte) {
		clean := Sanitize(raw)
		if !utf8.ValidString(clean) || len(clean) > MaxMessageBytes {
			t.Fatalf("Sanitize invariant failed: valid=%v bytes=%d", utf8.ValidString(clean), len(clean))
		}
		for _, r := range clean {
			if r <= 0x1f || r >= 0x7f && r <= 0x9f {
				t.Fatalf("Sanitize retained control U+%04X", r)
			}
		}
		got, ok := DecodeOSC(Encode(string(raw)))
		if !ok || got.Message != clean {
			t.Fatalf("round trip = %+v, %v; want %q", got, ok, clean)
		}
	})
}
