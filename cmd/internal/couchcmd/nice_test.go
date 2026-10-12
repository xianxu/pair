package couchcmd

import (
	"github.com/xianxu/pair/cmd/internal/couchmessage"
	"strings"
	"testing"
	"time"
)

func TestSlotNiceSettings(t *testing.T) {
	for raw, want := range map[string]int{"": 10, "0": 0, "5": 5, "20": 20} {
		got, err := slotNiceSettings(func(string) string { return raw })
		if err != nil || got != want {
			t.Errorf("COUCH_SLOT_NICE=%q = %d, %v; want %d", raw, got, err, want)
		}
	}
	for _, raw := range []string{"-1", "21", "ten", " 10", "1.5"} {
		if _, err := slotNiceSettings(func(string) string { return raw }); err == nil || !strings.Contains(err.Error(), "COUCH_SLOT_NICE") {
			t.Errorf("COUCH_SLOT_NICE=%q accepted or unnamed: %v", raw, err)
		}
	}
}

// The renicer's interval (#502): a duration of at least a second.
func TestReniceInterval(t *testing.T) {
	for raw, want := range map[string]time.Duration{"": defaultReniceInterval, "1s": time.Second, "30s": 30 * time.Second} {
		got, err := reniceInterval(func(string) string { return raw })
		if err != nil || got != want {
			t.Errorf("COUCH_RENICE_INTERVAL=%q = %v, %v; want %v", raw, got, err, want)
		}
	}
	for _, raw := range []string{"0", "500ms", "-1s", "2h", "soon"} {
		if _, err := reniceInterval(func(string) string { return raw }); err == nil || !strings.Contains(err.Error(), "COUCH_RENICE_INTERVAL") {
			t.Errorf("COUCH_RENICE_INTERVAL=%q accepted or unnamed: %v", raw, err)
		}
	}
}

func TestNiceColumn(t *testing.T) {
	for _, c := range []struct {
		f    *couchmessage.NiceFacts
		want string
	}{
		{nil, ""},
		{&couchmessage.NiceFacts{}, "\tnice=?"},
		{&couchmessage.NiceFacts{Confirmed: true, Lowered: 1, AtLevel: 2}, "\tnice=0 lowered=3"},
		{&couchmessage.NiceFacts{Confirmed: true, AgentNice: 10, Denied: 1}, "\tnice=10 lowered=0 denied=1"},
	} {
		if got := niceColumn(c.f); got != c.want {
			t.Errorf("niceColumn(%+v) = %q, want %q", c.f, got, c.want)
		}
	}
}
