package sessioninventory

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"
)

type recoveryFake struct {
	Runtime
	entries []FileEntry
	data    map[string][]byte
	calls   []string
	listErr error
	change  bool
	readErr error
}

func (f *recoveryFake) NativeRoots(a Agent) []StorageRoot {
	return []StorageRoot{{Agent: a, Name: "codex-sessions"}}
}
func (f *recoveryFake) VisitRecoveryFiles(ctx context.Context, root StorageRoot, limit int, visit func(FileEntry) error) (int, error) {
	f.calls = append(f.calls, "list")
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	if f.listErr != nil {
		return 0, f.listErr
	}
	for i, e := range f.entries {
		if i >= limit {
			return i, ErrReadLimit
		}
		if err := visit(e); err != nil {
			return i + 1, err
		}
	}
	return len(f.entries), nil
}
func (f *recoveryFake) StatRecoveryFile(ctx context.Context, a Artifact) (FileEntry, error) {
	f.calls = append(f.calls, "stat")
	if err := ctx.Err(); err != nil {
		return FileEntry{}, err
	}
	for _, e := range f.entries {
		if e.Artifact == a {
			if f.change && len(f.calls) > 3 {
				e.MutationToken = "changed"
			}
			return e, nil
		}
	}
	return FileEntry{}, errors.New("missing")
}
func (f *recoveryFake) ReadRecoveryAt(ctx context.Context, expected FileEntry, offset, limit int64) ([]byte, bool, error) {
	a := expected.Artifact
	f.calls = append(f.calls, "read")
	if err := ctx.Err(); err != nil {
		return nil, false, err
	}
	if f.readErr != nil {
		return nil, false, f.readErr
	}
	b := f.data[a.RelativePath]
	if offset >= int64(len(b)) {
		return nil, true, nil
	}
	end := offset + limit
	if end > int64(len(b)) {
		end = int64(len(b))
	}
	return append([]byte(nil), b[offset:end]...), end == int64(len(b)), nil
}
func (f *recoveryFake) ResolveRecoveryCWD(ctx context.Context, cwd string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if cwd == "/alias" {
		return "/repo", nil
	}
	return cwd, nil
}
func recoveryCollectionFixture() (*recoveryFake, RecoveryCollectRequest) {
	e := recoveryCodexEntry()
	e.Artifact.StorageRoot = "codex-sessions"
	var lines []string
	for _, r := range recoveryCodexRecords(`"cli"`) {
		lines = append(lines, string(r.Bytes))
	}
	data := []byte(strings.Join(lines, "\n") + "\n")
	e.Size = int64(len(data))
	mt := time.Date(2026, 8, 28, 10, 3, 0, 0, time.UTC)
	e.ModTime = &mt
	launch := mt.Add(-5 * time.Minute)
	capture := mt.Add(time.Minute)
	return &recoveryFake{entries: []FileEntry{e}, data: map[string][]byte{e.Artifact.RelativePath: data}}, RecoveryCollectRequest{Agent: AgentCodex, TargetCWD: "/alias", ProducingLaunch: &launch, CapturedAt: &capture, RenderedTail: []string{recoveryTestText}}
}
func TestRecoveryCollectorUsesBoundedSnapshotAndPhysicalCWD(t *testing.T) {
	f, q := recoveryCollectionFixture()
	got := CollectRecoveryEvidence(context.Background(), f, q)
	if got.Incomplete || len(got.Ranked) != 1 || got.Ranked[0].Score.Percent != 100 {
		t.Fatalf("evidence=%+v", got)
	}
	if strings.Join(f.calls, ",") != "list,stat,read,stat" {
		t.Fatalf("calls=%v", f.calls)
	}
}
func TestRecoveryCollectorReportsMissingEpochAndCapability(t *testing.T) {
	f, q := recoveryCollectionFixture()
	q.ProducingLaunch = nil
	if got := CollectRecoveryEvidence(context.Background(), f, q); !got.Incomplete {
		t.Fatal("missing epoch became complete")
	}
	if got := CollectRecoveryEvidence(context.Background(), emptyCLIRuntime{}, q); !got.Incomplete {
		t.Fatal("unbounded runtime became complete")
	}
}
func TestRecoveryCollectorFailureStates(t *testing.T) {
	for _, kind := range []string{"list", "read", "moving", "torn", "cancel", "oversize"} {
		t.Run(kind, func(t *testing.T) {
			f, q := recoveryCollectionFixture()
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			switch kind {
			case "list":
				f.listErr = errors.New("listing failed")
			case "read":
				f.readErr = errors.New("read failed")
			case "moving":
				f.change = true
			case "torn":
				p := f.entries[0].Artifact.RelativePath
				f.data[p] = append(f.data[p], []byte(`{"type":`)...)
				f.entries[0].Size = int64(len(f.data[p]))
			case "cancel":
				cancel()
			case "oversize":
				f.entries[0].Size = 257 << 20
			}
			got := CollectRecoveryEvidence(ctx, f, q)
			if !got.Incomplete || len(got.Ranked) != 0 {
				t.Fatalf("failure yielded candidate: %+v", got)
			}
		})
	}
}
func TestRecoveryCollectorCaptureCutoffKeepsOriginalTail(t *testing.T) {
	f, q := recoveryCollectionFixture()
	p := f.entries[0].Artifact.RelativePath
	later := strings.Replace(string(recoveryCodexRecords(`"cli"`)[1].Bytes), "10:02:00Z", "11:02:00Z", 1)
	later = strings.Replace(later, recoveryTestText, strings.Repeat("later ", 20), 1)
	for i := 0; i < 3; i++ {
		f.data[p] = append(f.data[p], []byte(later+"\n")...)
	}
	f.entries[0].Size = int64(len(f.data[p]))
	got := CollectRecoveryEvidence(context.Background(), f, q)
	if len(got.Ranked) != 1 || got.Ranked[0].Score.Percent != 100 {
		t.Fatalf("future displaced captured tail: %+v", got)
	}
}

func TestRecoveryExactRevalidationDoesNotEnumerate(t *testing.T) {
	f, q := recoveryCollectionFixture()
	expected := f.entries[0]
	if err := RevalidateRecoveryCandidate(context.Background(), f, expected, q.Agent, recoveryTestID, q.TargetCWD, *q.CapturedAt); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(strings.Join(f.calls, ","), "list") {
		t.Fatal("exact selection enumerated alternatives")
	}
	f.entries[0].MutationToken = "changed"
	if err := RevalidateRecoveryCandidate(context.Background(), f, expected, q.Agent, recoveryTestID, q.TargetCWD, *q.CapturedAt); err == nil {
		t.Fatal("changed identity revalidated")
	}
}
func TestRecoveryExactRevalidationRefusesWrongCWDSourceOrIdentity(t *testing.T) {
	for _, kind := range []string{"cwd", "source", "id", "cutoff"} {
		t.Run(kind, func(t *testing.T) {
			f, q := recoveryCollectionFixture()
			id := recoveryTestID
			cutoff := *q.CapturedAt
			switch kind {
			case "cwd":
				q.TargetCWD = "/other"
			case "id":
				id = "019d2222-2222-7222-8222-222222222222"
			case "cutoff":
				cutoff = time.Time{}
			case "source":
				p := f.entries[0].Artifact.RelativePath
				f.data[p] = []byte(strings.Replace(string(f.data[p]), `"source":"cli"`, `"source":"exec"`, 1))
				f.entries[0].Size = int64(len(f.data[p]))
			}
			if err := RevalidateRecoveryCandidate(context.Background(), f, f.entries[0], q.Agent, id, q.TargetCWD, cutoff); err == nil {
				t.Fatal("invalid exact candidate accepted")
			}
		})
	}
}
func TestRecoveryCollectorCountsLimits(t *testing.T) {
	f, q := recoveryCollectionFixture()
	original := f.entries[0]
	raw := string(f.data[original.Artifact.RelativePath])
	f.entries = nil
	for i := 0; i < 65; i++ {
		id := fmt.Sprintf("019d1111-1111-7111-8111-%012d", i)
		e := original
		e.Artifact.RelativePath = strings.Replace(e.Artifact.RelativePath, recoveryTestID, id, 1)
		data := []byte(strings.ReplaceAll(raw, recoveryTestID, id))
		e.Size = int64(len(data))
		f.entries = append(f.entries, e)
		f.data[e.Artifact.RelativePath] = data
	}
	got := CollectRecoveryEvidence(context.Background(), f, q)
	if !got.Incomplete || len(got.Candidates) != 64 {
		t.Fatalf("root bound=%+v", got)
	}
	f, q = recoveryCollectionFixture()
	for len(f.entries) < RecoveryMetadataLimit+1 {
		f.entries = append(f.entries, original)
	}
	got = CollectRecoveryEvidence(context.Background(), f, q)
	if !got.Incomplete || got.MetadataEntries > RecoveryMetadataLimit {
		t.Fatalf("metadata bound=%+v", got)
	}
}

func TestRecoveryPersistedFingerprintNeedsNoDerivedKindOrBirth(t *testing.T) {
	f, q := recoveryCollectionFixture()
	birth := time.Now()
	f.entries[0].BirthTime = &birth
	expected := f.entries[0]
	expected.BirthTime = nil
	if err := RevalidateRecoveryCandidate(context.Background(), f, expected, q.Agent, recoveryTestID, q.TargetCWD, *q.CapturedAt); err != nil {
		t.Fatal(err)
	}
}
func TestRecoveryCollectorChargesTTYBudget(t *testing.T) {
	f, q := recoveryCollectionFixture()
	q.BytesAlreadyRead = RecoveryTotalReadLimit - 1
	got := CollectRecoveryEvidence(context.Background(), f, q)
	if !got.Incomplete || len(got.Ranked) != 0 || got.BytesRead > RecoveryTotalReadLimit {
		t.Fatalf("combined budget=%+v", got)
	}
}
