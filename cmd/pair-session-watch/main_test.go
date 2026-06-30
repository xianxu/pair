package main

import (
	"reflect"
	"testing"
	"time"
)

func TestBuildOptionsFromArgsAndEnv(t *testing.T) {
	env := map[string]string{
		"HOME":                                "/home/me",
		"PAIR_DATA_DIR":                       "/tmp/pair-data",
		"PAIR_SESSION_WATCH_PID_WAIT_SECONDS": "3",
	}
	opts, ok := buildOptions([]string{"codex", "tag", "/repo", "resume", "old", "--flag"}, func(k string) string {
		return env[k]
	})
	if !ok {
		t.Fatalf("buildOptions returned !ok")
	}
	if opts.Agent != "codex" || opts.Tag != "tag" || opts.Cwd != "/repo" {
		t.Fatalf("opts identity = %+v", opts)
	}
	if opts.Home != "/home/me" || opts.DataDir != "/tmp/pair-data" {
		t.Fatalf("opts paths = %+v", opts)
	}
	if opts.PIDWait != 3*time.Second || opts.Timeout != 60*time.Second || opts.Poll != 100*time.Millisecond {
		t.Fatalf("opts durations = %+v", opts)
	}
	if !reflect.DeepEqual(opts.Args, []string{"resume", "old", "--flag"}) {
		t.Fatalf("opts args = %#v", opts.Args)
	}
}

func TestBuildOptionsRejectsMissingRequiredArgs(t *testing.T) {
	if _, ok := buildOptions([]string{"codex", "tag"}, func(string) string { return "" }); ok {
		t.Fatalf("buildOptions should reject missing cwd")
	}
}
