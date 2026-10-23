package testcouchbridge

import (
	"testing"
	"time"
)

func TestOptionsAreLexicalAndBudgetsFixed(t *testing.T) {
	if StartupBudget != 300*time.Second || LifetimeBudget != 600*time.Second || StopBudget != 15*time.Second || ReadinessLimit != 64<<10 || DiagnosticLimit != 64<<10 {
		t.Fatal("contract bounds changed")
	}
	for _, tt := range []struct {
		name    string
		options Options
		valid   bool
	}{
		{"valid", Options{ScratchParent: "/nonexistent/scratch"}, true},
		{"candidate", Options{ScratchParent: "/scratch", Couch: "/nonexistent/couch"}, true},
		{"relative", Options{ScratchParent: "scratch"}, false},
		{"unclean", Options{ScratchParent: "/scratch/../other"}, false},
		{"relative candidate", Options{ScratchParent: "/scratch", Couch: "bin/couch"}, false},
		{"nul", Options{ScratchParent: "/scratch\x00"}, false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if err := ValidateBridgeOptions(tt.options); (err == nil) != tt.valid {
				t.Fatalf("valid=%v: %v", tt.valid, err)
			}
		})
	}
}

func FuzzBridgeOptions(f *testing.F) {
	f.Add("/scratch", "/bin/couch")
	f.Add("relative", "")
	f.Fuzz(func(t *testing.T, scratch, couch string) {
		if len(scratch)+len(couch) > ReadinessLimit {
			t.Skip()
		}
		_ = ValidateBridgeOptions(Options{ScratchParent: scratch, Couch: couch})
	})
}
