package couchcmd

import "testing"

func TestAutomationCLIContract(t *testing.T) {
	for _, action := range []string{"on", "off", "status"} {
		for _, target := range []string{"", "ops:0"} {
			args := []string{"--automation", action}
			if target != "" {
				args = append(args, target)
			}
			inv, err := ParseCLI(args, nil)
			if err != nil {
				t.Fatalf("%v: %v", args, err)
			}
			if inv.messageOp != "automation" || inv.messageBody != action || inv.ref != target {
				t.Fatalf("%v: unexpected invocation %+v", args, inv)
			}
		}
	}
	for _, args := range [][]string{{"--automation"}, {"--automation", "toggle"}, {"--automation", "open"}, {"--automation", "on", "ops:0", "other"}} {
		if _, err := ParseCLI(args, nil); err == nil {
			t.Fatalf("accepted %v", args)
		}
	}
}
