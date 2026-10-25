package couchtty

import (
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/couchcore"
)

type ordinaryResultWarning string

func (r ordinaryResultWarning) Warning() string { return string(r) }

func TestLifecycleSuccessResultAttentionUsesTypedWarningFacts(t *testing.T) {
	target, archived := menuAddress("result-origin"), menuAddress("archived-origin")
	start := couchcore.StartResult{Record: couchcore.ActorRecord{Thread: target, AttachmentNotice: "BORROWED ATTACH: no unique retained client helper is identified; retained lifecycle state was preserved."}}
	publicationFailure := couchcore.StartResult{Record: start.Record, Configuration: &couchcore.AcceptedConfiguration{}}
	// An empty accepted profile takes the real failure path without storage IO.
	(&couchcore.Couch{}).PublishAdoptedConfiguration(publicationFailure)
	if !strings.Contains(publicationFailure.Warning(), "startup parameters were not remembered") {
		t.Fatal("fixture did not publish a configuration failure")
	}
	for _, tc := range []struct {
		name   string
		value  interface{ Warning() string }
		want   bool
		thread couchcore.ThreadAddress
	}{
		{"borrowed-attachment", start, false, target},
		{"configuration-publication-failure", publicationFailure, true, target},
		{"archive-left-running", couchcore.ArchiveResult{Record: couchcore.ThreadRecord{Address: archived}, SessionNotStopped: true}, true, archived},
		{"reboot-left-running", couchcore.RebootResult{Archived: archived, ArchiveOnly: true, SessionNotStopped: true}, true, archived},
		{"reboot-borrowed-attachment", couchcore.RebootResult{Start: start, Notice: "started fresh"}, false, target},
		{"recovery-borrowed-attachment", couchcore.RecoveryResumeResult{Start: start, Notice: "Continue"}, false, target},
		{"switch-borrowed-attachment", couchcore.SwitchAgentResult{Outcome: couchcore.SwitchStarted, Record: start.Record}, false, target},
		{"switch-cleanup", couchcore.SwitchAgentResult{Outcome: couchcore.SwitchStarted, Record: couchcore.ActorRecord{Thread: target}, Notice: "source cleanup refused; started fresh", CleanupWarning: "source cleanup refused"}, true, target},
		{"reboot-adoption-and-archive", couchcore.RebootResult{Start: start, Archived: archived, SessionNotStopped: true}, true, archived},
		{"bench-start-adoption-and-archive", couchcore.BenchStartResult{RebootResult: couchcore.RebootResult{Start: start, Archived: archived, SessionNotStopped: true}}, true, archived},
		{"bench-start-borrowed-attachment", couchcore.BenchStartResult{RebootResult: couchcore.RebootResult{Start: start, Notice: "started fresh"}}, false, target},
		{"park-retired-cleanup", couchcore.ParkRecoveryResult{Thread: couchcore.ThreadRecord{Address: target}, Diagnostic: "scrollback preservation incomplete"}, true, target},
		{"park-next-command", couchcore.ParkRecoveryResult{Thread: couchcore.ThreadRecord{Address: target}, NextCommand: "resume pair:1", OperatorOutcome: "bookkeeping resolved"}, false, target},
		{"ordinary-recovery", couchcore.RecoveryResumeResult{Notice: "resumed conversation"}, false, target},
		{"ordinary-switch", couchcore.SwitchAgentResult{Outcome: couchcore.SwitchStarted, Notice: "started fresh after missing transcript; no cleanup warning"}, false, target},
		{"ordinary-reboot", couchcore.RebootResult{Notice: "started fresh"}, false, target},
		{"untyped-error-wording", ordinaryResultWarning("cleanup failure; attention required"), false, target},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := newRemoteRig(t)
			r.c.menuReady = false
			completed := operationRequest{origin: MenuOperationOrigin{Operation: "resume", Address: target}}.complete(tc.value, nil)
			r.c.finishOperation(completed)
			r.c.finishOperation(completed) // UI completion has one owner too.
			items := warningSourceItems(r.c)
			if !tc.want {
				if len(items) != 0 {
					t.Fatalf("ordinary result raised attention: %+v", items)
				}
			} else if len(items) != 1 || items[0].Thread != tc.thread {
				t.Fatalf("typed completion warning lost source or duplicated: %+v", items)
			}
			messages := r.c.feed.Messages()
			if len(messages) != 1 || messages[0].Body != tc.value.Warning() {
				t.Fatalf("generic warning feed changed or duplicated: %+v, want %q", messages, tc.value.Warning())
			}
		})
	}
}
