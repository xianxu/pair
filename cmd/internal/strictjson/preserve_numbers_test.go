package strictjson

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestPreserveNumbersScannerKeepsLegacyOverflowRefusal(t *testing.T) {
	for _, raw := range []string{`{"future":1e9999}`, `[1e9999,{"n":1e9999}]`} {
		if e := RejectDuplicateKeys([]byte(raw)); e == nil {
			t.Fatal("legacy scanner accepted overflow")
		}
		if e := Decode([]byte(raw), new(any)); e == nil {
			t.Fatal("legacy typed reader accepted overflow")
		}
		if e := RejectDuplicateKeysPreserveNumbers([]byte(raw)); e != nil {
			t.Fatalf("opaque number rejected %s: %v", raw, e)
		}
	}
}
func TestPreserveNumbersScannerRetainsStructuralStrictness(t *testing.T) {
	for _, raw := range []string{`{"n":1e9999,"\u006e":2}`, `{"future":[{"x":1e-9999,"x":0}]}`, `{"n":1e9999} {}`, `{"n":01}`, `{"n":NaN}`, `{"n":1e}`, `{"n":+1}`, `{"n":Infinity}`} {
		if e := RejectDuplicateKeysPreserveNumbers([]byte(raw)); e == nil {
			t.Fatalf("accepted malformed JSON %s", raw)
		}
	}
}
func TestPreserveNumbersObjectHelpersRoundtripRawTokens(t *testing.T) {
	opaque := []byte(strings.ReplaceAll(`{ "nested" : [1e9999,1e-9999,0.123456789012345678901234567890,-0,GIANT] }`, "GIANT", strings.Repeat("9", 400)))
	known := map[string]bool{"version": true}
	raw := append(append([]byte(`{"version":1,"future":`), opaque...), '}')
	if _, _, e := SplitObject(raw, known, nil); e == nil {
		t.Fatal("legacy split accepted overflow")
	}
	owned, unknown, e := SplitObjectPreserveNumbers(raw, known, nil)
	if e != nil {
		t.Fatal(e)
	}
	if !bytes.Equal(unknown["future"], opaque) {
		t.Fatal("split changed raw number spelling")
	}
	if _, e = EncodeObject(owned, unknown, known); e == nil {
		t.Fatal("legacy encoder accepted overflow")
	}
	encoded, e := EncodeObjectPreserveNumbers(owned, unknown, known)
	if e != nil {
		t.Fatal(e)
	}
	_, again, e := SplitObjectPreserveNumbers(encoded, known, nil)
	if e != nil || !bytes.Equal(again["future"], opaque) {
		t.Fatalf("encode changed raw tokens: %s %v", encoded, e)
	}
	unknown["future"] = json.RawMessage(`{"n":1e9999,"\u006e":1}`)
	if _, e = EncodeObjectPreserveNumbers(owned, unknown, known); e == nil {
		t.Fatal("preserving encoder accepted escaped duplicate")
	}
	for _, bad := range []string{`null`, `[]`, `{"version":null,"future":1e9999}`, `{"version":1,"future":{"a":1e9999,"\u0061":2}}`} {
		if _, _, e := SplitObjectPreserveNumbers([]byte(bad), known, nil); e == nil {
			t.Fatalf("preserving split accepted %s", bad)
		}
	}
}
