package couchcore

import (
	"testing"

	"github.com/xianxu/pair/cmd/internal/launcher"
)

func TestParseLayoutAcceptsKnownValues(t *testing.T) {
	t.Parallel()

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

func TestParseLayoutTreatsEmptyAsLayout2(t *testing.T) {
	t.Parallel()

	got, err := ParseLayout("")
	if err != nil || got != Layout2 {
		t.Fatalf("ParseLayout(%q) = %v, %v; want Layout2, nil", "", got, err)
	}
}

func TestParseLayoutRejectsUnknown(t *testing.T) {
	t.Parallel()

	if _, err := ParseLayout("layout9"); err == nil {
		t.Fatal("ParseLayout(\"layout9\") = nil error; want a refusal")
	}
}

func TestNormalizeLayoutIsTotal(t *testing.T) {
	t.Parallel()

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
	t.Parallel()

	if Layout2.Flag() != "--layout2" || Layout3.Flag() != "--layout3" {
		t.Fatalf("flags = %q, %q; want --layout2, --layout3", Layout2.Flag(), Layout3.Flag())
	}
}

func TestCouchLayoutFlagsAreWhatPairParses(t *testing.T) {
	t.Parallel()

	for _, layout := range []Layout{Layout2, Layout3} {
		args, err := launcher.ParseArgs([]string{"resume", "couch-0102030405060708", layout.Flag()})
		if err != nil {
			t.Fatalf("pair refused couch's own argv for %s: %v", layout, err)
		}
		if !args.Layout.Explicit || args.Layout.Mode != layout {
			t.Fatalf("pair parsed %q as %+v; want an explicit %s", layout.Flag(), args.Layout, layout)
		}
	}
}

func TestFlagIsTotalOverUnsetAndUnknownLayouts(t *testing.T) {
	t.Parallel()

	for _, layout := range []Layout{"", LayoutUnknown, Layout("garbage")} {
		if got := layout.Flag(); got != "--layout2" {
			t.Fatalf("Layout(%q).Flag() = %q; want the --layout2 default", layout, got)
		}
	}
}
