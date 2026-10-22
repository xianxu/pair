package editorendpoint

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestStateCodecOpaqueAndStrict(t *testing.T) {
	s, _ := reg(t, EndpointState{}, req(incA, docA, "1", "desktop"))
	s.Unknown = map[string]json.RawMessage{"future": []byte(`{ "x" : [1, 2] }`)}
	s.Records[0].Unknown = map[string]json.RawMessage{"future": []byte(`[ 1, 2 ]`)}
	s.Records[0].Binding.Unknown = map[string]json.RawMessage{"future": []byte(`{ "k": true }`)}
	raw, e := EncodeEndpointState(s)
	if e != nil {
		t.Fatal(e)
	}
	decoded, e := DecodeEndpointState(raw)
	if e != nil {
		t.Fatal(e)
	}
	again, e := EncodeEndpointState(decoded)
	if e != nil || !bytes.Equal(raw, again) {
		t.Fatalf("opaque rewrite changed: %s %v", again, e)
	}
	for _, bad := range [][]byte{[]byte(`{}`), []byte(`null`), bytes.Replace(raw, []byte(`"Revision":1`), []byte(`"Revision":1,"Revision":2`), 1), bytes.Replace(raw, []byte(`"PID":42`), []byte(`"PID":null`), 1), bytes.Replace(raw, []byte(`"Status":"current"`), []byte(`"Status":"future"`), 1)} {
		if _, e := DecodeEndpointState(bad); e == nil {
			t.Fatalf("accepted invalid state: %s", bad)
		}
	}
}
func FuzzStateCodec(f *testing.F) {
	f.Add([]byte(`{"Version":1,"Namespace":"/project","Revision":0,"Records":[]}`))
	f.Add([]byte(`{"Version":1,"Version":1}`))
	f.Add([]byte(`{"Version":1,"Namespace":"/project","Revision":0,"Records":[],"future":[1e9999,1e-9999,-0,0.123456789012345678901234567890]}`))
	f.Fuzz(func(t *testing.T, raw []byte) {
		s, e := DecodeEndpointState(raw)
		if e != nil {
			return
		}
		encoded, e := EncodeEndpointState(s)
		if e != nil {
			t.Fatal(e)
		}
		next, e := DecodeEndpointState(encoded)
		if e != nil {
			t.Fatal(e)
		}
		again, e := EncodeEndpointState(next)
		if e != nil || !bytes.Equal(encoded, again) {
			t.Fatal("roundtrip unstable")
		}
	})
}
func TestCurrentOpaqueSurvivesActualRewrite(t *testing.T) {
	s, p := reg(t, EndpointState{}, req(incA, docA, "1", "desktop"))
	_ = p
	s.Records[0].Binding.Unknown = map[string]json.RawMessage{"opaque": []byte(`{ "future" : 3 }`)}
	s, _ = reg(t, s, req(incA, docA, "2", "headless"))
	raw, e := EncodeEndpointState(s)
	if e != nil {
		t.Fatal(e)
	}
	n, e := DecodeEndpointState(raw)
	if e != nil {
		t.Fatal(e)
	}
	if !bytes.Equal(n.Records[0].Current.Unknown["opaque"], []byte(`{ "future" : 3 }`)) {
		t.Fatal("current lost opaque field")
	}
	again, e := EncodeEndpointState(n)
	if e != nil || !bytes.Equal(raw, again) {
		t.Fatal("current rewrite unstable")
	}
}
func TestCodecByteLimit(t *testing.T) {
	if _, e := DecodeEndpointState(bytes.Repeat([]byte(" "), MaxStateBytes+1)); e == nil {
		t.Fatal("oversize decode accepted")
	}
	s, _ := reg(t, EndpointState{}, req(incA, docA, "1", "desktop"))
	s.Unknown = map[string]json.RawMessage{"big": append(append([]byte{'"'}, bytes.Repeat([]byte{'x'}, MaxStateBytes)...), '"')}
	if _, e := EncodeEndpointState(s); e == nil {
		t.Fatal("oversize encode accepted")
	}
}
func TestCodecRejectsImpossibleRetainedCurrent(t *testing.T) {
	s, _ := reg(t, EndpointState{}, req(incA, docA, "1", "desktop"))
	s, _ = reg(t, s, req(incA, docA, "2", "headless"))
	s.Records[0].Binding.Mode = "desktop"
	if _, e := EncodeEndpointState(s); e == nil {
		t.Fatal("accepted retained current under fresh desktop high-water")
	}
}

// Future numeric fields are opaque JSON tokens, not float64 values. Their
// spelling and interior whitespace must survive every enclosing owned object.
func endpointOpaqueNumberState(t *testing.T) EndpointState {
	t.Helper()
	s, _ := reg(t, EndpointState{}, req(incA, docA, "1", "desktop"))
	s, _ = reg(t, s, req(incA, docA, "2", "headless"))
	opaque := json.RawMessage(strings.ReplaceAll(`{ "nested" : [1e9999, 1e-9999, 0.123456789012345678901234567890123456789, -0, GIANT, {"n":1e9999}] }`, "GIANT", strings.Repeat("9", 400)))
	s.Unknown = map[string]json.RawMessage{"future": append(json.RawMessage(nil), opaque...)}
	s.Records[0].Unknown = map[string]json.RawMessage{"future": append(json.RawMessage(nil), opaque...)}
	s.Records[0].Binding.Unknown = map[string]json.RawMessage{"future": append(json.RawMessage(nil), opaque...)}
	s.Records[0].Current.Unknown = map[string]json.RawMessage{"future": append(json.RawMessage(nil), opaque...)}
	return s
}
func TestEndpointOpaqueNumbersAllLevelsRewrite(t *testing.T) {
	s := endpointOpaqueNumberState(t)
	raw, e := EncodeEndpointState(s)
	if e != nil {
		t.Fatal(e)
	}
	for cycle := 0; cycle < 3; cycle++ {
		n, e := DecodeEndpointState(raw)
		if e != nil {
			t.Fatal(e)
		}
		for level, unknown := range []map[string]json.RawMessage{n.Unknown, n.Records[0].Unknown, n.Records[0].Binding.Unknown, n.Records[0].Current.Unknown} {
			if !bytes.Equal(unknown["future"], s.Unknown["future"]) {
				t.Fatalf("level%d changed opaque number bytes: %s", level, unknown["future"])
			}
		}
		again, e := EncodeEndpointState(n)
		if e != nil || !bytes.Equal(raw, again) {
			t.Fatalf("rewrite changed numbers: %v", e)
		}
		raw = again
	}
}
func TestEndpointOpaqueNumbersDoNotRelaxKnownTypesOrDuplicates(t *testing.T) {
	s := endpointOpaqueNumberState(t)
	raw, e := EncodeEndpointState(s)
	if e != nil {
		t.Fatal(e)
	}
	for _, bad := range [][]byte{
		bytes.Replace(raw, []byte(`"Revision":2`), []byte(`"Revision":18446744073709551616`), 1),
		bytes.Replace(raw, []byte(`"PID":42`), []byte(`"PID":1e9999`), 1),
		bytes.Replace(raw, []byte(`"Buffer":1`), []byte(`"Buffer":1.5`), 1),
		bytes.Replace(raw, []byte(`"Version":1`), []byte(`"Version":"1"`), 1),
		bytes.Replace(raw, []byte(`"Mode":"headless"`), []byte(`"Mode":null`), 1),
		bytes.Replace(raw, []byte(`{"n":1e9999}`), []byte(`{"n":1e9999,"\u006e":1}`), 1),
		bytes.Replace(raw, []byte(`"PID":42`), []byte(`"PID":42,"\u0050ID":42`), 1),
	} {
		if bytes.Equal(raw, bad) {
			t.Fatal("fixture replacement did not happen")
		}
		if _, e := DecodeEndpointState(bad); e == nil {
			t.Fatalf("accepted malformed authority %s", bad)
		}
	}
}
