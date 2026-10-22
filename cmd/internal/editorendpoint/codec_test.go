package editorendpoint

import (
	"bytes"
	"encoding/json"
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
