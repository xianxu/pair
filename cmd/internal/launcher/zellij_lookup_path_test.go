package launcher

import (
	"net"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"testing"
)

func TestLookupSocketUsesNativeSymlinkParentTraversal(t *testing.T) {
	for _, source := range []string{"explicit", "temporary", "runtime"} {
		for _, nativePresent := range []bool{true, false} {
			t.Run(source+"/"+map[bool]string{true: "native-present-cleaned-absent", false: "native-absent-cleaned-present"}[nativePresent], func(t *testing.T) {
				// Budget the longest actual bind path; all topology paths stay owned.
				base := nativeSocketFixtureRoot(t, filepath.Join("real", "sockets", "zellij-"+strconv.Itoa(os.Getuid()), "contract_version_1", "pair-a"))
				actual := filepath.Join(base, "real")
				if err := os.MkdirAll(filepath.Join(actual, "child"), 0700); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(filepath.Join(actual, "child"), filepath.Join(base, "alias")); err != nil {
					t.Fatal(err)
				}
				// filepath.Join would erase the traversal we need the kernel to resolve.
				root := base + "/alias/../sockets"
				suffix := ""
				if source == "temporary" {
					suffix = "/zellij-" + strconv.Itoa(os.Getuid())
				}
				if source == "runtime" {
					suffix = "/zellij"
				}
				nativeRoot := filepath.Join(actual, "sockets") + suffix
				cleanedRoot := filepath.Join(base, "sockets") + suffix
				for _, r := range []string{nativeRoot, cleanedRoot} {
					if err := os.MkdirAll(filepath.Join(r, "contract_version_1"), 0700); err != nil {
						t.Fatal(err)
					}
				}
				selected := cleanedRoot
				if nativePresent {
					selected = nativeRoot
				}
				listener, err := net.Listen("unix", filepath.Join(selected, "contract_version_1", "pair-a"))
				if err != nil {
					t.Fatal(err)
				}
				t.Cleanup(func() {
					if err := listener.Close(); err != nil {
						t.Error(err)
					}
				})
				wantRoot := root + suffix
				var resolved string
				switch source {
				case "explicit":
					resolved, err = ResolveZellijSocketRoot(root, "/unused", "/ignored", runtime.GOOS, os.Getuid())
					t.Setenv("ZELLIJ_SOCKET_DIR", resolved)
				case "temporary":
					resolved, err = ResolveZellijSocketRoot("", root, "", runtime.GOOS, os.Getuid())
					t.Setenv("ZELLIJ_SOCKET_DIR", "")
					t.Setenv("XDG_RUNTIME_DIR", "")
					t.Setenv("TMPDIR", root)
				case "runtime":
					// Exercise Linux's resolver branch on every host; on Linux also
					// drive its real environment selection at the query boundary.
					resolved, err = ResolveZellijSocketRoot("", "/unused", root, "linux", os.Getuid())
					if runtime.GOOS == "linux" {
						t.Setenv("ZELLIJ_SOCKET_DIR", "")
						t.Setenv("XDG_RUNTIME_DIR", root)
					} else {
						t.Setenv("ZELLIJ_SOCKET_DIR", resolved)
					}
				}
				if err != nil {
					t.Fatal(err)
				}
				if resolved != wantRoot {
					t.Errorf("%s root=%q want native spelling %q", source, resolved, wantRoot)
				}
				if got := ObserveSocket(os.Lstat(wantRoot+"/contract_version_1/pair-a")) == SocketPresent; got != nativePresent {
					t.Fatalf("native filesystem oracle=%v want=%v", got, nativePresent)
				}
				if got := sessionSocketPresent("pair-a"); got != nativePresent {
					t.Errorf("native socket present=%v, admission observed=%v; root=%q", nativePresent, got, root)
				}
				args := []string{"--session", "pair-a", "action", "list-clients"}
				frames := []emptyReplyFrame{{Args: args, Exit: 1, Stderr: "There is no active session!\n"}}
				if nativePresent {
					frames = append(frames, emptyReplyFrame{Args: args, Stdout: "CLIENT_ID ZELLIJ_PANE_ID RUNNING_COMMAND\n"})
				}
				fake := newEmptyReplyFake(t, frames...)
				_, err = fake.source.runContext(t.Context(), args...)
				if nativePresent && (err != nil || fake.calls(t) != 2) {
					t.Fatalf("native socket retry err=%v calls=%d", err, fake.calls(t))
				}
				if !nativePresent && (err == nil || fake.calls(t) != 1) {
					t.Fatalf("unrelated cleaned socket admitted retry: err=%v calls=%d", err, fake.calls(t))
				}
			})
		}
	}
}
