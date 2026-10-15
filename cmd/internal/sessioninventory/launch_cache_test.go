package sessioninventory_test

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/xianxu/pair/cmd/internal/sessioninventory"
	"github.com/xianxu/pair/cmd/internal/sessioninventorytest"
)

type observedLaunchRuntime struct {
	cachingRuntime
	decoded  atomic.Int64
	readHook func()
}

func (r *observedLaunchRuntime) DecodeWork(_ sessioninventory.Artifact, n int64) { r.decoded.Add(n) }
func (r *observedLaunchRuntime) ReadAt(a sessioninventory.Artifact, off, limit int64) ([]byte, bool, error) {
	data, eof, err := r.FakeRuntime.ReadAt(a, off, limit)
	if r.readHook != nil {
		r.readHook()
	}
	return data, eof, err
}
func launchCacheFixture() (*observedLaunchRuntime, sessioninventory.Artifact) {
	fake := sessioninventorytest.NewFakeRuntime()
	fake.SetPairDataRoot(sessioninventory.StorageRoot{Name: "pair-data", Path: "/pair/scope"})
	ledger := sessioninventory.Artifact{StorageRoot: "pair-data", RelativePath: "ledger-work.jsonl"}
	fake.PutFile(sessioninventory.FileEntry{Artifact: ledger, MutationToken: "one"}, []byte(v1Launch))
	return &observedLaunchRuntime{cachingRuntime: cachingRuntime{FakeRuntime: fake, cache: sessioninventory.NewOwnerLaunchCache()}}, ledger
}
func TestOwnerLaunchCachedResumeRead(t *testing.T) {
	rt, ledger := launchCacheFixture()
	want, err := sessioninventory.QueryResumeTarget(rt, "scope", "work", sessioninventory.AgentCodex)
	if err != nil {
		t.Fatal(err)
	}
	reads := rt.OperationCount(sessioninventorytest.OperationReadAt, "pair-data:ledger-work.jsonl")
	if reads == 0 {
		t.Fatal("cold ledger never read")
	}
	got, err := sessioninventory.QueryResumeTarget(rt, "scope", "work", sessioninventory.AgentCodex)
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Fatalf("warm resume=%+v %v, want %+v", got, err, want)
	}
	if n := rt.OperationCount(sessioninventorytest.OperationReadAt, "pair-data:ledger-work.jsonl"); n != reads {
		t.Fatalf("unchanged resume re-read ledger: %d -> %d", reads, n)
	}
	if rt.decoded.Load() == 0 {
		t.Fatal("cold decode observer disconnected")
	}
	before := rt.decoded.Load()
	rt.AppendFile(ledger, []byte(v1Launch), "two")
	got, err = sessioninventory.QueryResumeTarget(rt, "scope", "work", sessioninventory.AgentCodex)
	if err != nil || got.LaunchOrdinal != 2 || rt.decoded.Load() <= before {
		t.Fatalf("mutation not reparsed: %+v %v", got, err)
	}
}
func TestOwnerLaunchDoesNotPublishChangedDuringRead(t *testing.T) {
	rt, ledger := launchCacheFixture()
	var once sync.Once
	rt.readHook = func() {
		once.Do(func() {
			rt.PutFile(sessioninventory.FileEntry{Artifact: ledger, MutationToken: "two"}, []byte(v1Launch+v1Launch))
		})
	}
	got, err := sessioninventory.QueryResumeTarget(rt, "scope", "work", sessioninventory.AgentCodex)
	if err != nil || got.LaunchOrdinal != 2 {
		t.Fatalf("published superseded ledger: %+v %v", got, err)
	}
}
func TestOwnerLaunchCanceledWaiter(t *testing.T) {
	rt, _ := launchCacheFixture()
	entered, release := make(chan struct{}), make(chan struct{})
	var once sync.Once
	rt.readHook = func() { once.Do(func() { close(entered); <-release }) }
	done := make(chan error, 1)
	go func() {
		_, err := sessioninventory.QueryResumeTarget(rt, "scope", "work", sessioninventory.AgentCodex)
		done <- err
	}()
	<-entered
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := sessioninventory.QueryResumeTargetContext(ctx, rt, "scope", "work", sessioninventory.AgentCodex)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("waiter error=%v", err)
	}
	close(release)
	if err := <-done; err != nil {
		t.Fatal(err)
	}
}
func TestOwnerLaunchCacheBoundsOwnerHistory(t *testing.T) {
	rt, _ := launchCacheFixture()
	for i := 0; i < 129; i++ {
		tag := fmt.Sprintf("owner-%d", i)
		a := sessioninventory.Artifact{StorageRoot: "pair-data", RelativePath: "ledger-" + tag + ".jsonl"}
		row := []byte(fmt.Sprintf("{\"v\":1,\"kind\":\"launch\",\"scope_key\":\"scope\",\"tag\":%q,\"agent\":\"codex\",\"pair_log_offset\":0,\"native_watermarks\":[]}\n", tag))
		rt.PutFile(sessioninventory.FileEntry{Artifact: a, MutationToken: "one"}, row)
		if _, err := sessioninventory.QuerySession(rt, "scope", tag, sessioninventory.AgentCodex); err != nil {
			t.Fatal(err)
		}
	}
	before := rt.OperationCount(sessioninventorytest.OperationReadAt, "pair-data:ledger-owner-0.jsonl")
	if _, err := sessioninventory.QuerySession(rt, "scope", "owner-0", sessioninventory.AgentCodex); err != nil {
		t.Fatal(err)
	}
	if after := rt.OperationCount(sessioninventorytest.OperationReadAt, "pair-data:ledger-owner-0.jsonl"); after <= before {
		t.Fatal("least recently used owner was not evicted")
	}
}

func TestOwnerLaunchCachedFallbackKeepsFreshMetadataAdmission(t *testing.T) {
	rt, ledger := launchCacheFixture()
	root := sessioninventory.StorageRoot{Name: "claude-projects", Path: "/native", Agent: sessioninventory.AgentClaude}
	rt.AddRoot(root)
	const old = "9a99ff57-cb0f-4590-8ea7-d684f2ff5a3f"
	const fresh = "11111111-1111-4111-8111-111111111111"
	launch := func(id string) string {
		return `{"v":3,"kind":"launch","scope_key":"scope","tag":"work","agent":"claude","pair_log_offset":0,"artifact_boundaries":[],"requested_native_id":"` + id + `","request_origin":"chosen-id","baseline_complete":true}` + "\n"
	}
	binding := `{"v":3,"kind":"binding","scope_key":"scope","tag":"work","agent":"claude","launch_ordinal":1,"root_native_id":"` + old + `","confirmation_reason":"chosen-id"}` + "\n"
	rt.PutFile(sessioninventory.FileEntry{Artifact: ledger, MutationToken: "fallback"}, []byte(launch(old)+binding+launch(fresh)))
	query := func() sessioninventory.ResumeTarget {
		t.Helper()
		got, err := sessioninventory.QueryResumeTarget(rt, "scope", "work", sessioninventory.AgentClaude)
		if err != nil {
			t.Fatal(err)
		}
		return got
	}
	want := query()
	if want.NativeID != old || want.FellBackFrom != 3 {
		t.Fatalf("missing earlier conversation: %+v", want)
	}
	decoded := rt.decoded.Load()
	if got := query(); !reflect.DeepEqual(got, want) {
		t.Fatalf("cached fallback %+v != %+v", got, want)
	}
	rt.PutFile(sessioninventory.FileEntry{Artifact: sessioninventory.Artifact{StorageRoot: root.Name, RelativePath: "project/" + fresh + ".jsonl"}}, []byte("fresh turn"))
	if got := query(); got.NativeID != fresh || got.FellBackFrom != 0 {
		t.Fatalf("cached durable evidence hid fresh native presence: %+v", got)
	}
	if rt.decoded.Load() != decoded {
		t.Fatal("native metadata change unnecessarily decoded ledger")
	}
}

func TestOwnerLaunchConcurrentQueriesShareOneParse(t *testing.T) {
	rt, _ := launchCacheFixture()
	entered, release := make(chan struct{}), make(chan struct{})
	var once sync.Once
	rt.readHook = func() { once.Do(func() { close(entered); <-release }) }
	done := make(chan error, 8)
	for i := 0; i < 8; i++ {
		go func() {
			_, err := sessioninventory.QueryResumeTarget(rt, "scope", "work", sessioninventory.AgentCodex)
			done <- err
		}()
	}
	<-entered
	ctx, cancel := context.WithCancel(context.Background())
	waiter := make(chan error, 1)
	go func() {
		_, err := sessioninventory.QueryResumeTargetContext(ctx, rt, "scope", "work", sessioninventory.AgentCodex)
		waiter <- err
	}()
	cancel()
	if err := <-waiter; !errors.Is(err, context.Canceled) {
		t.Fatalf("waiter cancellation=%v", err)
	}
	close(release)
	for i := 0; i < 8; i++ {
		if err := <-done; err != nil {
			t.Fatal(err)
		}
	}
	if reads := rt.OperationCount(sessioninventorytest.OperationReadAt, "pair-data:ledger-work.jsonl"); reads != 1 {
		t.Fatalf("cold queries each parsed ledger: %d reads", reads)
	}
}

func TestOwnerLaunchCanceledLoaderDoesNotPublish(t *testing.T) {
	rt, _ := launchCacheFixture()
	ctx, cancel := context.WithCancel(context.Background())
	var once sync.Once
	rt.readHook = func() { once.Do(cancel) }
	if _, err := sessioninventory.QueryResumeTargetContext(ctx, rt, "scope", "work", sessioninventory.AgentCodex); !errors.Is(err, context.Canceled) {
		t.Fatalf("loader cancellation=%v", err)
	}
	before := rt.OperationCount(sessioninventorytest.OperationReadAt, "pair-data:ledger-work.jsonl")
	if _, err := sessioninventory.QueryResumeTarget(rt, "scope", "work", sessioninventory.AgentCodex); err != nil {
		t.Fatal(err)
	}
	if reads := rt.OperationCount(sessioninventorytest.OperationReadAt, "pair-data:ledger-work.jsonl"); reads != before+1 {
		t.Fatal("canceled loader published evidence")
	}
}

func TestOwnerLaunchRepeatedDriftDoesNotPublish(t *testing.T) {
	rt, ledger := launchCacheFixture()
	var mutation int
	rt.readHook = func() {
		mutation++
		rt.AppendFile(ledger, []byte(v1Launch), sessioninventory.MutationToken(fmt.Sprintf("change-%d", mutation)))
	}
	if _, err := sessioninventory.QueryResumeTarget(rt, "scope", "work", sessioninventory.AgentCodex); !errors.Is(err, sessioninventory.ErrArtifactChanged) {
		t.Fatalf("moving source admitted: %v", err)
	}
	rt.readHook = nil
	got, err := sessioninventory.QueryResumeTarget(rt, "scope", "work", sessioninventory.AgentCodex)
	if err != nil || got.LaunchOrdinal != 3 {
		t.Fatalf("stable retry=%+v %v", got, err)
	}
}
