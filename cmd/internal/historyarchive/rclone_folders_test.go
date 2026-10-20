package historyarchive_test

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	ha "github.com/xianxu/pair/cmd/internal/historyarchive"
	"github.com/xianxu/pair/cmd/internal/historyarchive/archivetest"
)

type folderRecordingRunner struct {
	fake    *archivetest.FakeRclone
	calls   [][]string
	budgets []time.Duration
	after   func([]string)
}

func (r *folderRecordingRunner) Run(ctx context.Context, args []string, limit int) ([]byte, error) {
	r.calls = append(r.calls, append([]string(nil), args...))
	deadline, ok := ctx.Deadline()
	if !ok {
		r.budgets = append(r.budgets, 0)
	} else {
		r.budgets = append(r.budgets, time.Until(deadline))
	}
	raw, err := r.fake.Run(ctx, args, limit)
	if r.after != nil {
		r.after(args)
	}
	return raw, err
}
func folderParent() ha.PinnedDrive {
	return ha.PinnedDrive{Remote: "fixture", Root: "fixture:", FolderID: "year-id"}
}
func TestRcloneResolveChildCachedEvidence(t *testing.T) {
	parent := folderParent()
	for _, tc := range []struct {
		name       string
		entries    []ha.RemoteEntry
		found, bad bool
	}{
		{name: "absent"},
		{name: "unique", entries: []ha.RemoteEntry{{Name: "10", ID: "month-id", IsDir: true}}, found: true},
		{name: "duplicates", entries: []ha.RemoteEntry{{Name: "10", ID: "a", IsDir: true}, {Name: "10", ID: "b", IsDir: true}}, bad: true},
		{name: "file", entries: []ha.RemoteEntry{{Name: "10", ID: "a"}}, bad: true},
		{name: "missing ID", entries: []ha.RemoteEntry{{Name: "10", IsDir: true}}, bad: true},
		{name: "control ID", entries: []ha.RemoteEntry{{Name: "10", ID: "bad\n", IsDir: true}}, bad: true},
		{name: "different name", entries: []ha.RemoteEntry{{Name: "11", ID: "a", IsDir: true}}, bad: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fake := &archivetest.FakeRclone{}
			r := ha.Rclone{Runner: fake}
			inv := map[string][]ha.RemoteEntry{"10": tc.entries}
			got, found, err := r.ResolveChild(context.Background(), parent, "10", inv)
			if found != tc.found || (err != nil) != tc.bad {
				t.Fatalf("got %v %v %v", got, found, err)
			}
			if found && got != (ha.PinnedDrive{Remote: "fixture", Root: "fixture:", FolderID: "month-id"}) {
				t.Fatalf("pin = %+v", got)
			}
			if tc.found || tc.bad {
				ensured, ensureErr := r.EnsureChild(context.Background(), parent, "10", inv)
				if (ensureErr != nil) != tc.bad || tc.found && ensured != got {
					t.Fatalf("cached ensure = %+v, %v", ensured, ensureErr)
				}
			}
			if fake.Invocations != 0 {
				t.Fatal("cached resolution performed IO")
			}
			if !reflect.DeepEqual(inv["10"], tc.entries) {
				t.Fatal("mutated inventory")
			}
		})
	}
}
func TestRcloneChildRejectsUnsafeInputWithoutIO(t *testing.T) {
	for _, name := range []string{"", ".", "..", "a/b", "a\\b", "a:b", "bad\n", strings.Repeat("a", 1025)} {
		t.Run(name, func(t *testing.T) {
			fake := &archivetest.FakeRclone{}
			r := ha.Rclone{Runner: fake}
			if _, _, err := r.ResolveChild(context.Background(), folderParent(), name, nil); err == nil {
				t.Fatal("accepted invalid name")
			}
			if _, err := r.EnsureChild(context.Background(), folderParent(), name, nil); err == nil {
				t.Fatal("created invalid name")
			}
			if fake.Invocations != 0 {
				t.Fatal("unsafe input performed IO")
			}
		})
	}
	for _, parent := range []ha.PinnedDrive{{}, {Remote: "fixture", Root: "fixture:"}, {Remote: "other", Root: "fixture:", FolderID: "year-id"}, {Remote: "fixture", Root: "fixture:untrusted", FolderID: "year-id"}} {
		fake := &archivetest.FakeRclone{}
		r := ha.Rclone{Runner: fake}
		if _, err := r.EnsureChild(context.Background(), parent, "10", nil); err == nil {
			t.Fatalf("accepted invalid parent %+v", parent)
		}
		if fake.Invocations != 0 {
			t.Fatal("invalid parent performed IO")
		}
	}
}
func TestRcloneEnsureChildConfirmPinnedCreation(t *testing.T) {
	fake := &archivetest.FakeRclone{FolderID: "root-id"}
	runner := &folderRecordingRunner{fake: fake}
	r := ha.Rclone{Runner: runner}
	inv := map[string][]ha.RemoteEntry{}
	got, err := r.EnsureChild(context.Background(), folderParent(), "10", inv)
	if err != nil || got.FolderID == "" || got.Root != "fixture:" {
		t.Fatalf("pin=%+v err=%v", got, err)
	}
	if fake.Mkdirs != 1 || fake.FolderLists["year-id"] != 1 || fake.FolderLists["root-id"] != 0 {
		t.Fatalf("mkdir=%d lists=%v", fake.Mkdirs, fake.FolderLists)
	}
	if len(inv) != 0 {
		t.Fatal("mutated cached inventory")
	}
	if len(runner.calls) != 2 || runner.calls[0][0] != "mkdir" || runner.calls[0][1] != "fixture:10" || runner.calls[1][0] != "lsjson" || runner.calls[1][1] != "fixture:" {
		t.Fatalf("calls=%v", runner.calls)
	}
	for i, args := range runner.calls {
		if !strings.Contains(strings.Join(args, " "), "--drive-root-folder-id year-id") {
			t.Fatalf("unbound parent: %v", args)
		}
		if runner.budgets[i] <= 0 || runner.budgets[i] > 60*time.Second {
			t.Fatalf("budget=%v", runner.budgets[i])
		}
	}
	observed, err := r.ObserveFlatInventory(context.Background(), folderParent())
	if err != nil {
		t.Fatal(err)
	}
	again, err := r.EnsureChild(context.Background(), folderParent(), "10", observed)
	if err != nil || got != again || fake.Mkdirs != 1 || len(runner.calls) != 3 {
		t.Fatalf("retry=%+v err=%v calls=%v", again, err, runner.calls)
	}
}
func TestRcloneEnsureChildReconcilesMkdirError(t *testing.T) {
	fake := &archivetest.FakeRclone{AfterMkdirError: errors.New("acknowledgment lost")}
	got, err := (ha.Rclone{Runner: fake}).EnsureChild(context.Background(), folderParent(), "10", nil)
	if err != nil || got.FolderID == "" || fake.Mkdirs != 1 || fake.Lists != 1 {
		t.Fatalf("pin=%+v err=%v mkdir=%d lists=%d", got, err, fake.Mkdirs, fake.Lists)
	}
}
func TestRcloneEnsureChildRequiresPositiveObservation(t *testing.T) {
	for _, mode := range []string{"absent", "duplicate", "file", "invalid ID", "read failure", "mkdir failure"} {
		t.Run(mode, func(t *testing.T) {
			fake := &archivetest.FakeRclone{}
			fake.BeforeFolder = func(_ context.Context, op, id string) error {
				if mode == "mkdir failure" && op == "mkdir" {
					return errors.New("unavailable")
				}
				if op != "lsjson" {
					return nil
				}
				switch mode {
				case "absent", "mkdir failure":
					delete(fake.ChildFolders[id], "10")
				case "duplicate":
					fake.ChildFolders[id]["10"] = []archivetest.FakeRemoteObject{{ID: "a", IsDir: true}, {ID: "b", IsDir: true}}
				case "file":
					fake.ChildFolders[id]["10"] = []archivetest.FakeRemoteObject{{ID: "a"}}
				case "invalid ID":
					fake.ChildFolders[id]["10"] = []archivetest.FakeRemoteObject{{IsDir: true}}
				case "read failure":
					return errors.New("read failed")
				}
				return nil
			}
			got, err := (ha.Rclone{Runner: fake}).EnsureChild(context.Background(), folderParent(), "10", nil)
			if err == nil || got != (ha.PinnedDrive{}) || fake.Mkdirs > 1 {
				t.Fatalf("pin=%+v err=%v mkdir=%d", got, err, fake.Mkdirs)
			}
			if fake.Invocations != 2 {
				t.Fatalf("calls=%d; need exactly one mkdir attempt and confirmation", fake.Invocations)
			}
		})
	}
}
func TestRcloneEnsureChildCancellationStopsSubsequentIO(t *testing.T) {
	for _, when := range []string{"before", "mkdir", "observe"} {
		t.Run(when, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			fake := &archivetest.FakeRclone{}
			if when == "before" {
				cancel()
			} else {
				fake.BeforeFolder = func(_ context.Context, op, id string) error {
					if when == "mkdir" && op == "mkdir" || when == "observe" && op == "lsjson" {
						cancel()
					}
					return nil
				}
			}
			got, err := (ha.Rclone{Runner: fake}).EnsureChild(ctx, folderParent(), "10", nil)
			if !errors.Is(err, context.Canceled) || got != (ha.PinnedDrive{}) {
				t.Fatalf("pin=%+v err=%v", got, err)
			}
			want := map[string]int{"before": 0, "mkdir": 1, "observe": 2}[when]
			if fake.Invocations != want {
				t.Fatalf("calls=%d want=%d", fake.Invocations, want)
			}
		})
	}
}
func TestRcloneResolveChildLocalIdentityFallback(t *testing.T) {
	r := ha.Rclone{LocalConformance: true}
	parent := ha.PinnedDrive{Remote: "fixture", Root: "fixture:/private/test/archive"}
	got, found, err := r.ResolveChild(context.Background(), parent, "10", map[string][]ha.RemoteEntry{"10": {{Name: "10", IsDir: true}}})
	want := ha.PinnedDrive{Remote: "fixture", Root: parent.Root + "/10", FolderID: parent.Root + "/10"}
	if err != nil || !found || got != want {
		t.Fatalf("pin=%+v found=%v err=%v", got, found, err)
	}
}

func TestRcloneEnsureChildCancellationAfterRemoteCreation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	fake := &archivetest.FakeRclone{}
	runner := &folderRecordingRunner{fake: fake, after: func(args []string) {
		if args[0] == "mkdir" {
			cancel()
		}
	}}
	got, err := (ha.Rclone{Runner: runner}).EnsureChild(ctx, folderParent(), "10", nil)
	if !errors.Is(err, context.Canceled) || got != (ha.PinnedDrive{}) || fake.Mkdirs != 1 || fake.Lists != 0 || len(runner.calls) != 1 {
		t.Fatalf("pin=%+v err=%v mkdir=%d lists=%d calls=%v", got, err, fake.Mkdirs, fake.Lists, runner.calls)
	}
	if len(fake.ChildFolders["year-id"]["10"]) != 1 {
		t.Fatal("cancellation repaired or deleted retained folder")
	}
}
