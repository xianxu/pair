package strictjson

import "testing"

func TestValidateUnicodeBeforeReplacement(t *testing.T) {
	for _, raw := range []string{`"\ud800"`, `"\udfff"`, `"\ud800\ud800"`, `["\ud800"]`, `{"\ud800":"key"}`, `"` + string([]byte{0xff}) + `"`, `null trailing`} {
		if ValidateUnicode([]byte(raw)) == nil {
			t.Errorf("accepted malformed Unicode %q", raw)
		}
	}
	for _, raw := range []string{`null`, `[]`, `"�"`, `"\ufffd"`, `"\ud83d\ude00"`, `"literal \\ud800"`, `{"valid":"漢字 😀"}`} {
		if err := ValidateUnicode([]byte(raw)); err != nil {
			t.Errorf("refused %s: %v", raw, err)
		}
	}
}
