package testpair

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xianxu/pair/internal/reporoot"
)

func TestMain(m *testing.M) {
	os.Exit(Run(m))
}

// Each child gets a new package lifetime, including the fallback cache and Run's
// cleanup. The parent can inspect the candidate after that lifetime has ended.
func TestBinaryProcessHelper(t *testing.T) {
	mode := os.Getenv("PAIR_TEST_BINARY_HELPER_MODE")
	if mode == "" {
		t.Skip("subprocess-only helper")
	}
	if strings.HasPrefix(mode, "cancel-") {
		if strings.Contains(mode, "cached") {
			Binary(t)
		}
		ctx, cancel := context.WithCancel(t.Context())
		cancel()
		if strings.Contains(mode, "at") {
			AtContext(t, ctx, os.Getenv("PAIR_TEST_BINARY_ALIAS"))
		} else {
			BinaryContext(t, ctx)
		}
		t.Fatal("cancelled context was accepted")
	}
	var candidate string
	t.Run("first caller", func(t *testing.T) {
		candidate = Binary(t)
	})
	t.Run("later caller after first cleanup", func(t *testing.T) {
		if next := Binary(t); next != candidate {
			t.Fatalf("candidate changed across callers: %q then %q", candidate, next)
		}
		if _, err := os.Stat(candidate); err != nil {
			t.Fatalf("first caller cleanup removed shared candidate: %v", err)
		}
	})
	if t.Failed() {
		return
	}
	alias := os.Getenv("PAIR_TEST_BINARY_ALIAS")
	if got := At(t, alias); got != alias {
		t.Fatalf("At returned %q, want destination %q", got, alias)
	}
	sourceInfo, err := os.Stat(candidate)
	if err != nil {
		t.Fatal(err)
	}
	aliasInfo, err := os.Stat(alias)
	if err != nil {
		t.Fatal(err)
	}
	if !os.SameFile(sourceInfo, aliasInfo) {
		t.Fatal("same-filesystem alias must hard-link the candidate")
	}
	if aliasInfo.Mode()&0111 == 0 {
		t.Fatal("placed alias is not executable")
	}
	out, err := exec.Command(alias).CombinedOutput()
	if err != nil || string(out) != "candidate fixture\n" {
		t.Fatalf("placed executable: %v, output %q", err, out)
	}
	if err := os.WriteFile(os.Getenv("PAIR_TEST_BINARY_OBSERVED"), []byte(candidate), 0600); err != nil {
		t.Fatal(err)
	}
}

type binaryFixture struct {
	dir, calls, alias, observed string
	env                         []string
}

func newBinaryFixture(t *testing.T) binaryFixture {
	t.Helper()
	dir := t.TempDir()
	fakeBin := filepath.Join(dir, "bin")
	if err := os.Mkdir(fakeBin, 0700); err != nil {
		t.Fatal(err)
	}
	// This replaces only the compiler executable. The real central builder still
	// parses its arguments, chooses output paths and places the candidate.
	fakeGo := `#!/bin/sh
set -eu
[ "$1" = build ] || { echo "unexpected fake go command: $*" >&2; exit 91; }
shift
output=
while [ "$#" -gt 0 ]; do
  case "$1" in
    -o) output=$2; shift 2 ;;
    *) shift ;;
  esac
done
[ -n "$output" ] || exit 92
printf 'build\n' >> "$PAIR_TEST_FAKE_GO_CALLS"
cat > "$output" <<'CANDIDATE'
#!/bin/sh
printf 'candidate fixture\n'
CANDIDATE
chmod +x "$output"
`
	if err := os.WriteFile(filepath.Join(fakeBin, "go"), []byte(fakeGo), 0700); err != nil {
		t.Fatal(err)
	}
	fixture := binaryFixture{dir: dir, calls: filepath.Join(dir, "calls"), alias: filepath.Join(dir, "alias"), observed: filepath.Join(dir, "observed")}
	for _, entry := range os.Environ() {
		key, _, _ := strings.Cut(entry, "=")
		if key != "PAIR_TEST_BINARY" && key != "PAIR_TEST_BINARY_HELPER_MODE" && key != "PAIR_TEST_FAKE_GO_CALLS" && key != "PAIR_TEST_BINARY_ALIAS" && key != "PAIR_TEST_BINARY_OBSERVED" && key != "PATH" && key != "TMPDIR" {
			fixture.env = append(fixture.env, entry)
		}
	}
	fixture.env = append(fixture.env,
		"PATH="+fakeBin+string(os.PathListSeparator)+os.Getenv("PATH"),
		"TMPDIR="+dir,
		"PAIR_TEST_FAKE_GO_CALLS="+fixture.calls,
		"PAIR_TEST_BINARY_ALIAS="+fixture.alias,
		"PAIR_TEST_BINARY_OBSERVED="+fixture.observed,
	)
	return fixture
}

func (f binaryFixture) run(t *testing.T, mode, supplied string, wantFailure bool) string {
	t.Helper()
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(executable, "-test.run=^TestBinaryProcessHelper$", "-test.v")
	cmd.Env = append(append([]string(nil), f.env...), "PAIR_TEST_BINARY_HELPER_MODE="+mode, "PAIR_TEST_BINARY="+supplied)
	output, err := cmd.CombinedOutput()
	if (err != nil) != wantFailure {
		t.Fatalf("helper mode %q: error=%v, wantFailure=%t\n%s", mode, err, wantFailure, output)
	}
	return string(output)
}

func (f binaryFixture) buildCount(t *testing.T) int {
	t.Helper()
	raw, err := os.ReadFile(f.calls)
	if os.IsNotExist(err) {
		return 0
	}
	if err != nil {
		t.Fatal(err)
	}
	return strings.Count(string(raw), "build\n")
}

func TestBinaryStandaloneBuildsOnceAndRunCleansFallback(t *testing.T) {
	f := newBinaryFixture(t)
	f.run(t, "fallback", "", false)
	if got := f.buildCount(t); got != 1 {
		t.Fatalf("compiler invocations = %d, want 1 across Binary and At callers", got)
	}
	raw, err := os.ReadFile(f.observed)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(string(raw)); !os.IsNotExist(err) {
		t.Fatalf("owned fallback survived Run cleanup: %q: %v", raw, err)
	}
	if _, err := os.Stat(f.alias); err != nil {
		t.Fatalf("Run removed caller-owned alias: %v", err)
	}
}

func TestBinarySuppliedCandidateIsSharedAndPreserved(t *testing.T) {
	f := newBinaryFixture(t)
	candidate := filepath.Join(f.dir, "supplied")
	body := []byte("#!/bin/sh\nprintf 'candidate fixture\\n'\n")
	if err := os.WriteFile(candidate, body, 0700); err != nil {
		t.Fatal(err)
	}
	f.run(t, "supplied", candidate, false)
	if got := f.buildCount(t); got != 0 {
		t.Fatalf("supplied candidate triggered %d compiler invocations", got)
	}
	if raw, err := os.ReadFile(candidate); err != nil || string(raw) != string(body) {
		t.Fatalf("Run changed supplied candidate: %v, %q", err, raw)
	}
}

func TestBinaryRejectsInvalidSuppliedCandidateWithoutBuilding(t *testing.T) {
	for _, kind := range []string{"missing", "directory", "not executable"} {
		t.Run(kind, func(t *testing.T) {
			f := newBinaryFixture(t)
			candidate := filepath.Join(f.dir, "supplied")
			switch kind {
			case "directory":
				if err := os.Mkdir(candidate, 0700); err != nil {
					t.Fatal(err)
				}
			case "not executable":
				if err := os.WriteFile(candidate, []byte("fixture"), 0600); err != nil {
					t.Fatal(err)
				}
			}
			output := f.run(t, "invalid", candidate, true)
			if !strings.Contains(output, "PAIR_TEST_BINARY") {
				t.Fatalf("missing supplied-candidate diagnostic:\n%s", output)
			}
			if got := f.buildCount(t); got != 0 {
				t.Fatalf("invalid supplied candidate silently rebuilt %d times", got)
			}
		})
	}
}

func TestBinaryContextRefusesCancellationBeforeBuildOrPlacement(t *testing.T) {
	for _, mode := range []string{"cancel-binary", "cancel-at", "cancel-cached-binary", "cancel-cached-at"} {
		for _, supplied := range []bool{false, true} {
			name := mode + "/fallback"
			if supplied {
				name = mode + "/supplied"
			}
			t.Run(name, func(t *testing.T) {
				f := newBinaryFixture(t)
				candidate := ""
				if supplied {
					candidate = filepath.Join(f.dir, "supplied")
					if err := os.WriteFile(candidate, []byte("#!/bin/sh\nexit 0\n"), 0700); err != nil {
						t.Fatal(err)
					}
				}
				output := f.run(t, mode, candidate, true)
				if !strings.Contains(strings.ToLower(output), "context canceled") {
					t.Fatalf("missing context cancellation diagnostic:\n%s", output)
				}
				wantBuilds := 0
				if !supplied && strings.Contains(mode, "cached") {
					wantBuilds = 1
				}
				if got := f.buildCount(t); got != wantBuilds {
					t.Fatalf("builds=%d, want %d", got, wantBuilds)
				}
				if _, err := os.Stat(f.alias); !os.IsNotExist(err) {
					t.Fatalf("cancelled call placed an alias: %v", err)
				}
			})
		}
	}
}

func TestSharedBinaryShellContract(t *testing.T) {
	cmd := exec.Command("bash", "../../../tests/lib/pair-test-binary-test.sh")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("shell lifecycle: %v\n%s", err, out)
	}
}

func TestLegacyPlacementNormalizesDestinationAtCheckoutBoundary(t *testing.T) {
	original, err := reporoot.Current()
	if err != nil {
		t.Fatal(err)
	}
	builder, err := os.ReadFile(filepath.Join(original, "tests", "lib", "pair-test-binary.sh"))
	if err != nil {
		t.Fatal(err)
	}
	for _, form := range []string{"relative", "absolute-unclean"} {
		for _, withContext := range []bool{false, true} {
			name := "At"
			if withContext {
				name = "AtContext"
			}
			t.Run(form+"/"+name, func(t *testing.T) {
				checkout := t.TempDir()
				for _, dir := range []string{"tests/lib", "nested/caller", "aliases"} {
					if err := os.MkdirAll(filepath.Join(checkout, dir), 0700); err != nil {
						t.Fatal(err)
					}
				}
				if err := os.WriteFile(filepath.Join(checkout, "go.mod"), []byte("module github.com/xianxu/pair\n"), 0600); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(checkout, "tests", "lib", "pair-test-binary.sh"), builder, 0600); err != nil {
					t.Fatal(err)
				}
				source := filepath.Join(t.TempDir(), "suite-pair")
				body := []byte("#!/bin/sh\nexit 0\n")
				if err := os.WriteFile(source, body, 0700); err != nil {
					t.Fatal(err)
				}
				t.Setenv("PAIR_TEST_BINARY", source)
				nested := filepath.Join(checkout, "nested", "caller")
				t.Chdir(nested)
				destination := filepath.Join("aliases", "pair")
				expected := filepath.Join(checkout, destination)
				if form == "absolute-unclean" {
					destination = checkout + string(filepath.Separator) + "aliases" + string(filepath.Separator) + ".." + string(filepath.Separator) + "pair"
					expected = filepath.Join(checkout, "pair")
				}
				var got string
				if withContext {
					got = AtContext(t, t.Context(), destination)
				} else {
					got = At(t, destination)
				}
				if got != destination {
					t.Fatalf("returned destination %q; want unchanged %q", got, destination)
				}
				sourceInfo, err := os.Stat(source)
				if err != nil {
					t.Fatal(err)
				}
				placedInfo, err := os.Stat(expected)
				if err != nil {
					t.Fatalf("checkout-root placement missing: %v", err)
				}
				if !os.SameFile(sourceInfo, placedInfo) {
					t.Fatal("legacy placement did not reuse exact suite candidate")
				}
				if form == "relative" {
					if _, err := os.Stat(filepath.Join(nested, destination)); !os.IsNotExist(err) {
						t.Fatalf("placement used caller cwd: %v", err)
					}
				}
				if raw, err := os.ReadFile(source); err != nil || string(raw) != string(body) {
					t.Fatalf("source candidate changed: %q %v", raw, err)
				}
			})
		}
	}
}
