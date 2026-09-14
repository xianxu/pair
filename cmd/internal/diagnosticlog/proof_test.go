package diagnosticlog

import (
	"bufio"
	"errors"
	"github.com/xianxu/pair/cmd/internal/procutil"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"testing"
	"time"
)

func TestWriterProofRejectsLegacyAndUnknownHolders(t *testing.T) {
	me := Registration{PID: 123, Birth: "birth", Root: "/root"}
	for _, tt := range []struct {
		name     string
		holders  []Registration
		runtimes []Runtime
		err      error
		want     bool
	}{
		{"registered", []Registration{me}, []Runtime{{Process: me}}, nil, true},
		{"unknown holder", []Registration{{PID: 9, Birth: "old"}}, nil, nil, false},
		{"legacy transient", nil, []Runtime{{Process: Registration{PID: 8, Birth: "old", Root: "/root"}}}, nil, false},
		{"other root legacy", nil, []Runtime{{Process: Registration{PID: 8, Birth: "old", Root: "/other"}}}, nil, true},
		{"upgraded transient", nil, []Runtime{{Process: Registration{PID: 8, Birth: "new", Root: "/root"}, Protocol: true}}, nil, true},
		{"inspection denied", nil, nil, errors.New("denied"), false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			probe := &fakeInspection{tt.holders, tt.runtimes, tt.err}
			e := VerifyWriters("/trace", []Registration{me}, probe)
			if (e == nil) != tt.want {
				t.Fatalf("proof=%v", e)
			}
		})
	}
}

type fakeInspection struct {
	holders  []Registration
	runtimes []Runtime
	err      error
}

func (p *fakeInspection) OpenFiles(string) ([]Registration, error) { return p.holders, p.err }
func (p *fakeInspection) Runtimes() ([]Runtime, error)             { return p.runtimes, p.err }

func TestOpenFileInspectionConformance(t *testing.T) {
	if _, e := exec.LookPath("lsof"); e != nil {
		t.Skip("lsof unavailable")
	}
	path := filepath.Join(t.TempDir(), "trace")
	f, e := os.Create(path)
	if e != nil {
		t.Fatal(e)
	}
	defer f.Close()
	holders, e := (OSInspection{}).OpenFiles(path)
	if e != nil {
		t.Fatal(e)
	}
	found := false
	for _, p := range holders {
		if p.PID == os.Getpid() {
			found = p.Birth == procutil.StrictIdentity(strconv.Itoa(os.Getpid()))
		}
	}
	if !found {
		t.Fatalf("current open holder absent %+v", holders)
	}
	f.Close()
	holders, e = (OSInspection{}).OpenFiles(path)
	if e != nil {
		t.Fatal(e)
	}
	for _, p := range holders {
		if p.PID == os.Getpid() {
			t.Fatal("closed fd remains in inspection")
		}
	}
}

func TestUnregisteredChildOpenFileBlocksLegacyExpiry(t *testing.T) {
	if _, e := exec.LookPath("lsof"); e != nil {
		t.Skip("lsof unavailable")
	}
	path := filepath.Join(t.TempDir(), "legacy-trace")
	if e := os.WriteFile(path, []byte("old"), 0600); e != nil {
		t.Fatal(e)
	}
	child := exec.Command("sh", "-c", `exec 3<"$1"; printf 'ready\n'; read done`, "trace-reader", path)
	input, e := child.StdinPipe()
	if e != nil {
		t.Fatal(e)
	}
	output, e := child.StdoutPipe()
	if e != nil {
		t.Fatal(e)
	}
	if e = child.Start(); e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { _ = child.Process.Kill(); _ = child.Wait() })
	ready := make(chan error, 1)
	go func() { _, e := bufio.NewReader(output).ReadString('\n'); ready <- e }()
	select {
	case e := <-ready:
		if e != nil {
			t.Fatal(e)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("reader not ready")
	}
	self := Registration{PID: os.Getpid(), Birth: procutil.StrictIdentity(strconv.Itoa(os.Getpid()))}
	if e = VerifyWriters(path, []Registration{self}, OSInspection{}); !errors.Is(e, ErrUnknownWriters) {
		t.Fatalf("legacy child holder proof %v", e)
	}
	input.Write([]byte("done\n"))
	input.Close()
	if e = child.Wait(); e != nil {
		t.Fatal(e)
	}
	holders, e := (OSInspection{}).OpenFiles(path)
	if e != nil {
		t.Fatal(e)
	}
	for _, p := range holders {
		if p.PID == child.Process.Pid {
			t.Fatal("exited child still reported live")
		}
	}
}
