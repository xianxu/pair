package testwait

import (
	"errors"
	"strings"
	"testing"
)

func TestHostStateNamesMeasuredLoadOrUnavailableReason(t *testing.T) {
	for _, tc := range []struct {
		load string
		err  error
		want string
	}{
		{"{ 1.25 2.50 3.75 }\n", nil, "loadavg={ 1.25 2.50 3.75 }"},
		{"", errors.New("probe canceled"), "loadavg=unavailable: probe canceled"},
	} {
		calls := 0
		got := hostState(func() (string, error) { calls++; return tc.load, tc.err })
		if calls != 1 || !strings.Contains(got, tc.want) || !strings.Contains(got, "cpus=") || !strings.Contains(got, "os=") {
			t.Fatalf("host state=%q calls=%d want %q", got, calls, tc.want)
		}
	}
}
