package launcher

import (
	"context"
	"errors"
	"fmt"
	"github.com/xianxu/pair/cmd/internal/zellijtest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"

	"github.com/xianxu/pair/cmd/internal/zellijexec"
)

// Captured from installed Zellij 0.45.1's parse-only Clap rejection. The zero
// limit is real when its socket-directory prefix exhausts the native budget.
func nativeNameValidator(session, limit string) string {
	return "error: invalid value '" + session + "' for '--session <SESSION>': session name must be less than " + limit + " characters\n\nFor more information, try '--help'.\n"
}

type diagnosticCase struct {
	name, stdout, stderr                string
	status                              int
	inventory, acceptsName, rejectsName bool
}

func diagnosticMatrix() []diagnosticCase {
	valid := nativeNameValidator("pair-new", "108")
	cases := []diagnosticCase{
		{name: "native-validator", stderr: valid, status: 2, rejectsName: true},
		{name: "native-zero-limit", stderr: nativeNameValidator("pair-new", "0"), status: 2, rejectsName: true},
		{name: "native-empty-inventory", stderr: "No active zellij sessions found.\n", status: 1, inventory: true},
		{name: "native-no-active", stderr: "There is no active session!\n", status: 1, acceptsName: true},
		{name: "native-other-active", stderr: "Session 'pair-new' not found. The following sessions are active:\nother\n", status: 1, acceptsName: true},
		{name: "strict-zero-clients", stdout: "CLIENT_ID ZELLIJ_PANE_ID RUNNING_COMMAND\n", acceptsName: true},
		{name: "strict-one-client", stdout: "CLIENT_ID ZELLIJ_PANE_ID RUNNING_COMMAND\n1 terminal_0 sh\n", acceptsName: true},
		{name: "strict-inventory", stdout: "pair-new [Created 1s ago]\n", inventory: true},
		{name: "empty-success"},
		{name: "unrelated-success", stdout: "unrelated output\n"},
		{name: "malformed-clients", stdout: "CLIENT_ID ZELLIJ_PANE_ID RUNNING_COMMAND\nbroken\n"},
		{name: "success-validator-stderr-only", stderr: valid},
		{name: "success-clients-validator-stderr", stdout: "CLIENT_ID ZELLIJ_PANE_ID RUNNING_COMMAND\n", stderr: valid, acceptsName: true},
	}
	for _, status := range []int{0, 1, 3, 127} {
		cases = append(cases, diagnosticCase{name: fmt.Sprintf("validator-status-%d", status), stderr: valid, status: status})
	}
	for _, stdout := range []string{" ", "\n", "unrelated output\n", "CLIENT_ID ZELLIJ_PANE_ID RUNNING_COMMAND\n", valid} {
		cases = append(cases, diagnosticCase{name: fmt.Sprintf("validator-stdout-%q", stdout), stdout: stdout, stderr: valid, status: 2})
	}
	for _, variant := range []struct{ name, diagnostic string }{
		{"wrong-target", nativeNameValidator("foreign", "108")},
		{"missing-target", "error: session name must be less than 108 characters\n"},
		{"substring-only", "session name must be less than 108 characters"},
		{"wrong-option", strings.Replace(valid, "--session <SESSION>", "--layout <LAYOUT>", 1)},
		{"wrong-header", strings.Replace(valid, "error: invalid value", "warning: invalid value", 1)},
		{"missing-header", strings.TrimPrefix(valid, "error: ")},
		{"wrong-footer", strings.Replace(valid, "--help", "--version", 1)},
		{"missing-footer", strings.Split(valid, "\n\n")[0] + "\n"},
		{"missing-separator", strings.Replace(valid, "\n\n", "\n", 1)},
		{"missing-final-newline", strings.TrimSuffix(valid, "\n")},
		{"extra-prefix", "another error\n" + valid},
		{"extra-suffix", valid + "another error\n"},
		{"extra-blank", valid + "\n"},
		{"crlf", strings.ReplaceAll(valid, "\n", "\r\n")},
	} {
		cases = append(cases, diagnosticCase{name: variant.name, stderr: variant.diagnostic, status: 2})
	}
	for _, limit := range []string{"", "00", "01", "+1", "-1", "1.0", " 1", "1 ", "one", "99999999999999999999999999999999999999"} {
		cases = append(cases, diagnosticCase{name: "noncanonical-limit-" + limit, stderr: nativeNameValidator("pair-new", limit), status: 2})
	}
	for _, base := range []diagnosticCase{cases[2], cases[3], cases[4]} {
		for _, status := range []int{0, 2, 3} {
			cases = append(cases, diagnosticCase{name: fmt.Sprintf("%s-status-%d", base.name, status), stderr: base.stderr, status: status})
		}
		for _, stdout := range []string{" ", "unrelated\n"} {
			cases = append(cases, diagnosticCase{name: fmt.Sprintf("%s-stdout-%q", base.name, stdout), stdout: stdout, stderr: base.stderr, status: 1})
		}
		cases = append(cases, diagnosticCase{name: base.name + "-extra-diagnostic", stderr: base.stderr + "another error\n", status: 1})
	}
	return cases
}

type diagnosticNameRuntime struct {
	*fakeRuntime
	probe func(string) error
}

func (r diagnosticNameRuntime) ProbeSessionName(name string) error { return r.probe(name) }

// Exercise both production normalizers through the owned-process collector,
// including the assignment-local monotone oracle that consumes name evidence.
func TestZellijDiagnosticAuthorityMatrix(t *testing.T) {
	for _, tc := range diagnosticMatrix() {
		reply := zellijtest.Case(t, "launcher-matrix-"+tc.name)
		if reply.Stdout != tc.stdout || reply.Stderr != tc.stderr || reply.Code != tc.status {
			t.Fatalf("catalog changed independent diagnostic fixture %q", tc.name)
		}
		shellReply := zellijtest.ShellCase(t, "launcher-matrix-"+tc.name)
		for _, consumer := range []string{"inventory", "name-bound"} {
			for _, fault := range []string{"none", "joined-exit", "collection", "cancel", "collection-and-cancel"} {
				t.Run(tc.name+"/"+consumer+"/"+fault, func(t *testing.T) {
					dir := t.TempDir()
					path := filepath.Join(dir, "zellij")
					script := "#!/bin/sh\n"
					script += shellReply
					if err := os.WriteFile(path, []byte(script), 0700); err != nil {
						t.Fatal(err)
					}
					t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
					ctx, cancel := context.WithCancel(context.Background())
					defer cancel()
					calls := 0
					run := func(cmd *exec.Cmd) error {
						calls++
						err := zellijexec.Run(cmd)
						switch fault {
						case "joined-exit":
							if err == nil {
								return errors.New("query collector failed without an exit error")
							}
							return errors.Join(err)
						case "collection":
							return errors.Join(err, syscall.EIO)
						case "cancel":
							cancel()
						case "collection-and-cancel":
							cancel()
							return errors.Join(err, syscall.EIO)
						}
						return err
					}
					var gotErr error
					if consumer == "inventory" {
						_, gotErr = (ZellijSource{Path: path, queryRun: run}).LivenessContext(ctx)
						if (gotErr == nil) != (fault == "none" && tc.inventory) {
							t.Fatalf("inventory authority=%v, error=%v", gotErr == nil, gotErr)
						}
					} else {
						rt := diagnosticNameRuntime{fakeRuntime: newFakeRuntime(), probe: func(name string) error {
							gotErr = (OSRuntime{nameQueryRun: run}).ProbeSessionNameContext(ctx, name)
							return gotErr
						}}
						accepts, refusedAt, queryError := sessionNameAcceptor(rt)
						accepted := accepts("pair-new")
						_, refused := refusedAt()
						wantAccept, wantRefuse := fault == "none" && tc.acceptsName, fault == "none" && tc.rejectsName
						if accepted != wantAccept || refused != wantRefuse {
							t.Fatalf("name accepted=%v refused=%v error=%v; want %v,%v", accepted, refused, gotErr, wantAccept, wantRefuse)
						}
						if !wantAccept && !wantRefuse {
							wantCalls := 1
							if fault == "none" && (tc.name == "empty-success" || tc.name == "strict-inventory") {
								wantCalls = 3 // The shared read exhausts recovery before teaching no bound.
							}
							if queryError() == nil || accepts("x") || accepts("pair-much-longer-name") || calls != wantCalls {
								t.Fatalf("unknown query taught a bound or continued: error=%v calls=%d", queryError(), calls)
							}
						} else if queryError() != nil {
							t.Fatalf("native evidence became unknown: %v", queryError())
						}
						if wantAccept && (!accepts("x") || calls != 1) {
							t.Fatalf("positive bound lost: calls=%d", calls)
						}
						if wantRefuse && (accepts("pair-much-longer-name") || calls != 1) {
							t.Fatalf("negative bound lost: calls=%d", calls)
						}
					}
					if fault != "none" && tc.status != 0 {
						var exit *exec.ExitError
						if !errors.As(gotErr, &exit) || exit.ExitCode() != tc.status {
							t.Fatalf("lost command exit: %v", gotErr)
						}
					}
					if strings.Contains(fault, "collection") && !errors.Is(gotErr, syscall.EIO) {
						t.Fatalf("lost collection error: %v", gotErr)
					}
					if strings.Contains(fault, "cancel") && !errors.Is(gotErr, context.Canceled) {
						t.Fatalf("lost cancellation: %v", gotErr)
					}
				})
			}
		}
	}
}

func TestNativeValidatorStillMeasuresMonotoneNameBudget(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "zellij")
	// Model native validation before action dispatch, using the requested name
	// in the complete rejection. Every accepted probe receives a strict table.
	script := `#!/bin/sh
name=$2
if [ "${#name}" -gt 30 ]; then
 ` + zellijtest.ReplyCommand(t) + ` --kind name-invalid --data "{\"name\":\"$name\",\"limit\":31}"
 exit $?
fi
` + zellijtest.ShellCase(t, "launcher-monotone-clients") + `
`
	if err := os.WriteFile(path, []byte(script), 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	calls := 0
	rt := diagnosticNameRuntime{fakeRuntime: newFakeRuntime(), probe: func(name string) error {
		return (OSRuntime{nameQueryRun: func(cmd *exec.Cmd) error { calls++; return zellijexec.Run(cmd) }}).ProbeSessionNameContext(context.Background(), name)
	}}
	accepts, refusedAt, queryError := sessionNameAcceptor(rt)
	if limit, ok := measureAcceptedLimit(accepts); !ok || limit != 30 {
		t.Fatalf("measured native budget=%d,%v, error=%v", limit, ok, queryError())
	}
	if bound, ok := refusedAt(); !ok || bound != 31 || queryError() != nil {
		t.Fatalf("native refusal=%d,%v, error=%v", bound, ok, queryError())
	}
	measuredCalls := calls
	for n := len(sessionNameProbeMarker); n <= 64; n++ {
		if got := accepts(sessionNameProbeMarker + strings.Repeat("x", n-len(sessionNameProbeMarker))); got != (n <= 30) {
			t.Fatalf("name length %d accepted=%v", n, got)
		}
	}
	if calls != measuredCalls || calls > 8 {
		t.Fatalf("measured bounds required more subprocesses: measured=%d final=%d", measuredCalls, calls)
	}
}
