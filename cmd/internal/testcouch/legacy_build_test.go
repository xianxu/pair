package testcouch

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/testpair"
)

// The compatibility child exercises the real process-owned BinaryContext cache;
// Run must outlive every per-test Build cleanup, exactly as in Build's consumers.
func TestMain(m *testing.M) { os.Exit(testpair.Run(m)) }

const legacyBuildMode = "PAIR_TESTCOUCH_LEGACY_BUILD_MODE"
const legacyBuildObserved = "PAIR_TESTCOUCH_LEGACY_BUILD_OBSERVED"

func TestLegacyBuildProcess(t *testing.T) {
	mode := os.Getenv(legacyBuildMode)
	if mode == "" {
		t.Skip("subprocess-only compatibility fixture")
	}
	var selected string
	for _, name := range []string{"first caller", "later caller after cleanup"} {
		var placement string
		t.Run(name, func(t *testing.T) {
			// No TB substitute or injected Build function: this is the public wrapper.
			b := Build(t)
			placement = b.Pair
			source := testpair.Binary(t)
			if selected != "" && source != selected {
				t.Fatalf("process candidate changed: %s -> %s", selected, source)
			}
			selected = source
			actual, err := os.Stat(b.Pair)
			if err != nil {
				t.Fatal(err)
			}
			expected, err := os.Stat(source)
			if err != nil {
				t.Fatal(err)
			}
			if !os.SameFile(actual, expected) {
				t.Fatalf("Build replaced selected %s with a different Pair at %s", source, b.Pair)
			}
		})
		if t.Failed() {
			return
		}
		if _, err := os.Stat(placement); !os.IsNotExist(err) {
			t.Fatalf("Build cleanup retained private placement: %s %v", placement, err)
		}
		if _, err := os.Stat(selected); err != nil {
			t.Fatalf("Build cleanup removed process/suite candidate: %v", err)
		}
	}
	if err := os.WriteFile(os.Getenv(legacyBuildObserved), []byte(selected), 0600); err != nil {
		t.Fatal(err)
	}
}

func TestLegacyBuildPreservesCandidateSelectionAndProcessLifetime(t *testing.T) {
	for _, mode := range []string{"supplied", "fallback"} {
		t.Run(mode, func(t *testing.T) {
			root := testpair.CacheTempDir(t)
			tools, home, scratch := filepath.Join(root, "tools"), filepath.Join(root, "home"), filepath.Join(root, "scratch")
			for _, dir := range []string{tools, home, scratch, filepath.Join(tools, "cache")} {
				if err := os.Mkdir(dir, 0700); err != nil {
					t.Fatal(err)
				}
			}
			write := func(name, body string) {
				t.Helper()
				if err := os.WriteFile(filepath.Join(tools, name), []byte(body), 0700); err != nil {
					t.Fatal(err)
				}
			}
			// Only compilation/native version executables are fixtures. The actual
			// shared Pair builder and its hardlink/symlink placement execute unchanged.
			write("go", `#!/bin/sh
set -eu
base=${0%/*}
if [ "$1" = env ]; then
 printf '{"GOCACHE":"%s/cache","GOMODCACHE":"%s/cache"}\n' "$base" "$base"
 exit 0
fi
[ "$1" = build ] || exit 91
shift
output= package=
while [ "$#" -gt 0 ]; do
 case "$1" in -o) output=$2;shift 2;; *) package=$1;shift;; esac
done
[ -n "$output" ] || exit 92
if [ "$package" = ./cmd/pair-go ]; then printf 'pair\n' >> "$base/calls"; fi
printf '#!/bin/sh\nprintf "fixture Pair\\n"\n' > "$output"
/bin/chmod 700 "$output"
`)
			write("git", `#!/bin/sh
if [ "$1" = --exec-path ]; then printf '%s\n' "${0%/*}"; else printf 'selected-head\n'; fi
`)
			write("git-upload-pack", "#!/bin/sh\nexit 0\n")
			for _, name := range []string{"sdlc", "weave", "zellij", "nvim", "cc", "c++", "clang", "clang++", "xcrun", "ld", "ar", "ranlib", "dsymutil"} {
				write(name, "#!/bin/sh\nprintf 'fixture tool\\n'\n")
			}
			observed := filepath.Join(root, "observed")
			supplied := ""
			if mode == "supplied" {
				supplied = filepath.Join(root, "suite-pair")
				if err := os.WriteFile(supplied, []byte("#!/bin/sh\nprintf 'suite Pair\\n'\n"), 0700); err != nil {
					t.Fatal(err)
				}
			}
			env := testpair.Environment(t, home, "PATH="+tools+string(os.PathListSeparator)+os.Getenv("PATH"), "TMPDIR="+scratch, "PAIR_TEST_BINARY="+supplied, legacyBuildMode+"="+mode, legacyBuildObserved+"="+observed)
			executable, err := os.Executable()
			if err != nil {
				t.Fatal(err)
			}
			ctx := t.Context()
			var cancel context.CancelFunc
			if deadline, ok := t.Deadline(); ok {
				ctx, cancel = context.WithDeadline(ctx, deadline)
				defer cancel()
			}
			out, err := (couchcore.OSProvisionIO{Env: env}).Run(ctx, couchcore.ProvisionCommand{Program: executable, Args: []string{"-test.run=^TestLegacyBuildProcess$", "-test.v"}, Dir: runtimeSourceForTest(t)})
			if err != nil {
				t.Fatalf("legacy %s child: %v\n%s", mode, err, out)
			}
			selected, err := os.ReadFile(observed)
			if err != nil {
				t.Fatal(err)
			}
			calls, err := os.ReadFile(filepath.Join(tools, "calls"))
			if err != nil && !os.IsNotExist(err) {
				t.Fatal(err)
			}
			want := 0
			if mode == "fallback" {
				want = 1
			}
			if strings.Count(string(calls), "pair\n") != want {
				t.Fatalf("Pair fallback compilations=%q want%d", calls, want)
			}
			if mode == "supplied" {
				if string(selected) != supplied {
					t.Fatalf("selected %q instead of suite candidate %s", selected, supplied)
				}
				if raw, err := os.ReadFile(supplied); err != nil || !strings.Contains(string(raw), "suite Pair") {
					t.Fatalf("suite candidate changed: %q %v", raw, err)
				}
			} else if _, err := os.Stat(string(selected)); !os.IsNotExist(err) {
				t.Fatalf("Run retained fallback after process exit: %s %v", selected, err)
			}
		})
	}
}
