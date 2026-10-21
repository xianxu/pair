package launcher

import (
	"bytes"
	"context"
	"errors"
	"github.com/xianxu/pair/cmd/internal/zellijtest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func installEmptyReplyLegacyCLI(t *testing.T, emptyStage string) string {
	t.Helper()
	dir := t.TempDir()
	log := filepath.Join(dir, "calls")
	script := `#!/bin/sh
printf '%s\n' "$*" >> "$PAIR562_CALLS"
if [ "$*" = "$PAIR562_EMPTY_STAGE" ]; then ` + zellijtest.ShellCase(t, "launcher-empty-unanswered") + `; fi
case "$*" in
 'list-sessions --short') ` + zellijtest.ShellCase(t, "launcher-empty-prefix-short") + ` ;;
 'list-sessions --no-formatting') ` + zellijtest.ShellCase(t, "launcher-empty-prefix-full") + ` ;;
 '--session pair-work action list-clients') ` + zellijtest.ShellCase(t, "launcher-empty-prefix-clients") + ` ;;
 *) exit 1 ;;
esac
`
	if err := os.WriteFile(filepath.Join(dir, "zellij"), []byte(script), 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("PAIR562_CALLS", log)
	t.Setenv("PAIR562_EMPTY_STAGE", emptyStage)
	return log
}
func TestEmptySessionListingCannotFreeAName(t *testing.T) {
	log := installEmptyReplyLegacyCLI(t, "list-sessions --no-formatting")
	if !(OSRuntime{}).SessionBlocksReuse("pair-work") {
		t.Fatal("empty query freed an unproven name")
	}
	raw, err := os.ReadFile(log)
	if err != nil {
		t.Fatal(err)
	}
	if string(raw) != strings.Repeat("list-sessions --no-formatting\n", 3) {
		t.Fatalf("unexpected attempt count or mutation: %s", raw)
	}
}
func TestEmptyNameProbeCannotBecomeMonotoneAcceptance(t *testing.T) {
	installEmptyReplyLegacyCLI(t, "--session pair-work action list-clients")
	if err := (OSRuntime{}).ProbeSessionName("pair-work"); err == nil {
		t.Fatal("empty query accepted name length")
	}
}
func TestListSessionsCannotDisplayEmptyReplyAsDetached(t *testing.T) {
	for _, stage := range []string{"list-sessions --no-formatting", "--session pair-work action list-clients"} {
		t.Run(stage, func(t *testing.T) {
			installEmptyReplyLegacyCLI(t, stage)
			rt := NewOSRuntime(t.TempDir(), t.TempDir())
			rows, err := rt.ListSessions()
			if err == nil || rows != nil {
				t.Fatalf("empty %s returned authoritative rows=%v err=%v", stage, rows, err)
			}
			if !strings.Contains(err.Error(), "no reply") {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}
func TestExplicitEmptyInventoryStillFreesAName(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "zellij"), []byte("#!/bin/sh\n"+zellijtest.ShellCase(t, "launcher-empty-explicit-inventory")), 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	if (OSRuntime{}).SessionBlocksReuse("pair-work") {
		t.Fatal("explicit observed empty inventory did not free name")
	}
}

func TestUnansweredNameProbeReportsRetryWithoutShorterTagAdvice(t *testing.T) {
	rt := newFakeRuntime()
	rt.uuids = []string{"S"}
	rt.probeErr = errors.New("zellij returned no reply")
	var stderr bytes.Buffer
	code, err := RunLaunch(baseOpts(LaunchArgs{Agent: "claude", ForcedTag: "bugfix"}), rt, &stderr)
	if err != nil || code != 1 {
		t.Fatalf("code=%d err=%v", code, err)
	}
	text := stderr.String()
	if !strings.Contains(text, "no reply") || strings.Contains(text, "shorter tag") {
		t.Fatalf("misleading recovery advice: %s", text)
	}
	if rt.launched != "" || len(rt.ledger["bugfix"]) != 0 || len(rt.sessionIndex.Entries) != 0 {
		t.Fatal("unanswered preflight caused launch writes")
	}
}

func TestUnansweredPromptCannotMeasureOrReportARefusal(t *testing.T) {
	rt := newFakeRuntime()
	rt.promptValue = "work"
	rt.probeErr = errors.New("zellij returned no reply")
	var stderr bytes.Buffer
	_, code, ok := promptForTag(rt, "work", func(s string) string { return "pair-" + s }, "work", &stderr)
	if code != 1 || ok || !strings.Contains(stderr.String(), "no reply") || strings.Contains(stderr.String(), "shorter") {
		t.Fatalf("code=%d ok=%v advice=%s", code, ok, stderr.String())
	}
	if rt.probeCount != 1 {
		t.Fatalf("unknown query led to %d probes", rt.probeCount)
	}
}

func TestNameLengthBoundsRequireAnExplicitValidatorRefusal(t *testing.T) {
	rt := newFakeRuntime()
	rt.probeErr = errors.New("probe in /tmp/too long directory returned no reply")
	accepts, refusedAt, queryError := sessionNameAcceptor(rt)
	if accepts("pair-work") {
		t.Fatal("unknown name accepted")
	}
	if _, known := refusedAt(); known || queryError() == nil {
		t.Fatal("unrelated error text became a monotone refusal")
	}
	if accepts("x") || rt.probeCount != 1 {
		t.Fatal("assignment continued probing after unknown")
	}
}

type nameProbeSequence struct {
	*fakeRuntime
	replies []error
}

func (r *nameProbeSequence) ProbeSessionName(string) error {
	r.probeCount++
	if len(r.replies) == 0 {
		return errors.New("unexpected extra probe")
	}
	err := r.replies[0]
	r.replies = r.replies[1:]
	return err
}
func TestUnknownDuringNameBudgetMeasurementDoesNotReuseOldRefusal(t *testing.T) {
	rt := &nameProbeSequence{fakeRuntime: newFakeRuntime(), replies: []error{errSessionNameTooLong, ErrZellijNoReply}}
	rt.promptValue = "work"
	var stderr bytes.Buffer
	_, code, ok := promptForTag(rt, "work", func(s string) string { return "pair-very-long-" + s }, "work", &stderr)
	if code != 1 || ok || !strings.Contains(stderr.String(), "no reply") || strings.Contains(stderr.String(), "shorter") {
		t.Fatalf("code=%d ok=%v advice=%s", code, ok, stderr.String())
	}
	if rt.probeCount != 2 {
		t.Fatalf("budget probe continued after no answer: %d calls", rt.probeCount)
	}
}
func TestNameAcceptorFreshAssignmentCanMakeANewObservation(t *testing.T) {
	rt := newFakeRuntime()
	rt.probeErr = ErrZellijNoReply
	accepts, refusedAt, queryError := sessionNameAcceptor(rt)
	if accepts("pair-work") || queryError() == nil {
		t.Fatal("no answer accepted")
	}
	if _, known := refusedAt(); known {
		t.Fatal("no answer learned refusal bound")
	}
	rt.probeErr = nil
	if accepts("x") || rt.probeCount != 1 {
		t.Fatal("unknown assignment issued a hidden new probe")
	}
	fresh, _, queryError := sessionNameAcceptor(rt)
	if !fresh("pair-work") || queryError() != nil || rt.probeCount != 2 {
		t.Fatal("fresh assignment did not accept fresh evidence")
	}
}

func TestSessionOwnerProbeEmptyInjectedReplyRemainsUnanswered(t *testing.T) {
	server := SessionServerIdentity{PID: 42, Identity: "same", Session: "pair-work"}
	for _, raw := range []string{"", " \n\t"} {
		world := &ownerWorld{present: true, generation: server, panes: []byte(raw)}
		probe := SessionOwnerProbe{IO: world}
		got, err := probe.Probe(context.Background(), server.Session, "/tmp/data", "0123456789abcdef", "work")
		if !errors.Is(err, ErrZellijNoReply) || got.State != SessionOwnerUnknown {
			t.Fatalf("empty injected reply=%+v,%v", got, err)
		}
		if err := probe.Revalidate(context.Background(), got); err == nil {
			t.Fatal("empty reply gained effect authority")
		}
	}
}

func TestEmptySuccessfulNameProbeCannotUseStderrAsItsReply(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "zellij")
	if err := os.WriteFile(path, []byte("#!/bin/sh\n"+zellijtest.ShellCase(t, "launcher-empty-stderr-only")), 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	if err := (OSRuntime{}).ProbeSessionName("pair-work"); !errors.Is(err, ErrZellijNoReply) {
		t.Fatalf("stderr-only success accepted name: %v", err)
	}
}
