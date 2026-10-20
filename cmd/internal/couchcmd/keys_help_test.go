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
	for _, step := range agentdrive.Steps {
		if !strings.Contains(got, step.Syntax) || !strings.Contains(got, step.Meaning) {
			t.Errorf("keys help omits %s", step.Syntax)
		}
	}
	for _, want := range []string{"couch --keys pair:7 clear-line text:/usage submit", "couch --peek pair:7", "couch --keys pair:7 esc", "64 steps", "8192 bytes"} {
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
