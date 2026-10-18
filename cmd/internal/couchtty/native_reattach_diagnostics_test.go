package couchtty

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"

	"github.com/xianxu/pair/cmd/internal/ptychild"
)

const nativeDiagnosticTailBytes = 8192

func nativeReattachFailure(child *ptychild.Child, ready bool) error {
	if !ready && child.Done() {
		return fmt.Errorf("native child exited before reattach frame: exit=%d wire-tail=%q", child.Wait(), nativeChildWireTail(child))
	}
	return nil
}

func nativeChildWireTail(child *ptychild.Child) []byte {
	wire := child.Snapshot()
	if len(wire) > nativeDiagnosticTailBytes {
		wire = wire[len(wire)-nativeDiagnosticTailBytes:]
	}
	return wire
}

func readNativeDiagnosticTail(path string) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("diagnostic path is not a regular file")
	}
	if info.Size() > nativeDiagnosticTailBytes {
		if _, err := f.Seek(-nativeDiagnosticTailBytes, io.SeekEnd); err != nil {
			return nil, err
		}
	}
	return io.ReadAll(io.LimitReader(f, nativeDiagnosticTailBytes))
}

// Observe only files and the owned handles. A session-list/ConnStatus probe
// would introduce another transient client into the ordering under diagnosis.
func logNativeReattachObservation(t *testing.T, phase, dir, session string, child *ptychild.Child, agentPID int) {
	t.Helper()
	clientState := "running"
	if child.Done() {
		clientState = fmt.Sprintf("exited:%d", child.Wait())
	}
	agentState := "unknown"
	if agentPID > 0 {
		agentState = fmt.Sprint(syscall.Kill(agentPID, 0))
	}
	t.Logf("native phase=%s client-pid=%d client=%s owned-agent-pid=%d signal-zero=%s (liveness observation only)", phase, child.PID(), clientState, agentPID, agentState)
	root := filepath.Join(dir, "socket")
	socketDir, err := os.Open(root)
	if err != nil {
		t.Logf("native phase=%s socket-directory: %v", phase, err)
		return
	}
	defer socketDir.Close()
	entries, err := socketDir.ReadDir(8)
	t.Logf("native phase=%s socket-contract-directories=%d read=%v (maximum 8)", phase, len(entries), err)
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		info, err := os.Lstat(filepath.Join(root, entry.Name(), session))
		if err != nil {
			t.Logf("native phase=%s socket=%s/%s: %v", phase, entry.Name(), session, err)
		} else {
			t.Logf("native phase=%s socket=%s/%s mode=%s metadata=%v", phase, entry.Name(), session, info.Mode(), info.Sys())
		}
	}
}

func logNativeServerTail(t *testing.T, dir string) {
	t.Helper()
	path := filepath.Join(dir, fmt.Sprintf("zellij-%d", os.Getuid()), "zellij-log", "zellij.log")
	data, err := readNativeDiagnosticTail(path)
	t.Logf("native private server log tail bytes=%d read=%v body=%q", len(data), err, data)
}

func TestNativeReattachFailureReportsExitedChild(t *testing.T) {
	child := ptychild.NewFakeChild([]byte("No session with the name 'owned' found!"))
	t.Cleanup(func() { _ = child.Close() })
	child.Exit(1)
	<-child.Exited()
	err := nativeReattachFailure(child, false)
	if err == nil || !strings.Contains(err.Error(), "exit=1") || !strings.Contains(err.Error(), "No session") {
		t.Fatalf("missing exited-child evidence: %v", err)
	}
}

func TestNativeReattachFailureKeepsLiveUnknownAndReadyFrame(t *testing.T) {
	child := ptychild.NewFakeChild(nil)
	t.Cleanup(func() { _ = child.Close() })
	if err := nativeReattachFailure(child, false); err != nil {
		t.Fatalf("live child was classified absent: %v", err)
	}
	child.Exit(0)
	<-child.Exited()
	if err := nativeReattachFailure(child, true); err != nil {
		t.Fatalf("already observed frame rejected: %v", err)
	}
}

func TestNativeReattachFailureBoundsLatestWire(t *testing.T) {
	child := ptychild.NewFakeChild(bytes.Repeat([]byte("old"), nativeDiagnosticTailBytes))
	t.Cleanup(func() { _ = child.Close() })
	child.Feed([]byte("latest-exit-reason"))
	child.Exit(1)
	<-child.Exited()
	err := nativeReattachFailure(child, false)
	if err == nil || len(err.Error()) > nativeDiagnosticTailBytes+128 || !strings.HasSuffix(err.Error(), "latest-exit-reason\"") {
		t.Fatalf("exit evidence did not retain bounded latest wire: %v", err)
	}
}

func TestReadNativeDiagnosticTailKeepsBoundedLatestBytes(t *testing.T) {
	path := filepath.Join(t.TempDir(), "log")
	want := bytes.Repeat([]byte("z"), nativeDiagnosticTailBytes)
	if err := os.WriteFile(path, append(bytes.Repeat([]byte("old"), nativeDiagnosticTailBytes), want...), 0600); err != nil {
		t.Fatal(err)
	}
	got, err := readNativeDiagnosticTail(path)
	if err != nil || !bytes.Equal(got, want) {
		t.Fatalf("tail length=%d err=%v", len(got), err)
	}
	if _, err := readNativeDiagnosticTail(path + ".missing"); !os.IsNotExist(err) {
		t.Fatalf("missing log became successful empty evidence: %v", err)
	}
}
