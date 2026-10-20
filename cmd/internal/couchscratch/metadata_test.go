package couchscratch

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func metadataFixture(t *testing.T) (MetadataOptions, string, string) {
	t.Helper()
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	cache := filepath.Join(root, "home", "Library", "Caches", "org.Zellij-Contributors.Zellij", "contract_version_1")
	if err := os.MkdirAll(filepath.Join(cache, "session_info", "owned"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cache, "session_info", "owned", "session-layout.kdl"), []byte("original layout\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(root, "ram"), 0700); err != nil {
		t.Fatal(err)
	}
	return MetadataOptions{CacheDir: cache, Root: filepath.Join(root, "ram"), UseRAM: true, ServerPresence: func(context.Context) (bool, error) { return false, nil }}, filepath.Join(cache, "session_info"), filepath.Join(cache, "session_info.disk")
}

func TestMetadataNoServersRoutesToRAMAndKeepsBackup(t *testing.T) {
	opts, route, backup := metadataFixture(t)
	before, err := os.Stat(filepath.Join(route, "owned", "session-layout.kdl"))
	if err != nil {
		t.Fatal(err)
	}
	if err := ConfigureMetadata(t.Context(), opts); err != nil {
		t.Fatal(err)
	}
	link, err := os.Readlink(route)
	if err != nil || link != filepath.Join(opts.Root, "zellij-session-info") {
		t.Fatalf("route=%q, %v", link, err)
	}
	after, err := os.Stat(filepath.Join(backup, "owned", "session-layout.kdl"))
	if err != nil || !os.SameFile(before, after) {
		t.Fatalf("original layout inode lost: %v", err)
	}
	if err := ConfigureMetadata(t.Context(), opts); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(backup, "owned", "session-layout.kdl"))
	if err != nil || string(data) != "original layout\n" {
		t.Fatalf("backup changed: %q %v", data, err)
	}
}

func TestMetadataLiveServerLeavesEveryPathAlone(t *testing.T) {
	opts, route, backup := metadataFixture(t)
	opts.Root = filepath.Join(filepath.Dir(opts.Root), "absent-ram")
	before, _ := os.Stat(route)
	var notes []string
	opts.Notify = func(note string) { notes = append(notes, note) }
	opts.ServerPresence = func(context.Context) (bool, error) { return true, nil }
	if err := ConfigureMetadata(t.Context(), opts); err != nil {
		t.Fatal(err)
	}
	after, err := os.Stat(route)
	if err != nil || !os.SameFile(before, after) {
		t.Fatalf("live route changed: %v", err)
	}
	if _, err := os.Lstat(backup); !os.IsNotExist(err) {
		t.Fatalf("backup created for live server: %v", err)
	}
	if _, err := os.Lstat(opts.Root); !os.IsNotExist(err) {
		t.Fatalf("RAM root created for live server: %v", err)
	}
	if !reflect.DeepEqual(notes, []string{"Zellij metadata stays on disk until the next reboot"}) {
		t.Fatalf("notes=%q", notes)
	}
}

func TestMetadataAbsentRAMRestoresDirectoryWithoutBackup(t *testing.T) {
	opts, route, backup := metadataFixture(t)
	if err := ConfigureMetadata(t.Context(), opts); err != nil {
		t.Fatal(err)
	}
	before, _ := os.Stat(backup)
	opts.UseRAM = false
	if err := ConfigureMetadata(t.Context(), opts); err != nil {
		t.Fatal(err)
	}
	info, err := os.Lstat(route)
	if err != nil || !info.IsDir() {
		t.Fatalf("real route not restored: %v", err)
	}
	entries, err := os.ReadDir(route)
	if err != nil || len(entries) != 0 {
		t.Fatalf("old cache restored instead of empty directory: %v %v", entries, err)
	}
	after, _ := os.Stat(backup)
	if !os.SameFile(before, after) {
		t.Fatal("backup replaced")
	}
	if err := ConfigureMetadata(t.Context(), opts); err != nil {
		t.Fatal(err)
	}
}

func TestMetadataUnknownServerEvidenceDoesNotMutate(t *testing.T) {
	opts, route, backup := metadataFixture(t)
	want := errors.New("native snapshot unavailable")
	opts.ServerPresence = func(context.Context) (bool, error) { return false, want }
	before, _ := os.Stat(route)
	if err := ConfigureMetadata(t.Context(), opts); !errors.Is(err, want) {
		t.Fatalf("error=%v", err)
	}
	after, _ := os.Stat(route)
	if !os.SameFile(before, after) {
		t.Fatal("unknown observation changed route")
	}
	if _, err := os.Lstat(backup); !os.IsNotExist(err) {
		t.Fatalf("backup created: %v", err)
	}
}

func TestMetadataForeignLinkAndBackupConflictArePreserved(t *testing.T) {
	t.Run("foreign link", func(t *testing.T) {
		opts, route, _ := metadataFixture(t)
		foreign := filepath.Join(t.TempDir(), "foreign")
		if err := os.Rename(route, foreign); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(foreign, route); err != nil {
			t.Fatal(err)
		}
		if err := ConfigureMetadata(t.Context(), opts); err == nil {
			t.Fatal("foreign link accepted")
		}
		if got, _ := os.Readlink(route); got != foreign {
			t.Fatal("foreign link changed")
		}
	})
	t.Run("backup conflict", func(t *testing.T) {
		opts, route, backup := metadataFixture(t)
		if err := os.Mkdir(backup, 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(backup, "sentinel"), []byte("keep"), 0600); err != nil {
			t.Fatal(err)
		}
		if err := ConfigureMetadata(t.Context(), opts); err == nil {
			t.Fatal("backup conflict overwritten")
		}
		if info, err := os.Lstat(route); err != nil || !info.IsDir() {
			t.Fatal("route changed")
		}
		if data, err := os.ReadFile(filepath.Join(backup, "sentinel")); err != nil || string(data) != "keep" {
			t.Fatal("backup changed")
		}
	})
}

func TestMetadataDirectoryMatchesNativePlatformRoots(t *testing.T) {
	for _, tc := range []struct{ platform, cache, want string }{
		{"darwin", "/ignored", "/fixture/home/Library/Caches/org.Zellij-Contributors.Zellij/contract_version_1"},
		{"linux", "", "/fixture/home/.cache/zellij/contract_version_1"},
		{"linux", "/fixture/cache", "/fixture/cache/zellij/contract_version_1"},
	} {
		got, err := MetadataDirectory("/fixture/home", tc.cache, tc.platform)
		if err != nil || got != tc.want {
			t.Fatalf("%s: %q %v", tc.platform, got, err)
		}
	}
}

func TestMetadataDecisions(t *testing.T) {
	for _, tc := range []struct {
		name  string
		state MetadataState
		want  MetadataAction
	}{
		{"live", MetadataState{Live: true, UseRAM: true, Route: MetadataDirectoryRoute}, MetadataDefer},
		{"unknown", MetadataState{Unknown: true, UseRAM: true}, MetadataRefuse},
		{"foreign", MetadataState{UseRAM: true, Route: MetadataForeignRoute}, MetadataRefuse},
		{"backup", MetadataState{UseRAM: true, Route: MetadataDirectoryRoute}, MetadataBackupAndLink},
		{"retry", MetadataState{UseRAM: true, Backup: true}, MetadataCreateLink},
		{"conflict", MetadataState{UseRAM: true, Backup: true, Route: MetadataDirectoryRoute}, MetadataRefuse},
		{"empty", MetadataState{UseRAM: true, Backup: true, Empty: true, Route: MetadataDirectoryRoute}, MetadataReplaceEmptyWithLink},
		{"restore", MetadataState{Route: MetadataOwnedLink}, MetadataCreateDirectory},
		{"missing disk", MetadataState{}, MetadataCreateDirectory},
		{"already linked", MetadataState{UseRAM: true, Route: MetadataOwnedLink}, MetadataKeep},
		{"already disk", MetadataState{Route: MetadataDirectoryRoute}, MetadataKeep},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := DecideMetadata(tc.state); got != tc.want {
				t.Fatalf("got %v want %v", got, tc.want)
			}
		})
	}
}

func TestMetadataUnsafeServerStatesNeverMutate(t *testing.T) {
	for route := MetadataMissing; route <= MetadataForeignRoute; route++ {
		for _, ram := range []bool{false, true} {
			for _, backup := range []bool{false, true} {
				for _, empty := range []bool{false, true} {
					for _, unknown := range []bool{false, true} {
						s := MetadataState{Route: route, UseRAM: ram, Backup: backup, Empty: empty, Live: !unknown, Unknown: unknown}
						a := DecideMetadata(s)
						if a != MetadataDefer && a != MetadataRefuse {
							t.Fatalf("unsafe state %+v emitted %v", s, a)
						}
					}
				}
			}
		}
	}
}

func TestMetadataInterruptedRenameAndDiskFallbackCanReadoptRAM(t *testing.T) {
	opts, route, backup := metadataFixture(t)
	if err := os.Rename(route, backup); err != nil {
		t.Fatal(err)
	}
	if err := ConfigureMetadata(t.Context(), opts); err != nil {
		t.Fatal(err)
	}
	opts.UseRAM = false
	if err := ConfigureMetadata(t.Context(), opts); err != nil {
		t.Fatal(err)
	}
	opts.UseRAM = true
	if err := ConfigureMetadata(t.Context(), opts); err != nil {
		t.Fatal(err)
	}
	if got, err := os.Readlink(route); err != nil || got != filepath.Join(opts.Root, "zellij-session-info") {
		t.Fatalf("route=%q err=%v", got, err)
	}
}

func TestMetadataRefusesUnsafeDestinationsBeforeMovingRoute(t *testing.T) {
	for _, kind := range []string{"ram link", "cache link", "nil census", "canceled"} {
		t.Run(kind, func(t *testing.T) {
			opts, route, backup := metadataFixture(t)
			ctx := t.Context()
			switch kind {
			case "ram link":
				if err := os.MkdirAll(opts.Root, 0700); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(t.TempDir(), filepath.Join(opts.Root, "zellij-session-info")); err != nil {
					t.Fatal(err)
				}
			case "cache link":
				actual := opts.CacheDir + ".actual"
				if err := os.Rename(opts.CacheDir, actual); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(actual, opts.CacheDir); err != nil {
					t.Fatal(err)
				}
			case "nil census":
				opts.ServerPresence = nil
			case "canceled":
				var cancel context.CancelFunc
				ctx, cancel = context.WithCancel(ctx)
				cancel()
			}
			if err := ConfigureMetadata(ctx, opts); err == nil {
				t.Fatal("unsafe setup accepted")
			}
			if st, err := os.Lstat(route); err != nil || !st.IsDir() {
				t.Fatalf("route changed: %v", err)
			}
			if _, err := os.Lstat(backup); !os.IsNotExist(err) {
				t.Fatalf("backup unexpectedly created: %v", err)
			}
		})
	}
}

func TestMetadataMissingRAMRootIsNeverCreated(t *testing.T) {
	opts, route, backup := metadataFixture(t)
	// Admission can have observed RAM before a mount disappears. Configuration
	// must treat this observation as stale rather than recreating its root.
	opts.Root = filepath.Join(filepath.Dir(opts.Root), "missing-volume")
	before, err := os.Stat(route)
	if err != nil {
		t.Fatal(err)
	}
	if err := ConfigureMetadata(t.Context(), opts); err == nil {
		t.Fatal("missing RAM root accepted")
	}
	if _, err := os.Lstat(opts.Root); !os.IsNotExist(err) {
		t.Fatalf("RAM root recreated: %v", err)
	}
	after, err := os.Stat(route)
	if err != nil || !os.SameFile(before, after) {
		t.Fatalf("route changed: %v", err)
	}
	if _, err := os.Lstat(backup); !os.IsNotExist(err) {
		t.Fatalf("backup unexpectedly created: %v", err)
	}
	opts.CacheDir = filepath.Join(filepath.Dir(opts.CacheDir), "missing-cache")
	if err := ConfigureMetadata(t.Context(), opts); err == nil {
		t.Fatal("missing RAM root accepted for new cache")
	}
	if _, err := os.Lstat(opts.CacheDir); !os.IsNotExist(err) {
		t.Fatalf("cache created despite missing RAM root: %v", err)
	}
}
