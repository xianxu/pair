package archivecmd

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/historyarchive"
)

func restoreCLIConfig(t *testing.T) historyarchive.ContinuousConfig {
	t.Helper()
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("PAIR_DATA_DIR", root)
	store := filepath.Join(root, "archive")
	if err = os.Mkdir(store, 0700); err != nil {
		t.Fatal(err)
	}
	cfg := historyarchive.ContinuousConfig{Version: 1, Recipients: []string{strings.Repeat("A", 40)}, Sink: filepath.Join(root, "sink"), Catalog: filepath.Join(store, "history.jsonl")}
	raw, err := json.Marshal(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(filepath.Join(store, historyarchive.ContinuousConfigName), raw, 0600); err != nil {
		t.Fatal(err)
	}
	return cfg
}
func setRestoreCLI(t *testing.T, fn func(context.Context, historyarchive.Restorer, historyarchive.RestoreRequest) (historyarchive.RestoreReport, error)) {
	t.Helper()
	old := RestoreFunc
	RestoreFunc = fn
	t.Cleanup(func() { RestoreFunc = old })
}

func TestRestoreCLIHelpIsReadOnly(t *testing.T) {
	root := t.TempDir()
	t.Setenv("PAIR_DATA_DIR", filepath.Join(root, "absent"))
	calls := 0
	setRestoreCLI(t, func(context.Context, historyarchive.Restorer, historyarchive.RestoreRequest) (historyarchive.RestoreReport, error) {
		calls++
		return historyarchive.RestoreReport{}, nil
	})
	for _, args := range [][]string{{"restore", "--help"}, {"restore", "-h"}, {"--help"}} {
		var out, diag bytes.Buffer
		if code := RunContext(context.Background(), args, &out, &diag, nil); code != 0 || !strings.Contains(out.String(), "restore") || !strings.Contains(out.String(), "--to") {
			t.Fatalf("%v: %d %q %q", args, code, out.String(), diag.String())
		}
	}
	if calls != 0 {
		t.Fatal("help attempted restore")
	}
	entries, err := os.ReadDir(root)
	if err != nil || len(entries) != 0 {
		t.Fatalf("help created storage: %v %v", entries, err)
	}
}
func TestRestoreCLIParsesRepeatedSelectionsAndProductionWiring(t *testing.T) {
	cfg := restoreCLIConfig(t)
	for _, source := range []string{"", "auto", "local", "drive"} {
		t.Run(source, func(t *testing.T) {
			idA, idB := strings.Repeat("a", 64), strings.Repeat("b", 64)
			target := filepath.Join(t.TempDir(), "fresh target")
			args := []string{"restore", "--id", idA, "--path", "literal/*.md", "--id", idB, "--to", target, "--path", "Unicode/你好.md"}
			if source != "" {
				args = append(args, "--from", source)
			}
			calls := 0
			setRestoreCLI(t, func(ctx context.Context, r historyarchive.Restorer, req historyarchive.RestoreRequest) (historyarchive.RestoreReport, error) {
				calls++
				if ctx.Err() != nil {
					t.Fatal(ctx.Err())
				}
				if !reflect.DeepEqual(r.Config, cfg) {
					t.Fatalf("configuration lost: %+v", r.Config)
				}
				if _, ok := r.Decoder.(historyarchive.CommandDecoder); !ok {
					t.Fatalf("production decoder not wired: %T", r.Decoder)
				}
				wantFrom := source
				if wantFrom == "" {
					wantFrom = "auto"
				}
				want := historyarchive.RestoreRequest{IDs: []string{idA, idB}, To: target, From: wantFrom, Paths: []string{"literal/*.md", "Unicode/你好.md"}}
				if !reflect.DeepEqual(req, want) {
					t.Fatalf("request %+v want %+v", req, want)
				}
				return historyarchive.RestoreReport{Bundles: 2, Files: 3, Target: target, State: historyarchive.RestoreState{Phase: historyarchive.RestorePublished}}, nil
			})
			var out, diag bytes.Buffer
			if code := RunContext(context.Background(), args, &out, &diag, nil); code != 0 || calls != 1 || !strings.Contains(out.String(), "2 bundle(s)") || !strings.Contains(out.String(), "3 file(s)") || strings.Contains(out.String(), "literal/") {
				t.Fatalf("restore: %d calls%d %q %q", code, calls, out.String(), diag.String())
			}
		})
	}
}
func TestRestoreCLIRejectsArgumentsBeforeConfiguration(t *testing.T) {
	t.Setenv("PAIR_DATA_DIR", t.TempDir())
	id := strings.Repeat("a", 64)
	calls := 0
	setRestoreCLI(t, func(context.Context, historyarchive.Restorer, historyarchive.RestoreRequest) (historyarchive.RestoreReport, error) {
		calls++
		return historyarchive.RestoreReport{}, nil
	})
	for _, args := range [][]string{{"restore"}, {"restore", "--id", id}, {"restore", "--to", "/fresh"}, {"restore", "--id"}, {"restore", "--id", "", "--to", "/fresh"}, {"restore", "--id", id, "--to", "/fresh", "--to", "/other"}, {"restore", "--id", id, "--to", "/fresh", "--from", "other"}, {"restore", "--id", id, "--to", "/fresh", "--from", "local", "--from", "drive"}, {"restore", id, "--to", "/fresh"}, {"restore", "--help", "extra"}, {"restore", "--id", id, "--to", "/fresh", "--path", ""}, {"restore", "--id", id, "--to", "/fresh", "--bogus", "x"}} {
		var out, diag bytes.Buffer
		if code := RunContext(context.Background(), args, &out, &diag, nil); code != 1 || !strings.Contains(diag.String(), "arguments") || out.Len() != 0 {
			t.Fatalf("%v: code%d out%q diag%q", args, code, out.String(), diag.String())
		}
	}
	if calls != 0 {
		t.Fatal("invalid arguments attempted restore")
	}
}
func TestRestoreCLIMissingConfigurationAndCancellation(t *testing.T) {
	t.Setenv("PAIR_DATA_DIR", t.TempDir())
	calls := 0
	setRestoreCLI(t, func(context.Context, historyarchive.Restorer, historyarchive.RestoreRequest) (historyarchive.RestoreReport, error) {
		calls++
		return historyarchive.RestoreReport{}, nil
	})
	args := []string{"restore", "--id", strings.Repeat("a", 64), "--to", "/fresh"}
	var out, diag bytes.Buffer
	if code := RunContext(context.Background(), args, &out, &diag, nil); code != 1 || !strings.Contains(diag.String(), "not configured") {
		t.Fatalf("missing config: %d %q", code, diag.String())
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	diag.Reset()
	if code := RunContext(ctx, args, &out, &diag, nil); code != 1 || !strings.Contains(diag.String(), "canceled") {
		t.Fatalf("canceled: %d %q", code, diag.String())
	}
	if calls != 0 {
		t.Fatal("missing config/canceled operation attempted restore")
	}
}
func TestRestoreCLIPublicationAndOutputFailuresAreHonest(t *testing.T) {
	restoreCLIConfig(t)
	for _, tc := range []struct {
		name  string
		phase historyarchive.RestorePhase
	}{{"refused", historyarchive.RestoreRefused}, {"uncertain", historyarchive.RestorePublicationUncertain}, {"committed-cleanup", historyarchive.RestorePublished}} {
		phase := tc.phase
		t.Run(tc.name, func(t *testing.T) {
			setRestoreCLI(t, func(context.Context, historyarchive.Restorer, historyarchive.RestoreRequest) (historyarchive.RestoreReport, error) {
				return historyarchive.RestoreReport{State: historyarchive.RestoreState{Phase: phase}}, errors.New("failure\ninspect safely")
			})
			var out, diag bytes.Buffer
			code := RunContext(context.Background(), []string{"restore", "--id", strings.Repeat("a", 64), "--to", "/fresh"}, &out, &diag, nil)
			if code != 1 || out.Len() != 0 || strings.Count(diag.String(), "\n") != 1 {
				t.Fatalf("failure: %d %q %q", code, out.String(), diag.String())
			}
			if phase == historyarchive.RestorePublicationUncertain && !strings.Contains(diag.String(), "may exist") {
				t.Fatal("uncertainty hidden: ", diag.String())
			}
			if phase == historyarchive.RestorePublished && !strings.Contains(diag.String(), "published") {
				t.Fatal("committed output hidden: ", diag.String())
			}
		})
	}
	setRestoreCLI(t, func(context.Context, historyarchive.Restorer, historyarchive.RestoreRequest) (historyarchive.RestoreReport, error) {
		return historyarchive.RestoreReport{Bundles: 1, Files: 1, Target: "/fresh", State: historyarchive.RestoreState{Phase: historyarchive.RestorePublished}}, nil
	})
	var diag bytes.Buffer
	if code := RunContext(context.Background(), []string{"restore", "--id", strings.Repeat("a", 64), "--to", "/fresh"}, failedOutput{}, &diag, nil); code != 1 || !strings.Contains(diag.String(), "published") {
		t.Fatalf("output failure: %d %q", code, diag.String())
	}
}

func TestRestoreCLIUsesRestoreModelValidationWithoutTools(t *testing.T) {
	cfg := restoreCLIConfig(t)
	for _, tc := range []struct {
		args       []string
		diagnostic string
	}{
		{[]string{"restore", "--id", strings.Repeat("a", 64), "--to", "relative"}, "absolute"},
		{[]string{"restore", "--id", "short", "--to", "/fresh"}, "64-hex"},
		{[]string{"restore", "--id", strings.Repeat("a", 64), "--to", "/fresh", "--path", "../escape"}, "literal restore path"},
		{[]string{"restore", "--id", strings.Repeat("a", 64), "--to", "/fresh"}, "find or sync-index"},
	} {
		var out, diag bytes.Buffer
		if code := RunContext(context.Background(), tc.args, &out, &diag, nil); code != 1 || out.Len() != 0 || !strings.Contains(diag.String(), tc.diagnostic) {
			t.Fatalf("%v: %d %q %q", tc.args, code, out.String(), diag.String())
		}
	}
	entries, err := os.ReadDir(filepath.Dir(cfg.Catalog))
	if err != nil || len(entries) != 1 {
		t.Fatalf("validation wrote archive metadata: %v %v", entries, err)
	}
	if _, err = os.Stat(cfg.Sink); !os.IsNotExist(err) {
		t.Fatal("validation created sink", err)
	}
}

func TestRestoreCLISanitizesUntrustedDiagnostics(t *testing.T) {
	restoreCLIConfig(t)
	setRestoreCLI(t, func(context.Context, historyarchive.Restorer, historyarchive.RestoreRequest) (historyarchive.RestoreReport, error) {
		return historyarchive.RestoreReport{}, errors.New("checksum failed\nprintable 雪 \x1b]52;c;ZmFrZQ==\a \x0e\x7f\u009b")
	})
	var out, diag bytes.Buffer
	code := RunContext(context.Background(), []string{"restore", "--id", strings.Repeat("a", 64), "--to", "/private/scratch/fresh"}, &out, &diag, nil)
	if code == 0 || out.Len() != 0 {
		t.Fatal("unsafe diagnostic became success", code, out.String())
	}
	if strings.ContainsAny(diag.String(), "\x1b\a\x0e\x7f\u009b") || strings.Count(diag.String(), "\n") != 1 {
		t.Fatalf("raw terminal control escaped CLI boundary: %q", diag.String())
	}
	if !strings.Contains(diag.String(), "checksum failed printable 雪") {
		t.Fatalf("readable failure context lost: %q", diag.String())
	}
}
