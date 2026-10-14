package launcher

import (
	"reflect"
	"testing"
)

func TestTerminalLaunchArgsEnforceNonSerializingPolicy(t *testing.T) {
	got := terminalLaunchArgs("name", "/config", "/layout", true)
	want := []string{"--config-dir", "/config", "--new-session-with-layout", "/layout", "--session", "name", "options", "--session-serialization", "false"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("argv = %q", got)
	}
	got = terminalLaunchArgs("name", "/config", "", false)
	if !reflect.DeepEqual(got, []string{"--config-dir", "/config", "attach", "name"}) {
		t.Fatalf("attach argv = %q", got)
	}
}
