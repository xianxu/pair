package launcher

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/artifactpath"
	"github.com/xianxu/pair/cmd/internal/durablefile"
	"github.com/xianxu/pair/cmd/internal/procutil"
	"github.com/xianxu/pair/cmd/internal/storagegc"
)

func admissionStorePath(t *testing.T, home string) string {
	t.Helper()
	p, e := artifactpath.ResolvePairCache(home)
	if e != nil {
		t.Fatal(e)
	}
	path, e := p.NativeAdmissions()
	if e != nil {
		t.Fatal(e)
	}
	return path
}
func TestNativeAdmissionStoreRejectsCorruptUnsafeAndOversizedState(t *testing.T) {
	for _, kind := range []string{"version", "unknown-field", "duplicate-field", "duplicate-key", "oversized", "directory", "symlink"} {
		t.Run(kind, func(t *testing.T) {
			w := newAdmissionWorld()
			home := t.TempDir()
			s := w.store(home)
			r := admissionRequest(t, "native")
			reserveAdmission(t, s, r)
			path := admissionStorePath(t, home)
			switch kind {
			case "version":
				os.WriteFile(path, []byte(`{"Version":2,"Entries":[]}`), 0600)
			case "unknown-field":
				os.WriteFile(path, []byte(`{"Version":1,"Entries":[],"other":1}`), 0600)
			case "duplicate-field":
				os.WriteFile(path, []byte(`{"Version":1,"Version":1,"Entries":[]}`), 0600)
			case "duplicate-key":
				raw, _ := os.ReadFile(path)
				var state nativeAdmissionFile
				json.Unmarshal(raw, &state)
				state.Entries = append(state.Entries, state.Entries[0])
				raw, _ = json.Marshal(state)
				os.WriteFile(path, raw, 0600)
			case "oversized":
				os.WriteFile(path, []byte(strings.Repeat("x", maxNativeAdmissionBytes+1)), 0600)
			case "directory":
				os.Remove(path)
				os.Mkdir(path, 0700)
			case "symlink":
				os.Remove(path)
				os.Symlink(t.TempDir()+"/missing", path)
			}
			r.NativeID = "other"
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			if _, e := s.Reserve(ctx, r, noLegacy); e == nil {
				t.Fatal("unsafe store accepted")
			}
		})
	}
}
func TestNativeAdmissionPublicationFailureNeverReleasesOwnership(t *testing.T) {
	for _, stage := range []string{"reserve", "claim", "publish"} {
		t.Run(stage, func(t *testing.T) {
			w := newAdmissionWorld()
			s := w.store(t.TempDir())
			r := admissionRequest(t, "native")
			var grant string
			var claim NativeAdmissionClaim
			if stage != "reserve" {
				grant = reserveAdmission(t, s, r)
				w.as(12)
			}
			if stage == "publish" {
				var e error
				claim, e = s.Claim(context.Background(), grant, r.Agent, r.CommandDigest)
				if e != nil {
					t.Fatal(e)
				}
			}
			s.io.write = func(path string, raw []byte) error {
				if e := durablefile.WriteAtomic(path, raw, ".test-admission-*"); e != nil {
					return e
				}
				return errors.New("post-rename sync failed")
			}
			switch stage {
			case "reserve":
				if _, e := s.Reserve(context.Background(), r, noLegacy); e == nil {
					t.Fatal("failure hidden")
				}
			case "claim":
				if _, e := s.Claim(context.Background(), grant, r.Agent, r.CommandDigest); e == nil {
					t.Fatal("failure hidden")
				}
			case "publish":
				if e := s.PublishChild(context.Background(), claim, 13); e == nil {
					t.Fatal("failure hidden")
				}
			}
			s.io.write = nil
			w.as(7)
			r.Attempt = "replacement"
			if _, e := s.Reserve(context.Background(), r, noLegacy); e == nil {
				t.Fatal("uncertain publication released native key")
			}
		})
	}
}
func TestNativeAdmissionCapacityAndBoundedCollection(t *testing.T) {
	w := newAdmissionWorld()
	home := t.TempDir()
	s := w.store(home)
	r := admissionRequest(t, "native")
	g := reserveAdmission(t, s, r)
	parsed, e := decodeAdmissionGrant(g)
	if e != nil {
		t.Fatal(e)
	}
	state := nativeAdmissionFile{Version: 1}
	for i := 0; i < maxNativeAdmissions; i++ {
		next := r
		next.NativeID = strings.Repeat("n", i+1)
		entry, e := AdvanceNativeAdmission(nil, admissionChange(nativeAdmissionGrant{Request: next, Helper: parsed.Helper, Token: parsed.Token}, NativeAdmissionReserve, storagegc.ProcessIdentity{}))
		if e != nil {
			t.Fatal(e)
		}
		state.Entries = append(state.Entries, *entry)
	}
	raw, _ := json.Marshal(state)
	os.WriteFile(admissionStorePath(t, home), raw, 0600)
	if _, e = s.Reserve(context.Background(), r, noLegacy); e == nil {
		t.Fatal("capacity overflow accepted")
	}
	// Only a genuinely both-dead running entry is collected at capacity.
	state.Entries[0].Phase = NativeAdmissionRunning
	state.Entries[0].Wrapper = storagegc.ProcessIdentity{PID: 12, Birth: "wrapper"}
	state.Entries[0].Child = storagegc.ProcessIdentity{PID: 13, Birth: "child"}
	w.lives[12] = storagegc.ProcessDead
	w.lives[13] = storagegc.ProcessDead
	raw, _ = json.Marshal(state)
	os.WriteFile(admissionStorePath(t, home), raw, 0600)
	if _, e = s.Reserve(context.Background(), r, noLegacy); e != nil {
		t.Fatal("both-dead entry did not free bounded capacity", e)
	}
}
func TestAdvanceNativeAdmissionDoesNotMutateInputsOrInferDeath(t *testing.T) {
	r := admissionRequest(t, "native")
	g := nativeAdmissionGrant{Request: r, Token: strings.Repeat("a", 64), Helper: storagegc.ProcessIdentity{PID: 7, Birth: "helper"}}
	entry, e := AdvanceNativeAdmission(nil, admissionChange(g, NativeAdmissionReserve, storagegc.ProcessIdentity{}))
	if e != nil {
		t.Fatal(e)
	}
	before := *entry
	spawning, e := AdvanceNativeAdmission(entry, admissionChange(g, NativeAdmissionSpawn, storagegc.ProcessIdentity{PID: 12, Birth: "wrapper"}))
	if e != nil {
		t.Fatal(e)
	}
	if *entry != before || spawning.Phase != NativeAdmissionSpawning {
		t.Fatal("reducer mutated prior authority")
	}
	for _, entry := range []*NativeAdmissionEntry{entry, spawning} {
		if _, e := AdvanceNativeAdmission(entry, NativeAdmissionChange{Transition: NativeAdmissionCollect, WrapperLiveness: storagegc.ProcessDead, ChildLiveness: storagegc.ProcessDead}); e == nil {
			t.Fatal("unconfirmed entry collected without exact child")
		}
	}
}
func TestNativeAdmissionChecksUnderLockAndHonorsCancellation(t *testing.T) {
	w := newAdmissionWorld()
	s := w.store(t.TempDir())
	r := admissionRequest(t, "native")
	calls := 0
	_, e := s.Reserve(context.Background(), r, func(ctx context.Context, owners NativeAdmissionOwners) error {
		calls++
		return errors.New("legacy writer")
	})
	if e == nil || calls != 1 {
		t.Fatalf("legacy check calls=%d err=%v", calls, e)
	}
	if _, e = s.Reserve(context.Background(), r, noLegacy); e != nil {
		t.Fatal("failed precheck created authority", e)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, e = s.Reserve(ctx, admissionRequest(t, "other"), noLegacy); !errors.Is(e, context.Canceled) {
		t.Fatalf("cancellation=%v", e)
	}
}

func TestNativeAdmissionOperationHasAnInternalDeadline(t *testing.T) {
	w := newAdmissionWorld()
	s := w.store(t.TempDir())
	r := admissionRequest(t, "native")
	_, err := s.Reserve(context.Background(), r, func(ctx context.Context, _ NativeAdmissionOwners) error {
		if _, ok := ctx.Deadline(); !ok {
			return errors.New("unbounded native admission callback")
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
func TestNativeAdmissionOwnershipViewCannotEscapeTheLockedCallback(t *testing.T) {
	w := newAdmissionWorld()
	s := w.store(t.TempDir())
	r := admissionRequest(t, "other")
	g := reserveAdmission(t, s, r)
	w.as(12)
	if _, err := s.Claim(context.Background(), g, r.Agent, r.CommandDigest); err != nil {
		t.Fatal(err)
	}
	var escaped NativeAdmissionOwners
	w.as(7)
	if _, err := s.Reserve(context.Background(), admissionRequest(t, "native"), func(_ context.Context, v NativeAdmissionOwners) error { escaped = v; return nil }); err != nil {
		t.Fatal(err)
	}
	snapshot := recoveryTerminalSnapshot{Processes: []procutil.Process{{PID: 12, PPID: 1, Identity: "wrapper"}}}
	if escaped.coversDifferentWrapper("codex", "native", snapshot.Processes[0], snapshot) {
		t.Fatal("ownership facts escaped their locked check")
	}
}

func TestNativeAdmissionCollectionStopsAtCancellationBetweenProcessProbes(t *testing.T) {
	for _, cancelAt := range []int{1, 2} {
		t.Run(fmt.Sprint(cancelAt), func(t *testing.T) {
			world := newAdmissionWorld()
			store := world.store(t.TempDir())
			req := admissionRequest(t, "selected")
			rawGrant := reserveAdmission(t, store, req)
			grant, err := decodeAdmissionGrant(rawGrant)
			if err != nil {
				t.Fatal(err)
			}
			state := nativeAdmissionFile{Version: 1}
			for i := 0; i < 3; i++ {
				old := req
				old.NativeID = fmt.Sprintf("old-%d", i)
				state.Entries = append(state.Entries, NativeAdmissionEntry{Request: old, Token: grant.Token, Helper: grant.Helper, Phase: NativeAdmissionRunning, Wrapper: storagegc.ProcessIdentity{PID: 12 + i*10, Birth: "wrapper"}, Child: storagegc.ProcessIdentity{PID: 13 + i*10, Birth: "child"}})
			}
			path := admissionStorePath(t, store.Home)
			before, err := json.Marshal(state)
			if err != nil {
				t.Fatal(err)
			}
			if err = os.WriteFile(path, before, 0600); err != nil {
				t.Fatal(err)
			}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			probes, checks, writes := 0, 0, 0
			store.io.inspect = func(storagegc.ProcessIdentity) storagegc.Liveness {
				probes++
				if probes == cancelAt {
					cancel()
				}
				return storagegc.ProcessDead
			}
			store.io.write = func(string, []byte) error { writes++; return nil }
			obtained, err := store.Reserve(ctx, req, func(context.Context, NativeAdmissionOwners) error { checks++; return nil })
			if !errors.Is(err, context.Canceled) || obtained != "" {
				t.Fatalf("grant=%q err=%v", obtained, err)
			}
			if probes != cancelAt || checks != 0 || writes != 0 {
				t.Fatalf("work continued after cancellation: probes=%d wanted=%d checks=%d writes=%d", probes, cancelAt, checks, writes)
			}
			after, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(before, after) {
				t.Fatal("canceled collection changed authority")
			}
		})
	}
}
