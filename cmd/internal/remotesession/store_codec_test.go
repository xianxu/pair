package remotesession

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDecodeLatchStrict(t *testing.T) {
	good := `{"version":1,"phase":"off","generation":0}`
	for _, raw := range []string{`null`, `[]`, good + ` {}`, `{"version":1,"phase":"off"}`, strings.Replace(good, `"off"`, `null`, 1), strings.Replace(good, `"off"`, `"OFF"`, 1), strings.Replace(good, `"version":1`, `"version":"1"`, 1), strings.Replace(good, `"version":1`, `"version":1,"Version":1`, 1), strings.Replace(good, `"version":1`, `"version":1,"version":1`, 1), strings.Replace(good, `"generation":0`, `"generation":-1`, 1), strings.Replace(good, `"generation":0`, `"generation":0.0`, 1), strings.Replace(good, `"generation":0`, `"generation":null`, 1), strings.Replace(good, `"generation":0`, `"generation":18446744073709551616`, 1), good[:len(good)-1] + `,"future":{"x":1,"x":2}}`, good[:len(good)-1] + `,"future":` + strings.Repeat(`[`, 32) + `0` + strings.Repeat(`]`, 32) + `}`} {
		if _, err := decodeLatch([]byte(raw)); err == nil {
			t.Errorf("accepted %s", raw)
		}
	}
	if _, err := decodeLatch([]byte(good)); err != nil {
		t.Fatal(err)
	}
}

func TestStorePreservesFutureFields(t *testing.T) {
	path := filepath.Join(t.TempDir(), "latch")
	future := json.RawMessage(`{ "nested" : [null, {"v":123456789012345678901234567890}], "escape":"a\\b" }`)
	raw := append([]byte(`{"version":1,"phase":"off","generation":7,"future":`), future...)
	raw = append(raw, '}')
	if err := os.WriteFile(path, raw, 0600); err != nil {
		t.Fatal(err)
	}
	s, err := OpenStore(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, transition := range []func() error{func() error { _, err := s.Arm(); return err }, s.Kill, s.consoleReset} {
		if err := transition(); err != nil {
			t.Fatal(err)
		}
		got, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		var fields map[string]json.RawMessage
		if err := json.Unmarshal(got, &fields); err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(fields["future"], future) {
			t.Fatalf("future changed: %s", fields["future"])
		}
	}
	reopened, err := OpenStore(path)
	if err != nil {
		t.Fatal(err)
	}
	if p, g := reopened.Snapshot(); p != latchOff || g != 8 {
		t.Fatalf("reopen %s %d", p, g)
	}
}

func FuzzDecodeLatch(f *testing.F) {
	for _, s := range []string{`{"version":1,"phase":"off","generation":0}`, `null`, `{"version":1,"phase":"killed","generation":7,"future":{"x":[true]}}`} {
		f.Add([]byte(s))
	}
	f.Fuzz(func(t *testing.T, raw []byte) {
		r, err := decodeLatch(raw)
		if err != nil {
			return
		}
		encoded, err := encodeLatch(r)
		if err != nil {
			t.Fatal(err)
		}
		again, err := decodeLatch(encoded)
		if err != nil {
			t.Fatal(err)
		}
		if r.Version != again.Version || r.Phase != again.Phase || r.Generation != again.Generation {
			t.Fatal("projection changed")
		}
	})
}

func TestLatchBounds(t *testing.T) {
	base := `{"version":1,"phase":"off","generation":0,"future":`
	atDepth := base + strings.Repeat(`[`, 31) + `0` + strings.Repeat(`]`, 31) + `}`
	if _, err := decodeLatch([]byte(atDepth)); err != nil {
		t.Fatalf("depth 32 rejected: %v", err)
	}
	good := `{"version":1,"phase":"off","generation":0}`
	atSize := good + strings.Repeat(" ", maxLatchBytes-len(good))
	if _, err := decodeLatch([]byte(atSize)); err != nil {
		t.Fatalf("size limit rejected: %v", err)
	}
	if _, err := decodeLatch([]byte(atSize + " ")); err == nil {
		t.Fatal("oversize accepted")
	}
}
