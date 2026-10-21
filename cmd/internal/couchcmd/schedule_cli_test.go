package couchcmd

import (
	"reflect"
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/couchcore"
)

func TestScheduleCLIStatus(t *testing.T) {
	for _, args := range [][]string{{"--schedule", "--status"}, {"--schedule", "--status", "--json"}} {
		if _, err := ParseCLI(args, couchcore.Operations()); err != nil {
			t.Fatalf("ParseCLI(%q): %v", args, err)
		}
	}
}

func TestScheduleCLIForms(t *testing.T) {
	for _, tt := range []struct {
		args []string
		want scheduleArgs
	}{
		{[]string{"--schedule", "--status"}, scheduleArgs{verb: "status"}},
		{[]string{"--schedule", "--status", "--json"}, scheduleArgs{verb: "status", json: true}},
		{[]string{"--schedule", "run", "refresh-plan", "--"}, scheduleArgs{verb: "run", task: "refresh-plan", args: []string{}}},
		{[]string{"--schedule", "run", "refresh-plan", "--", "--help", "", "--layout3", "--json", "--", "$(literal)"}, scheduleArgs{verb: "run", task: "refresh-plan", args: []string{"--help", "", "--layout3", "--json", "--", "$(literal)"}}},
		{[]string{"--schedule", "history", "refresh-plan"}, scheduleArgs{verb: "history", task: "refresh-plan"}},
		{[]string{"--schedule", "history", "refresh-plan", "--json"}, scheduleArgs{verb: "history", task: "refresh-plan", json: true}},
		{[]string{"--schedule", "history", "refresh-plan", "--run", "run_1-2", "--json"}, scheduleArgs{verb: "history", task: "refresh-plan", runID: "run_1-2", json: true}},
		{[]string{"--schedule", "history", "refresh-plan", "--json", "--run", "run_1-2"}, scheduleArgs{verb: "history", task: "refresh-plan", runID: "run_1-2", json: true}},
		{[]string{"--schedule", "history", strings.Repeat("a", 128), "--run", strings.Repeat("1", 128)}, scheduleArgs{verb: "history", task: strings.Repeat("a", 128), runID: strings.Repeat("1", 128)}},
	} {
		got, err := ParseCLI(tt.args, couchcore.Operations())
		if err != nil || got.kind != cliSchedule || got.schedule == nil || !reflect.DeepEqual(*got.schedule, tt.want) || got.layout != "" {
			t.Errorf("ParseCLI(%q) = %#v, %v; want %#v", tt.args, got, err, tt.want)
		}
	}
}

func TestScheduleCLIRejectsMalformedForms(t *testing.T) {
	bad := [][]string{
		{"--schedule"}, {"--schedule", "status"}, {"--schedule", "--json", "--status"},
		{"--schedule", "--status", "--json", "--json"}, {"--schedule", "--status", "--layout3"},
		{"--schedule", "run"}, {"--schedule", "run", "task"}, {"--schedule", "run", "task", "--help"},
		{"--schedule", "run", "task", "--json", "--"}, {"--schedule", "history"},
		{"--schedule", "history", "task", "--json", "--json"}, {"--schedule", "history", "task", "extra"},
		{"--schedule", "history", "task", "--run"}, {"--schedule", "history", "task", "--run", "id", "--run", "id"},
		{"--schedule", "history", "task", "--layout3"}, {"--schedule", "history", "task", "--"},
	}
	for _, value := range []string{"", ".", "..", "../task", "a/b", "a\\b", "/tmp/task", "hub:task", "hub#task", "Upper", "-task", "_task", "a.b", "has space", "a\n", "a\x00", "é", strings.Repeat("a", 129)} {
		bad = append(bad, []string{"--schedule", "run", value, "--"}, []string{"--schedule", "history", value}, []string{"--schedule", "history", "task", "--run", value})
	}
	for _, args := range bad {
		got, err := ParseCLI(args, couchcore.Operations())
		if err == nil || got.kind != cliInvalid || got.schedule != nil {
			t.Errorf("ParseCLI(%q) = %#v, %v; want empty invalid invocation", args, got, err)
		}
	}
}

func TestScheduleCLIArgvDoesNotAliasCaller(t *testing.T) {
	args := []string{"--schedule", "run", "task", "--", "original"}
	got, err := ParseCLI(args, couchcore.Operations())
	if err != nil {
		t.Fatal(err)
	}
	args[4] = "changed"
	if got.schedule.args[0] != "original" {
		t.Fatal("task argv aliases caller storage")
	}
}
