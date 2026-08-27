package strictjson

import "testing"

func TestDecodeRejectsEveryNonCanonicalEnvelopeClass(t *testing.T) {
	type value struct {
		Name string `json:"name"`
	}
	for name, raw := range map[string]string{
		"duplicate": `{"name":"one","name":"two"}`,
		"unknown":   `{"name":"one","extra":true}`,
		"trailing":  `{"name":"one"} {"name":"two"}`,
	} {
		t.Run(name, func(t *testing.T) {
			var got value
			if err := Decode([]byte(raw), &got); err == nil {
				t.Fatalf("Decode accepted %s JSON", name)
			}
		})
	}
	var got value
	if err := Decode([]byte(`{"name":"one"}`), &got); err != nil || got.Name != "one" {
		t.Fatalf("valid decode = %+v, %v", got, err)
	}
}
