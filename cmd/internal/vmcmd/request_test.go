package vmcmd

import (
	"fmt"
	"reflect"
	"testing"
	"time"
)

func TestParseRunDefaultsAndLiteralCommand(t *testing.T) {
	args := []string{"--checkout", "/work/repo", "--", "printf", "%s", "--lane", "train", "a b", ""}
	got, err := ParseRun(args)
	if err != nil {
		t.Fatal(err)
	}
	if got.Checkout != "/work/repo" || got.Lane != AdhocLane || got.Timeout != 20*time.Minute || got.Artifacts != "" || !reflect.DeepEqual(got.Args, args[3:]) {
		t.Fatalf("request: %#v", got)
	}
	args[3] = "changed"
	if got.Args[0] != "printf" {
		t.Fatal("request aliases caller argv")
	}
}

func TestParseRunExplicitOptions(t *testing.T) {
	got, err := ParseRun([]string{"--lane=train", "--checkout=/work/repo", "--timeout", "1ns", "--artifacts", "/work/output", "--", "true"})
	if err != nil {
		t.Fatal(err)
	}
	if got.Lane != TrainLane || got.Timeout != time.Nanosecond || got.Artifacts != "/work/output" {
		t.Fatalf("request: %#v", got)
	}
}

func TestParseRunRejectsInvalidRequests(t *testing.T) {
	cases := [][]string{
		{}, {"--", "true"}, {"--checkout", "/repo"}, {"--checkout", "/repo", "--"},
		{"--checkout", "/repo", "--", ""}, {"--checkout", "repo", "--", "true"},
		{"--checkout", "/repo", "--", "true\x00bad"},
		{"--checkout", "/repo", "--", "true", "argument\x00bad"},
		{"--checkout", "/repo/../other", "--", "true"}, {"--checkout", "/repo/", "--", "true"},
		{"--checkout", "/repo\x00bad", "--", "true"}, {"--checkout", "/repo\nbad", "--", "true"},
		{"--checkout", "/repo", "--artifacts", "relative", "--", "true"},
		{"--checkout", "/repo", "--artifacts=", "--", "true"},
		{"--checkout", "/repo", "--lane", "invalid", "--", "true"},
		{"--checkout", "/repo", "--lane=", "--", "true"},
		{"--checkout", "/repo", "--timeout", "0", "--", "true"},
		{"--checkout", "/repo", "--timeout", "-1s", "--", "true"},
		{"--checkout", "/repo", "--timeout", "999999999999999999h", "--", "true"},
		{"--checkout", "/repo", "--unknown", "value", "--", "true"},
		{"--checkout", "/repo", "--checkout=/other", "--", "true"},
		{"--checkout", "/repo", "true", "--", "true"}, {"--checkout=", "--", "true"},
	}
	for _, args := range cases {
		t.Run(stringJoinForTest(args), func(t *testing.T) {
			if got, err := ParseRun(args); err == nil {
				t.Fatalf("accepted %q: %#v", args, got)
			}
		})
	}
}

func stringJoinForTest(args []string) string { return fmt.Sprint(args) }
