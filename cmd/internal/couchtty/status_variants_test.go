package couchtty

import (
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/broadcast"
	"github.com/xianxu/pair/cmd/internal/textwidth"
	"github.com/xianxu/pair/cmd/internal/webcast"
	"github.com/xianxu/pair/cmd/internal/webcast/capability"
)

// Exercise the actual producer outputs with every finite chrome choice. Labels,
// user names, IDs and error details are deliberately absent, not truncated here.
func TestBuiltInChromeStatusVariants(t *testing.T) {
	check := func(label, message string) {
		t.Helper()
		if width := textwidth.Width(message); width > ChromeStatusCells {
			t.Errorf("%s: %d cells: %q", label, width, message)
		}
	}
	for _, kind := range []webcast.CapabilityKind{webcast.CapabilityAnnotation, webcast.CapabilityDrive} {
		for _, outcome := range []capability.CleanupOutcome{capability.Running, capability.Retryable, capability.Broken} {
			caps := webcast.Capabilities{}
			snapshot := capability.Snapshot{CapabilityState: capability.CapabilityState{Phase: capability.Disabling, Generation: 1, Attempt: 1, Outcome: outcome}}
			if kind == webcast.CapabilityAnnotation {
				caps.Annotation = snapshot
			} else {
				caps.Drive = snapshot
			}
			check(string(kind), capabilityNotice(caps))
		}
	}
	for _, link := range []webcastLink{webcastViewOnlyLink, webcastPointerLink, webcastDriveLink} {
		check(link.name+" copied", webcastLinkStatus(link, true))
		check(link.name+" not-ready", webcastLinkStatus(link, false))
		if link.missing != nil {
			check(link.name+" unavailable", link.missing(nil))
		}
	}
	for _, kind := range []castKind{castScreen, castWeb} {
		for _, link := range []string{broadcast.LinkView, broadcast.LinkPointer, broadcast.LinkDrive} {
			check(kind.name()+" rotation", castRotationNotice(kind, link+" code"))
			for _, capped := range []bool{false, true} {
				check(kind.name()+" guess", castGuessNotice(kind, link, capped))
			}
		}
		check(kind.name()+" all-code rotation", castRotationNotice(kind, "codes"))
	}
}

func TestBuiltInChromeStatusVariantsKeepDynamicDiagnostics(t *testing.T) {
	detail := strings.Repeat("diagnostic ", 30)
	for _, kind := range []webcast.CapabilityKind{webcast.CapabilityAnnotation, webcast.CapabilityDrive} {
		for _, outcome := range []capability.CleanupOutcome{capability.Retryable, capability.Broken} {
			caps := webcast.Capabilities{}
			snapshot := capability.Snapshot{CapabilityState: capability.CapabilityState{Phase: capability.Disabling, Generation: 1, Attempt: 1, Outcome: outcome}, Problem: detail}
			if kind == webcast.CapabilityAnnotation {
				caps.Annotation = snapshot
			} else {
				caps.Drive = snapshot
			}
			if got := capabilityNotice(caps); !strings.HasSuffix(got, detail) {
				t.Fatalf("diagnostic was modified: %q", got)
			}
		}
	}
}

func TestBuiltInChromeStatusMissingLinkCallback(t *testing.T) {
	r := newWebcastRig(t, nil)
	r.open(t)
	r.con.mu.Lock()
	cast := r.con.wcast.cast
	r.con.mu.Unlock()
	if cast.DriveOff() == "" {
		t.Fatal("fixture must exercise unrecorded drive reason")
	}
	for _, link := range []webcastLink{webcastPointerLink, webcastDriveLink} {
		message := link.missing(cast)
		if width := textwidth.Width(message); width > ChromeStatusCells {
			t.Errorf("%s unavailable: %d cells: %q", link.name, width, message)
		}
	}
}
