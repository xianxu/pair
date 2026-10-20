package couchcmd

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/couchmessage"
	"github.com/xianxu/pair/cmd/internal/sessioninventory"
	"strings"
	"testing"
	"time"
)

func TestResumeCandidatesCLI(t *testing.T) {
	for _, args := range [][]string{{"--resume-candidates", "pair:1", "--json"}, {"--resume", "pair:1", "--candidate", "v1.key"}} {
		if _, err := ParseCLI(args, couchcore.Operations()); err != nil {
			t.Fatalf("%v: %v", args, err)
		}
	}
	for _, args := range [][]string{{"--resume-candidates", "pair"}, {"--resume", "pair:1", "--candidate"}, {"--reboot", "pair:1", "--confirm", "--candidate", "key"}, {"--resume", "pair:1", "--candidate", "one", "--candidate", "two"}} {
		if _, err := ParseCLI(args, couchcore.Operations()); err == nil {
			t.Fatalf("accepted %v", args)
		}
	}
}

func TestResumeCandidatesCLIUsesReadOnlyBoundedRequest(t *testing.T) {
	var out, diagnostic bytes.Buffer
	calls := 0
	code := runMessageCLIWithCall(cliInvocation{kind: cliMessage, messageOp: "resume-candidates", ref: "pair:1", jsonOutput: true}, messageRuntime(), &out, &diagnostic, func(ctx context.Context, _ string, request any, response any) error {
		calls++
		r := request.(couchmessage.Request)
		if r.Op != "resume-candidates" || r.Target != "pair:1" || r.ID != "" || r.CandidateKey != "" {
			t.Fatalf("unexpected preview request %+v", r)
		}
		deadline, ok := ctx.Deadline()
		if !ok || time.Until(deadline) > 35*time.Second || time.Until(deadline) <= 2*time.Second {
			t.Fatalf("bad preview deadline %v", deadline)
		}
		*response.(*couchmessage.Response) = couchmessage.Response{Code: "ok", RecoveryPreview: &couchcore.RecoveryPreview{Candidates: []couchcore.RecoveryChoice{{Key: "opaque", Label: "best"}}}}
		return nil
	})
	if code != 0 || calls != 1 || !strings.Contains(out.String(), "opaque") {
		t.Fatalf("preview code%d calls%d out%s diagnostic%s", code, calls, &out, &diagnostic)
	}
}

func TestResumeCandidatesCLILegacyFreshShowsUnknownActivityWithoutNativeScore(t *testing.T) {
	var out, diagnostic bytes.Buffer
	code := runMessageCLIWithCall(cliInvocation{kind: cliMessage, messageOp: "resume-candidates", ref: "pair:1"}, messageRuntime(), &out, &diagnostic, func(_ context.Context, _ string, _ any, response any) error {
		*response.(*couchmessage.Response) = couchmessage.Response{Code: "ok", RecoveryPreview: &couchcore.RecoveryPreview{Fresh: &couchcore.RecoveryChoice{Key: "fresh", Label: "start fresh from previous screen", Agent: "codex"}}}
		return nil
	})
	rendered := out.String()
	if code != 0 || !strings.Contains(rendered, "unknown") || strings.Contains(rendered, "0001") || strings.Contains(rendered, "0%") || strings.Contains(rendered, "matched") {
		t.Fatalf("invented fresh evidence: code=%d output=%s diagnostic=%s", code, rendered, &diagnostic)
	}
}

func TestAdvertisedFreshCommandPreservesSelectionThroughCLIRequest(t *testing.T) {
	for _, tc := range []struct {
		name, key, operation string
		explicitCommand      string
		confirmed            bool
	}{
		{name: "keyed Fresh", key: "v1.fresh-opaque_123", operation: "resume"},
		{name: "keyless Fresh", operation: "reboot", confirmed: true},
		{name: "explicit reboot", operation: "reboot", confirmed: true, explicitCommand: "couch --reboot pair:1 --confirm"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var out, diagnostic bytes.Buffer
			command := tc.explicitCommand
			if command == "" {
				inv, err := ParseCLI([]string{"--resume-candidates", "pair:1"}, couchcore.Operations())
				if err != nil {
					t.Fatal(err)
				}
				code := runMessageCLIWithCall(inv, messageRuntime(), &out, &diagnostic, func(_ context.Context, _ string, request any, response any) error {
					req := request.(couchmessage.Request)
					if req.Op != "resume-candidates" || req.Target != "pair:1" || req.CandidateKey != "" {
						t.Fatalf("preview request mutated selection: %+v", req)
					}
					*response.(*couchmessage.Response) = couchmessage.Response{Code: "ok", RecoveryPreview: &couchcore.RecoveryPreview{
						Candidates: []couchcore.RecoveryChoice{{Key: "different-native-key", NativeID: "different-native", Label: "Resume from disk"}},
						Fresh:      &couchcore.RecoveryChoice{Key: tc.key, Label: "Start fresh"},
					}}
					return nil
				})
				if code != 0 {
					t.Fatalf("preview failed: %d %s", code, &diagnostic)
				}
				// Execute the command printed for Fresh, rather than constructing
				// its invocation independently of the advertised surface.
				for _, line := range strings.Split(out.String(), "\n") {
					if line = strings.TrimSpace(line); strings.HasPrefix(line, "couch --") {
						command = line
					}
				}
			}
			argv := strings.Fields(command)
			if len(argv) < 2 || argv[0] != "couch" {
				t.Fatalf("Fresh has no executable command: %q", command)
			}
			inv, err := ParseCLI(argv[1:], couchcore.Operations())
			if err != nil {
				t.Fatalf("printed command cannot parse: %q: %v", command, err)
			}
			out.Reset()
			calls := 0
			code := runMessageCLIWithCall(inv, messageRuntime(), &out, &diagnostic, func(_ context.Context, _ string, request any, response any) error {
				calls++
				req := request.(couchmessage.Request)
				if req.Op != tc.operation || req.Target != "pair:1" || req.CandidateKey != tc.key || req.Confirmed != tc.confirmed || req.ID == "" || req.RecoveryConfirmation != "" {
					t.Fatalf("printed Fresh selection changed in transport: %q -> %+v", command, req)
				}
				*response.(*couchmessage.Response) = couchmessage.Response{Code: "accepted", Operation: &couchmessage.OperationReceipt{ID: req.ID, Op: req.Op, Target: req.Target, Status: couchmessage.ReceiptSucceeded}}
				return nil
			})
			if code != 0 || calls != 1 {
				t.Fatalf("printed Fresh command failed: code=%d calls=%d output=%s diagnostic=%s", code, calls, &out, &diagnostic)
			}
		})
	}
}

func TestResumeCandidatesCLINativeOnlyUnconfirmedEvidence(t *testing.T) {
	for _, jsonOutput := range []bool{false, true} {
		t.Run(fmt.Sprint(jsonOutput), func(t *testing.T) {
			var out, diagnostic bytes.Buffer
			activity := time.Date(2026, 10, 19, 10, 12, 0, 0, time.UTC)
			code := runMessageCLIWithCall(cliInvocation{kind: cliMessage, messageOp: "resume-candidates", ref: "pair:1", jsonOutput: jsonOutput}, messageRuntime(), &out, &diagnostic, func(_ context.Context, _ string, _ any, response any) error {
				*response.(*couchmessage.Response) = couchmessage.Response{Code: "ok", RecoveryPreview: &couchcore.RecoveryPreview{Candidates: []couchcore.RecoveryChoice{{SourceKind: sessioninventory.RecoverySourceNative, Key: "v2.exact", Label: "Resume native transcript", Agent: "codex", NativeID: "native-id", CWD: "/physical/repo\x1b]0;hostile\x07", LastActivity: activity}}, Reasons: []string{"Screen unavailable: capture missing"}, Fresh: &couchcore.RecoveryChoice{Label: "Start fresh"}}}
				return nil
			})
			rendered := out.String()
			if code != 0 {
				t.Fatalf("code=%d diagnostic=%s", code, &diagnostic)
			}
			if jsonOutput {
				for _, name := range []string{"tail_match", "matched", "total"} { // Fresh keeps the v1 representation; inspect native candidate only.
					var preview struct {
						Candidates []map[string]json.RawMessage `json:"candidates"`
					}
					if err := json.Unmarshal(out.Bytes(), &preview); err != nil {
						t.Fatal(err)
					}
					if _, present := preview.Candidates[0][name]; present {
						t.Fatalf("native JSON fabricated %s: %s", name, rendered)
					}
				}
				return
			}
			for _, want := range []string{"unconfirmed", "physical cwd /physical/repo", "native activity " + activity.Local().Format("2006-01-02 15:04 MST"), "native-id", "Screen unavailable: capture missing", "--candidate v2.exact"} {
				if !strings.Contains(rendered, want) {
					t.Fatalf("missing %q: %s", want, rendered)
				}
			}
			for _, unwanted := range []string{"best ·", "tail ", "matched ", "0%", "\x1b", "\x07"} {
				if strings.Contains(rendered, unwanted) {
					t.Fatalf("misleading/unsafe %q: %s", unwanted, rendered)
				}
			}
		})
	}
}

func TestResumeCandidatesCLINativeOnlyIncompleteHasNoResumeCommand(t *testing.T) {
	var out, diagnostic bytes.Buffer
	code := runMessageCLIWithCall(cliInvocation{kind: cliMessage, messageOp: "resume-candidates", ref: "pair:1"}, messageRuntime(), &out, &diagnostic, func(_ context.Context, _ string, _ any, response any) error {
		*response.(*couchmessage.Response) = couchmessage.Response{Code: "ok", RecoveryPreview: &couchcore.RecoveryPreview{Incomplete: true, Candidates: []couchcore.RecoveryChoice{{SourceKind: sessioninventory.RecoverySourceNative, Label: "Native evidence", Agent: "codex", NativeID: "native-id", CWD: "/physical/repo"}}}}
		return nil
	})
	if code != 0 || strings.Contains(out.String(), "couch --resume") || strings.Contains(out.String(), "established native binding") || !strings.Contains(out.String(), "Refresh candidates") || !strings.Contains(out.String(), "Candidate scan incomplete") {
		t.Fatalf("incomplete evidence became authority: code=%d output=%s diagnostic=%s", code, &out, &diagnostic)
	}
}

func TestResumeCandidatesCLIRetainsPartialScreenActivity(t *testing.T) {
	var out, diagnostic bytes.Buffer
	activity := time.Date(2026, 10, 19, 11, 12, 0, 0, time.UTC)
	code := runMessageCLIWithCall(cliInvocation{kind: cliMessage, messageOp: "resume-candidates", ref: "pair:1"}, messageRuntime(), &out, &diagnostic, func(_ context.Context, _ string, _ any, response any) error {
		*response.(*couchmessage.Response) = couchmessage.Response{Code: "ok", RecoveryPreview: &couchcore.RecoveryPreview{Screen: couchcore.ScreenObservation{Availability: couchcore.ScreenUnavailable, Reason: "Capture missing", LastActivity: activity, ActivitySource: "capture-file activity\x1b]0;unsafe\x07"}, Reasons: []string{"Capture missing"}}}
		return nil
	})
	text := out.String()
	if code != 0 || !strings.Contains(text, "capture-file activity") || !strings.Contains(text, activity.Local().Format("2006-01-02 15:04 MST")) || strings.Count(text, "Capture missing") != 1 || strings.ContainsAny(text, "\x1b\x07") {
		t.Fatalf("partial screen fact lost or unsafe: code=%d output=%s diagnostic=%s", code, &out, &diagnostic)
	}
}

// TestResumeCandidatesCLIIncompleteScanSparesTheConfirmedChoice (pair#726):
// when the confirmed Resume from disk is offered, an incomplete scan says that
// choice comes from the established binding, not from the scan.
func TestResumeCandidatesCLIIncompleteScanSparesTheConfirmedChoice(t *testing.T) {
	var out, diagnostic bytes.Buffer
	code := runMessageCLIWithCall(cliInvocation{kind: cliMessage, messageOp: "resume-candidates", ref: "pair:6"}, messageRuntime(), &out, &diagnostic, func(_ context.Context, _ string, _ any, response any) error {
		*response.(*couchmessage.Response) = couchmessage.Response{Code: "ok", RecoveryPreview: &couchcore.RecoveryPreview{Incomplete: true, Proven: &couchcore.RecoveryChoice{Label: "Resume from disk (confirmed)", Agent: "codex", NativeID: "proven-native"}}}
		return nil
	})
	if code != 0 || !strings.Contains(out.String(), "comes from the established binding, not from this scan") {
		t.Fatalf("code=%d output=%s diagnostic=%s", code, &out, &diagnostic)
	}
}
