package strictjson

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"testing"
)

func TestPreserveNumberDuplicateValidationIsAdditive(t *testing.T) {
	for _, raw := range []string{`1e999`, `-1e999`, `{"future":[1e999,-0,1e-999,0.12345678901234567890123456789]}`} {
		if err := RejectDuplicateKeysPreserveNumbers([]byte(raw)); err != nil {
			t.Fatalf("valid raw number refused: %v", err)
		}
	}
	if err := RejectDuplicateKeys([]byte(`1e999`)); err == nil {
		t.Fatal("legacy overflow acceptance changed")
	}
	for _, raw := range []string{`{"future":{"x":1e999,"\u0078":2}}`, `{"x":1} {}`, `NaN`, `Infinity`, `{"x":01}`, `{"x":1e}`} {
		if err := RejectDuplicateKeysPreserveNumbers([]byte(raw)); err == nil {
			t.Fatalf("ambiguous or invalid JSON accepted: %s", raw)
		}
	}
}
func FuzzPreserveNumberDuplicateValidation(f *testing.F) {
	for _, raw := range []string{`1e999`, `{"x":[1e999]}`, `{"x":1,"\u0078":2}`, `{"x":null}`, `[]`, `{} {}`} {
		f.Add([]byte(raw))
	}
	f.Fuzz(func(t *testing.T, raw []byte) {
		d := json.NewDecoder(bytes.NewReader(raw))
		d.UseNumber()
		reference := legacyScanValue(d)
		if reference == nil {
			var extra any
			if err := d.Decode(&extra); !errors.Is(err, io.EOF) {
				reference = errors.New("trailing data")
			}
		}
		got := RejectDuplicateKeysPreserveNumbers(raw)
		if (reference == nil) != (got == nil) {
			t.Fatalf("preserving scanner differs from UseNumber oracle: reference=%v got=%v raw=%q", reference, got, raw)
		}
	})
}
