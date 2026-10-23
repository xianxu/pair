package couchcore

import (
	"context"
	"errors"
	"io/fs"
	"testing"

	"github.com/xianxu/pair/cmd/internal/sessioninventory"
	"github.com/xianxu/pair/cmd/internal/threadrecord"
)

func editedRestartObserved(agent, id string) *threadrecord.TranscriptBinding {
	b := &threadrecord.TranscriptBinding{Agent: agent, NativeID: id, Source: threadrecord.TranscriptSourceLedger}
	if agent == "codex" {
		b.StorageRoot, b.RelativePath = "codex-sessions", "2026/08/28/rollout-root-"+id+".jsonl"
	} else {
		b.StorageRoot, b.RelativePath = bindRoot, "project/"+id+".jsonl"
	}
	return b
}

func TestEditedRestartNativeEvidenceKeepsAbsenceAndReplacementDistinct(t *testing.T) {
	for _, agent := range []string{"claude", "codex"} {
		t.Run(agent, func(t *testing.T) {
			a, b := bindID(3), bindID(5)
			thread := ThreadRecord{LatestLaunchProfile: &LaunchProfile{Agent: agent}}
			found := func(id string) TranscriptBind {
				return TranscriptBind{Resolution: NativeBindingResolution{Status: sessioninventory.BindingEstablished, NativeID: id}, Observed: editedRestartObserved(agent, id)}
			}
			missing := func(id string) TranscriptBind { return TranscriptBind{Observed: editedRestartObserved(agent, id)} }
			uncertainMissing := missing(a)
			uncertainMissing.Resolution.Status = sessioninventory.BindingProvisional
			for _, tc := range []struct {
				name     string
				bound    TranscriptBind
				err      error
				expected *string
				kind     EditedRestartNativeKind
				id       string
				success  bool
			}{
				{"discover-found", found(a), nil, nil, EditedRestartNativeFound, a, true},
				{"found-exact", found(a), nil, &a, EditedRestartNativeFound, a, true},
				{"discover-missing", missing(a), refuseResume(ResumeTranscriptMissing, "missing"), nil, EditedRestartNativeMissing, a, true},
				{"missing-exact", missing(a), refuseResume(ResumeTranscriptMissing, "missing"), &a, EditedRestartNativeMissing, a, true},
				{"changed-found", found(b), nil, &a, EditedRestartNativeChanged, b, false},
				{"changed-missing", missing(b), refuseResume(ResumeTranscriptMissing, "missing"), &a, EditedRestartNativeChanged, b, false},
				{"unbound-discovery", TranscriptBind{Resolution: NativeBindingResolution{Status: sessioninventory.BindingUnbound}}, nil, nil, EditedRestartNativeMissing, "", true},
				{"unbound-after-found", TranscriptBind{Resolution: NativeBindingResolution{Status: sessioninventory.BindingUnbound}}, nil, &a, EditedRestartNativeMissing, a, true},
				{"provisional-missing-is-unknown", uncertainMissing, refuseResume(ResumeTranscriptMissing, "missing"), &a, EditedRestartNativeUnknown, "", false},
				{"unbound-requested-is-unknown", TranscriptBind{Resolution: NativeBindingResolution{Status: sessioninventory.BindingUnbound, RequestedNativeID: a}}, refuseBinding(ResumeBindingUnbound), &a, EditedRestartNativeUnknown, "", false},
				{"default-is-unknown", TranscriptBind{}, nil, &a, EditedRestartNativeUnknown, "", false},
				{"unlocated-missing", TranscriptBind{}, refuseResume(ResumeTranscriptMissing, "missing"), &a, EditedRestartNativeUnknown, "", false},
				{"permission-is-unknown", missing(a), fs.ErrPermission, &a, EditedRestartNativeUnknown, "", false},
				{"cancellation-is-unknown", missing(a), context.Canceled, &a, EditedRestartNativeUnknown, "", false},
				{"provisional-id-is-unknown", TranscriptBind{Resolution: NativeBindingResolution{Status: sessioninventory.BindingProvisional, NativeID: a}}, nil, &a, EditedRestartNativeUnknown, "", false},
				{"provisional-fresh-is-unknown", TranscriptBind{Resolution: NativeBindingResolution{Status: sessioninventory.BindingProvisional, FreshRequired: true, RequestedNativeID: a}}, nil, &a, EditedRestartNativeUnknown, "", false},
				{"ambiguous-is-unknown", TranscriptBind{Resolution: NativeBindingResolution{Status: sessioninventory.BindingAmbiguous}}, nil, &a, EditedRestartNativeUnknown, "", false},
				{"incomplete-unbound", TranscriptBind{Resolution: NativeBindingResolution{Status: sessioninventory.BindingUnbound, ObservationIncomplete: true}}, nil, &a, EditedRestartNativeUnknown, "", false},
				{"empty-established", TranscriptBind{Resolution: NativeBindingResolution{Status: sessioninventory.BindingEstablished}}, nil, &a, EditedRestartNativeUnknown, "", false},
			} {
				t.Run(tc.name, func(t *testing.T) {
					got, err := classifyEditedRestartNative(thread, tc.expected, tc.bound, tc.err)
					if got.Kind != tc.kind || got.NativeID != tc.id || (err == nil) != tc.success {
						t.Fatalf("got %+v, %v; want kind=%v id=%q success=%t", got, err, tc.kind, tc.id, tc.success)
					}
					if errors.Is(tc.err, fs.ErrPermission) && !errors.Is(err, fs.ErrPermission) {
						t.Fatal("lost permission cause")
					}
					if errors.Is(tc.err, context.Canceled) && !errors.Is(err, context.Canceled) {
						t.Fatal("lost cancellation cause")
					}
					if got.Transcript != nil {
						if got.Transcript == tc.bound.Observed || got.Transcript == tc.bound.Store {
							t.Fatal("observation aliases binder")
						}
						before := got.Transcript.NativeID
						tc.bound.Observed.NativeID = "mutated"
						if got.Transcript.NativeID != before {
							t.Fatal("observation changed")
						}
					}
				})
			}
			empty := ""
			got, err := classifyEditedRestartNative(thread, &empty, found(b), nil)
			if got.Kind != EditedRestartNativeChanged || err == nil {
				t.Fatal("initial absence silently adopted new conversation", got, err)
			}
			forged := missing(a)
			forged.Observed.Agent = "foreign"
			got, err = classifyEditedRestartNative(thread, &a, forged, refuseResume(ResumeTranscriptMissing, "missing"))
			if got.Kind != EditedRestartNativeUnknown || err == nil {
				t.Fatal("foreign artifact became absence", got, err)
			}
		})
	}
}

func TestResolveEditedRestartEvidenceUsesStatefulBindingSeam(t *testing.T) {
	for _, agent := range []string{"claude", "codex"} {
		t.Run(agent, func(t *testing.T) {
			env := newTestEnv(t, "/repo")
			thread := createParkedThreadInCouch(t, env, LaunchProfile{Agent: agent, Argv: []string{}})
			before := env.Artifacts.BindingResolutions()
			// The real resolver seam returns a typed unbound refusal, not a nil error.
			got, err := env.Couch.resolveEditedRestartEvidence(context.Background(), thread, nil)
			if err != nil || got.Kind != EditedRestartNativeMissing || got.NativeID != "" {
				t.Fatal("explicit unbound was not absence", got, err)
			}
			a, b := bindID(3), bindID(5)
			env.Artifacts.SetNativeBinding(thread.Address, agent, sessioninventory.BindingEstablished, a)
			got, err = env.Couch.resolveEditedRestartEvidence(context.Background(), thread, &a)
			if err != nil || got.Kind != EditedRestartNativeFound || got.NativeID != a {
				t.Fatal(got, err)
			}
			env.Artifacts.SetNativeBinding(thread.Address, agent, sessioninventory.BindingEstablished, b)
			got, err = env.Couch.resolveEditedRestartEvidence(context.Background(), thread, &a)
			if err == nil || got.Kind != EditedRestartNativeChanged {
				t.Fatal("replacement inherited accepted ID", got, err)
			}
			env.Artifacts.SetNativeBinding(thread.Address, agent, sessioninventory.BindingUnbound, "")
			got, err = env.Couch.resolveEditedRestartEvidence(context.Background(), thread, &a)
			if err != nil || got.Kind != EditedRestartNativeMissing || got.NativeID != a {
				t.Fatal("lost exact missing anchor", got, err)
			}
			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			got, err = env.Couch.resolveEditedRestartEvidence(ctx, thread, &a)
			if !errors.Is(err, context.Canceled) || got.Kind != EditedRestartNativeUnknown {
				t.Fatal("cancel became missing", got, err)
			}
			if calls := env.Artifacts.BindingResolutions() - before; calls != 4 {
				t.Fatalf("native resolution calls=%d want4", calls)
			}
		})
	}
}
