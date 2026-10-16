package nativeid

import (
	"strings"
	"testing"
)

func TestValidate(t *testing.T) {
	for _, tc := range []struct {
		id string
		ok bool
	}{
		{"019d1111-1111-7111-8111-111111111111", true},
		{"named-session_1.2", true},
		{strings.Repeat("x", MaxLen), true},
		{"", false},
		{strings.Repeat("x", MaxLen+1), false},
		{"has space", false},
		{"tab\there", false},
		{"zero​width", false},
		{"ctl\x01", false},
		{"a/b", false},
		{`a\b`, false},
		{".", false},
		{"..", false},
		{"bad\xffutf8", false},
	} {
		if err := Validate(tc.id); (err == nil) != tc.ok {
			t.Errorf("%q: err=%v, want ok=%v", tc.id, err, tc.ok)
		}
	}
}
