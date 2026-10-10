package couchcore

import (
	"strings"
	"testing"
)

// Exhaustive over the admission fact space for both verbs: the expected code is
// stated from the rule's order (not-live, then idle evidence, then the
// checkout, then freshness), never by calling the rule.
func TestDecideLiveRestartExhaustive(t *testing.T) {
	fresh := BinaryFacts{RunningSHA: "old", OnDiskSHA: "new"}
	stale := BinaryFacts{RunningSHA: "same", OnDiskSHA: "same"}
	bools := []bool{false, true}
	checks := 0
	for _, op := range []string{OpRelaunch, opReloadContext} {
		for _, live := range bools {
			for _, session := range bools {
				for _, known := range bools {
					for _, settled := range bools {
						for _, gitKnown := range bools {
							for _, dirty := range bools {
								for _, binary := range []BinaryFacts{fresh, stale} {
									for _, force := range bools {
										f := LiveRestartFacts{Live: live, Session: session, SettledKnown: known, Settled: settled, GitKnown: gitKnown, Dirty: dirty, Binary: binary}
										want := ""
										switch {
										case !live:
											want = LiveRestartNotLive
										case (!session || !known) && !force:
											want = LiveRestartBusyUnknown
										case session && known && !settled:
											want = LiveRestartBusy
										case !gitKnown || dirty:
											want = LiveRestartDirty
										case op == OpRelaunch && binary == stale:
											want = LiveRestartStaleBinary
										}
										got := DecideLiveRestart(op, f, LiveRestartOptions{ForceUnknown: force})
										checks++
										if got.Code != want {
											t.Fatalf("op=%s facts=%+v force=%v: code %q want %q", op, f, force, got.Code, want)
										}
										if got.Code != "" && got.Detail == "" {
											t.Fatalf("refusal without a reason: %+v", got)
										}
										if (!session || !known) && force && live && got.Code != LiveRestartBusyUnknown && !strings.Contains(got.Note, "--force-unknown") {
											t.Fatalf("forced unknown not recorded: %+v", got)
										}
									}
								}
							}
						}
					}
				}
			}
		}
	}
	if checks != 2*2*2*2*2*2*2*2*2 {
		t.Fatalf("enumeration incomplete: %d", checks)
	}
}

func TestDecideBinaryFreshness(t *testing.T) {
	const checkout = "/w/pair"
	for _, tc := range []struct {
		name      string
		b         BinaryFacts
		same      bool
		stale     bool
		detailHas string
		noteHas   string
	}{
		{"tonight: same binary", BinaryFacts{RunningSHA: "a", OnDiskSHA: "a", OnDiskPath: checkout + "/bin/pair", Checkout: checkout}, false, true, "make build in " + checkout, ""},
		{"same binary overridden", BinaryFacts{RunningSHA: "a", OnDiskSHA: "a"}, true, false, "", "--same-binary"},
		{"newer and current", BinaryFacts{RunningSHA: "a", OnDiskSHA: "b", OnDiskRevision: "r2", CheckoutHEAD: "r2"}, false, false, "", ""},
		{"newer but behind its checkout", BinaryFacts{RunningSHA: "a", OnDiskSHA: "b", OnDiskRevision: "r1", CheckoutHEAD: "r2", Checkout: checkout}, false, false, "", "make build in " + checkout},
		{"dirty build", BinaryFacts{RunningSHA: "a", OnDiskSHA: "b", OnDiskModified: true}, false, false, "", "dirty tree"},
		{"slot reports no build", BinaryFacts{OnDiskSHA: "b"}, false, false, "", "unverified"},
		{"binary unreadable", BinaryFacts{RunningSHA: "a"}, false, false, "", "unverified"},
		{"PAIR_DEV rebuilds", BinaryFacts{DevRebuild: true, RunningSHA: "a", OnDiskSHA: "a"}, false, false, "", "PAIR_DEV"},
	} {
		stale, detail, note := DecideBinaryFreshness(tc.b, tc.same)
		if stale != tc.stale || !strings.Contains(detail, tc.detailHas) || !strings.Contains(note, tc.noteHas) {
			t.Errorf("%s: stale=%v detail=%q note=%q", tc.name, stale, detail, note)
		}
	}
}
