package couchcmd

import (
	"bytes"
	"os"
	"regexp"
	"strings"
	"testing"
)

// helpVerbLine matches a usage line naming a verb: the bare launch or a flag.
var helpVerbLine = regexp.MustCompile(`^(?:usage: |\s+)couch(?: (--[a-z-]+|mock))?\b`)

// helpCosts reads each verb's tag back out of `couch --help`.
func helpCosts(t *testing.T) map[string]verbCost {
	t.Helper()
	var out bytes.Buffer
	usage(&out)
	got := map[string]verbCost{}
	for _, line := range strings.Split(out.String(), "\n") {
		m := helpVerbLine.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		verb := m[1]
		if verb == "" {
			verb = launchVerb
		}
		open := strings.LastIndex(line, "  [")
		if open < 0 || !strings.HasSuffix(line, "]") {
			t.Errorf("help line for %s carries no cost tag: %q", verb, line)
			continue
		}
		got[verb] = verbCost(line[open+3 : len(line)-1])
	}
	return got
}

var skillCostLine = regexp.MustCompile("^- \\*\\*(light|seconds|heavy):\\*\\* (.+)$")

// skillCosts reads the skill's cost section: one bullet per tier listing its
// verbs in code spans.
func skillCosts(t *testing.T) (map[string]verbCost, string) {
	t.Helper()
	start := strings.Index(couchSkill, "\n## What each verb costs\n")
	if start < 0 {
		t.Fatal("the skill has no cost section")
	}
	section := couchSkill[start+1:]
	if end := strings.Index(section[3:], "\n## "); end >= 0 {
		section = section[:end+3]
	}
	got := map[string]verbCost{}
	for _, line := range strings.Split(section, "\n") {
		m := skillCostLine.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		for _, span := range regexp.MustCompile("`([^`]+)`").FindAllStringSubmatch(m[2], -1) {
			verb := span[1]
			if _, dup := got[verb]; dup {
				t.Errorf("skill lists %s twice", verb)
			}
			got[verb] = verbCost(m[1])
		}
	}
	return got, section
}

func assertCosts(t *testing.T, surface string, got map[string]verbCost) {
	t.Helper()
	for verb, want := range verbCosts {
		if got[verb] != want {
			t.Errorf("%s: %s is %q, want %q", surface, verb, got[verb], want)
		}
	}
	for verb := range got {
		if _, ok := verbCosts[verb]; !ok {
			t.Errorf("%s tags %s, which has no recorded cost", surface, verb)
		}
	}
}

// Every verb in --help and in the skill carries the tag the table records,
// so an agent reading either one learns the same cost (pair#490).
func TestHelpAndSkillCarryEveryVerbCost(t *testing.T) {
	for _, verb := range []string{"--keys", "--set-model", "--interrupt"} {
		if verbCosts[verb] != costSeconds {
			t.Errorf("%s must be seconds cost for bounded input/control observation", verb)
		}
	}
	if verbCosts["--notify"] != costLight {
		t.Error("notify must be a light verb")
	}
	assertCosts(t, "couch --help", helpCosts(t))
	skill, section := skillCosts(t)
	assertCosts(t, "couch --skill", skill)

	var out bytes.Buffer
	usage(&out)
	flat := func(s string) string { return strings.Join(strings.Fields(s), " ") }
	if !strings.Contains(flat(out.String()), flat(costGuidance)) {
		t.Error("couch --help lacks the cost guidance")
	}
	if !strings.Contains(flat(section), flat(costGuidance)) {
		t.Error("the skill's cost section lacks the cost guidance")
	}
	readme, err := os.ReadFile("../../../README.md")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(flat(string(readme)), flat(costGuidance)) {
		t.Error("README lacks the cost guidance")
	}
}

func TestNotifyHelpAndSkillExplainHubAuthority(t *testing.T) {
	var help bytes.Buffer
	usage(&help)
	for name, text := range map[string]string{"help": help.String(), "skill": couchSkill} {
		for _, want := range []string{"--notify", "swallowed", "hub", "do not retry automatically"} {
			if !strings.Contains(text, want) {
				t.Errorf("%s missing %q", name, want)
			}
		}
	}
}

// A verb without a recorded cost cannot reach --help untagged.
func TestVerbLineRefusesAnUntaggedVerb(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("verbLine accepted a verb with no cost")
		}
	}()
	verbLine(&bytes.Buffer{}, "--no-such-verb", "couch --no-such-verb")
}
