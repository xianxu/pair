package launcher

import (
	"reflect"
	"testing"
)

func TestKillSessionExecsJoinedHelperForNativeOnly(t *testing.T) {
	for _, tc := range []struct {
		override string
		want     []string
	}{
		{"", []string{"/current/pair", "internal", "zellij-command", "--", "kill-session", "demo"}},
		{"zellij kill-session", []string{"/current/pair", "internal", "zellij-command", "--", "kill-session", "demo"}},
		{"/native/zellij kill-session", []string{"/current/pair", "internal", "zellij-command", "--", "/native/zellij", "kill-session", "demo"}},
		{"/private/kill --record", []string{"/private/kill", "--record", "demo"}},
	} {
		got, err := killSessionExecArgv("/current/pair", tc.override, "demo")
		if err != nil || !reflect.DeepEqual(got, tc.want) {
			t.Fatalf("override %q: argv=%q err=%v want=%q", tc.override, got, err, tc.want)
		}
	}
	if _, err := killSessionExecArgv("/current/pair", "   ", "demo"); err == nil {
		t.Fatal("empty override must refuse rather than panic")
	}
}
