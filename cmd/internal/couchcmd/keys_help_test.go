package couchcmd

import (
	"bytes"
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/agentdrive"
)

func TestKeysHelpIsTheVerbsOwnHelp(t *testing.T) {
	for _, flag := range []string{"--help", "-h"} {
		inv, err := ParseCLI([]string{"--keys", flag}, nil)
		if err != nil || inv.kind != cliKeysHelp {
			t.Fatalf("couch --keys %s: %+v %v", flag, inv, err)
		}
	}
	var out bytes.Buffer
	keysUsage(&out)
	got := out.String()
	for _, step := range agentdrive.StepGroups() {
		if !strings.Contains(got, "  "+step.Syntax+" ") || !strings.Contains(got, step.Meaning) {
			t.Errorf("keys help omits %s", step.Syntax)
		}
	}
	covered := map[string]bool{}
	for _, step := range agentdrive.StepGroups() {
		covered[step.Syntax] = true
	}
	for _, step := range agentdrive.Steps {
		if !covered[step.Syntax] && !covered[step.Group] {
			t.Errorf("%s is in no help line", step.Syntax)
		}
	}
	for name, reason := range agentdrive.RefusedSteps {
		if !strings.Contains(got, string(name)) || !strings.Contains(got, reason) {
			t.Errorf("keys help omits the refused %s", name)
		}
	}
	if strings.Count(got, "\n") > 60 {
		t.Errorf("keys help prints families member by member: %d lines", strings.Count(got, "\n"))
	}
	for _, want := range []string{"couch --keys pair:7 clear-line text:/usage enter", "couch --peek pair:7", "couch --keys pair:7 esc", "64 steps", "8192 bytes"} {
		if !strings.Contains(got, want) {
			t.Errorf("keys help omits %q:\n%s", want, got)
		}
	}
}

func TestCouchHelpNamesTheKeySteps(t *testing.T) {
	var out bytes.Buffer
	usage(&out)
	if !strings.Contains(out.String(), "STEP: "+agentdrive.StepSyntax()) || !strings.Contains(out.String(), "couch --keys --help") {
		t.Fatalf("couch --help --keys entry:\n%s", out.String())
	}
}

// TestSkillKeyStepsMatchTheTable fails when the skill's step sentence and
// agentdrive.Steps disagree in either direction (#716).
func TestSkillKeyStepsMatchTheTable(t *testing.T) {
	sentence := regexp.MustCompile("`--keys repo:N STEP\\.\\.\\. \\[--json\\]` accepts ([^.]*)\\.").FindStringSubmatch(couchSkill)
	if sentence == nil {
		t.Fatal("skill has no --keys steps sentence")
	}
	var listed []string
	for _, m := range regexp.MustCompile("`([^`]+)`").FindAllStringSubmatch(sentence[1], -1) {
		listed = append(listed, m[1])
	}
	if got, want := strings.Join(listed, " "), agentdrive.StepSyntax(); got != want {
		t.Fatalf("skill steps %q, table %q", got, want)
	}
	if !strings.Contains(couchSkill, "| `couch --keys pair:7 esc` |") {
		t.Fatal("skill quick table has no close-panel row")
	}
	limits := fmt.Sprintf("%d steps, %d KiB aggregate printable text, waits of 0–%d ms each and %d ms total", agentdrive.MaxKeySteps, agentdrive.MaxKeyTextBytes/1024, agentdrive.MaxKeyWaitMS, agentdrive.MaxKeyTotalWaitMS)
	if !strings.Contains(strings.Join(strings.Fields(couchSkill), " "), limits) {
		t.Fatalf("skill limits disagree with the constants; want %q", limits)
	}
}

func TestInterruptCLIRequiresConfirmAndKeysRefuseCtrlC(t *testing.T) {
	inv, err := ParseCLI([]string{"--interrupt", "pair:7", "--confirm"}, nil)
	if err != nil || inv.messageOp != "interrupt" || inv.drive == nil || !inv.drive.Interrupt || !inv.confirmed {
		t.Fatalf("interrupt parse %+v %v", inv, err)
	}
	if _, err := ParseCLI([]string{"--interrupt", "pair:7"}, nil); err == nil || !strings.Contains(err.Error(), "--confirm") {
		t.Fatalf("interrupt without confirm: %v", err)
	}
	if _, err := ParseCLI([]string{"--interrupt", "pair:7", "--confirm", "esc"}, nil); err == nil {
		t.Fatal("interrupt took a key step")
	}
	_, err = ParseCLI([]string{"--keys", "pair:7", "ctrl-c"}, nil)
	if err == nil || !strings.Contains(err.Error(), "couch --interrupt") {
		t.Fatalf("keys ctrl-c: %v", err)
	}
	var out bytes.Buffer
	usage(&out)
	if !strings.Contains(out.String(), "couch --interrupt repo:N --confirm") {
		t.Fatal("couch --help omits --interrupt")
	}
}
