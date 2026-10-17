package couchcmd

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/clipcmd"
	"github.com/xianxu/pair/cmd/internal/couchtty"
	"github.com/xianxu/pair/cmd/internal/hostty"
	"github.com/xianxu/pair/cmd/internal/ptychild"
)

func TestClipboardCopyFactProjectsOnlyContentFreeMetadata(t *testing.T) {
	exit := 7
	result := clipcmd.ClipboardCopyResult{
		Metadata: clipcmd.ClipboardCopyMetadata{Tool: "pbcopy", Executable: "/usr/bin/pbcopy", Status: "exit", Started: true, ExitCode: &exit, TimedOut: true, StderrBytes: 12000, StderrHash: "0123456789abcdef", StderrTruncated: true},
		Stderr:   "private stderr must remain on screen",
	}
	observation := couchtty.CopyObservation{Outcome: "unconfirmed", Native: result.Metadata, OSC: "unconfirmed"}
	fact := clipboardCopyFact(observation)
	if fact.Event != "clipboard_copy" || fact.Clipboard == nil || !reflect.DeepEqual(*fact.Clipboard, observation) {
		t.Fatalf("projection=%+v", fact)
	}
	raw, err := encodeFact(fact)
	if err != nil {
		t.Fatal(err)
	}
	var got map[string]any
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatal(err)
	}
	clipboard := got["clipboard"].(map[string]any)
	if len(clipboard) != 3 {
		t.Fatalf("unexpected observation fields: %v", clipboard)
	}
	native := clipboard["native"].(map[string]any)
	wantKeys := map[string]bool{"tool": true, "executable": true, "status": true, "started": true, "exit_code": true, "timeout": true, "canceled": true, "stderr_bytes": true, "stderr_hash": true, "stderr_truncated": true}
	for key := range native {
		if !wantKeys[key] {
			t.Fatalf("unapproved native field %q", key)
		}
	}
	if len(native) != len(wantKeys) || native["stderr_bytes"] != float64(12000) || native["stderr_hash"] != "0123456789abcdef" || native["exit_code"] != float64(7) || native["timeout"] != true {
		t.Fatalf("lost typed evidence: %v", native)
	}
	if strings.Contains(string(raw), result.Stderr) {
		t.Fatal("free-form stderr entered supervision")
	}
	// The projection owns a value, not the caller's mutable observation struct.
	observation.Outcome = "changed"
	if fact.Clipboard.Outcome != "unconfirmed" {
		t.Fatal("projection aliases caller observation")
	}
}

func TestClipboardCopyObserverUsesExistingFactWriter(t *testing.T) {
	sink := &factSink{}
	facts := testFactLog(t, sink)
	facts.clipboardCopy(couchtty.CopyObservation{Outcome: "empty", OSC: "not-requested"})
	supervisionEventually(t, "copy fact not written", func() bool { return len(sink.rows(t)) == 1 })
	row := sink.rows(t)[0]
	if row.Event != "clipboard_copy" || row.Version != 1 || row.Seq != 1 || row.Generation == "" || row.Clipboard == nil || row.Clipboard.Outcome != "empty" {
		t.Fatalf("copy bypassed standard writer metadata: %+v", row)
	}
	facts.Close()
	facts.clipboardCopy(couchtty.CopyObservation{Outcome: "empty"})
	if len(sink.rows(t)) != 1 {
		t.Fatal("closed writer admitted copy fact")
	}
	var absent *supervisionFactLog
	absent.clipboardCopy(couchtty.CopyObservation{Outcome: "empty"})
}

func TestClipboardCopyConsoleKeysReachContentFreeFactLog(t *testing.T) {
	toolDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(toolDir, "pbcopy"), []byte("#!/bin/sh\nprintf 'private tool stderr' >&2\nexit 7\n"), 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", toolDir)
	for _, key := range []string{"SSH_CONNECTION", "SSH_CLIENT", "SSH_TTY"} {
		t.Setenv(key, "")
	}
	sink := &factSink{}
	facts := testFactLog(t, sink)
	host := hostty.NewFakeHost(ptychild.Size{Rows: 24, Cols: 80})
	reader, writer := io.Pipe()
	console := couchtty.New(host, reader)
	console.SetCopyObserver(facts.clipboardCopy)
	console.Notify(couchtty.Notice{Kind: "status", Body: "private original notice"})
	done := make(chan struct{})
	go func() { console.Run(); close(done) }()
	t.Cleanup(func() {
		console.Stop()
		_ = writer.Close()
		_ = reader.Close()
		select {
		case <-done:
		case <-time.After(3 * time.Second):
			t.Error("Console did not stop")
		}
	})
	supervisionEventually(t, "Console not ready", func() bool { return host.RawDepth() > 0 && strings.Contains(host.Written(), "Ctrl+Y") })
	if _, err := writer.Write([]byte{0x19}); err != nil {
		t.Fatal(err)
	}
	supervisionEventually(t, "key did not reach fact writer", func() bool { return len(sink.rows(t)) == 1 })
	row := sink.rows(t)[0]
	if row.Clipboard == nil {
		t.Fatalf("no clipboard observation: %+v", row)
	}
	observation := *row.Clipboard
	if row.Event != "clipboard_copy" || observation.Outcome != "unconfirmed" || observation.OSC != "unconfirmed" || observation.Native.Tool != "pbcopy" || observation.Native.ExitCode == nil || *observation.Native.ExitCode != 7 || observation.Native.StderrBytes != int64(len("private tool stderr")) || observation.Native.StderrHash == "" {
		t.Fatalf("missing real execution evidence: %+v", row)
	}
	raw, err := encodeFact(row)
	if err != nil {
		t.Fatal(err)
	}
	for _, private := range []string{"private original notice", "private tool stderr"} {
		if strings.Contains(string(raw), private) {
			t.Fatal("private content reached copy log")
		}
	}
	supervisionEventually(t, "operator screen lost bounded tool evidence", func() bool {
		// The Presenter writes changed terminal cells in separate cursor-addressed
		// runs, so multiword text need not be contiguous in the byte stream.
		return strings.Contains(host.Written(), "stderr;")
	})
}
