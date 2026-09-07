package couchcore

import "testing"

func TestParseLayoutAcceptsKnownValues(t *testing.T) {
	for _, tc := range []struct {
		in   string
		want Layout
	}{{"layout2", Layout2}, {"layout3", Layout3}} {
		got, err := ParseLayout(tc.in)
		if err != nil || got != tc.want {
			t.Fatalf("ParseLayout(%q) = %v, %v; want %v, nil", tc.in, got, err, tc.want)
		}
	}
}

// An absent field is every record written before #198, and those are layout2
// with certainty -- couch pinned layout2 from 2026-08-22 until then.
func TestParseLayoutTreatsEmptyAsLayout2(t *testing.T) {
	got, err := ParseLayout("")
	if err != nil || got != Layout2 {
		t.Fatalf("ParseLayout(%q) = %v, %v; want Layout2, nil", "", got, err)
	}
}

// ARCH-SECURE: a hand-edited or newer-version value must not coerce to a
// plausible default the guard would then trust.
func TestParseLayoutRejectsUnknown(t *testing.T) {
	if _, err := ParseLayout("layout9"); err == nil {
		t.Fatal("ParseLayout(\"layout9\") = nil error; want a refusal")
	}
}

// NormalizeLayout is the total variant for the row projection, which has no
// error return.
func TestNormalizeLayoutIsTotal(t *testing.T) {
	if got := NormalizeLayout(""); got != Layout2 {
		t.Fatalf("NormalizeLayout(\"\") = %v; want Layout2", got)
	}
	if got := NormalizeLayout("layout3"); got != Layout3 {
		t.Fatalf("NormalizeLayout(\"layout3\") = %v; want Layout3", got)
	}
	if got := NormalizeLayout("layout9"); got != LayoutUnknown {
		t.Fatalf("NormalizeLayout(\"layout9\") = %v; want LayoutUnknown", got)
	}
}

func TestLayoutFlagIsTheOnlyFormatter(t *testing.T) {
	if Layout2.Flag() != "--layout2" || Layout3.Flag() != "--layout3" {
		t.Fatalf("flags = %q, %q; want --layout2, --layout3", Layout2.Flag(), Layout3.Flag())
	}
}
