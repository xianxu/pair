package couchcmd

import (
	"bytes"
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/couchcore"
)

func TestMockCLI(t *testing.T) {
	for _, args := range [][]string{{"mock", "serve", "mocks"}, {"mock", "use", "--layout3", "mock-2"}, {"mock", "serve", "--layout2", "--port", "0"}, {"mock", "--help"}} {
		inv, err := ParseCLI(args, couchcore.Operations())
		if err != nil || inv.kind != cliMock {
			t.Fatalf("%v: %+v %v", args, inv, err)
		}
	}
	for _, args := range [][]string{{"mock", "serve"}, {"mock", "use", "x"}, {"mock", "serve", "x", "--port", "-1"}, {"mock", "serve", "x", "extra"}, {"mock", "use", "x", "../../oops"}} {
		if _, err := ParseCLI(args, couchcore.Operations()); err == nil {
			t.Errorf("accepted %v", args)
		}
	}
	inv, err := ParseCLI([]string{"--", "mock"}, couchcore.Operations())
	if err != nil || inv.kind != cliLaunch {
		t.Fatalf("explicit directory: %+v %v", inv, err)
	}
	var out, errs bytes.Buffer
	rt := noMockRuntime{}
	if rc := RunWithRuntime([]string{"mock", "--help"}, nil, &out, &errs, rt); rc != 0 || !strings.Contains(out.String(), "mock serve") {
		t.Fatalf("help rc=%d out=%q err=%q", rc, &out, &errs)
	}
	if rc := RunWithRuntime([]string{"mock", "use", t.TempDir(), "mock-1"}, nil, &out, &errs, rt); rc != 1 {
		t.Fatalf("missing project rc=%d", rc)
	}
}

type noMockRuntime struct{ Runtime }

func (noMockRuntime) Getenv(string) string { panic("mock command initialized Couch") }
