package couchtty

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"os"
	"os/exec"
	"runtime"
	"testing"
	"time"
)

// Opt-in only: this touches the real clipboard. The operator/hub authorizes
// each invocation; normal suites must never read or replace clipboard contents.
func TestClipboardLiveConsoleConformance(t *testing.T) {
	if os.Getenv("PAIR_LIVE_CLIPBOARD_CONFORMANCE") != "1" {
		t.Skip("explicit clipboard conformance opt-in required")
	}
	if runtime.GOOS != "darwin" || os.Getenv("COUCH_TREE") == "" {
		t.Skip("requires a Darwin Couch-launched process")
	}
	for _, key := range []string{"SSH_CONNECTION", "SSH_CLIENT", "SSH_TTY"} {
		if os.Getenv(key) != "" {
			t.Skip("remote clipboard belongs to the client")
		}
	}
	tool, err := exec.LookPath("pbcopy")
	if err != nil || tool != "/usr/bin/pbcopy" {
		t.Fatal("requires the native /usr/bin/pbcopy in the inherited process PATH")
	}
	read := func() ([]byte, error) {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		// A bounded reader prevents a large pre-existing clipboard from consuming
		// unbounded memory. CommandContext reaps the process on the same deadline.
		cmd := exec.CommandContext(ctx, "/usr/bin/pbpaste")
		var out liveClipboardBuffer
		cmd.Stdout = &out
		err := cmd.Run()
		if out.exceeded {
			return nil, exec.ErrWaitDelay
		}
		return out.data, err
	}
	backup, err := read()
	if err != nil {
		t.Fatal("clipboard backup failed (contents withheld)")
	}
	var nonce [16]byte
	if _, err := rand.Read(nonce[:]); err != nil {
		t.Fatal("marker generation failed")
	}
	marker := []byte("pair clipboard conformance " + hex.EncodeToString(nonce[:]))
	defer func() {
		current, err := read()
		if err != nil {
			t.Error("clipboard restore check failed (contents withheld)")
			return
		}
		if !bytes.Equal(current, marker) {
			return
		} // Never overwrite a later operator copy.
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		cmd := exec.CommandContext(ctx, "/usr/bin/pbcopy")
		cmd.Stdin = bytes.NewReader(backup)
		if cmd.Run() != nil {
			t.Error("clipboard restore failed (contents withheld)")
		}
	}()
	f := newFixtureBeforeRun(t, 24, 80, func(c *Console) { c.feed.Push(Notice{Kind: "status", Body: string(marker)}) })
	_, _ = f.stdin.Write([]byte{0, 0x19})
	waitFor(t, "real Console copy outcome", func() bool { return f.con.menuSnapshot().CopyFeedback.Kind != CopyFeedbackNone })
	if f.con.menuSnapshot().CopyFeedback.Kind != CopyFeedbackSuccess {
		t.Fatal("native Console copy was not confirmed; inspect clipboard_copy metadata")
	}
	got, err := read()
	if err != nil || !bytes.Equal(got, marker) {
		t.Fatal("native clipboard readback did not match (contents withheld)")
	}
}

type liveClipboardBuffer struct {
	data     []byte
	exceeded bool
}

func (b *liveClipboardBuffer) Write(p []byte) (int, error) {
	const maxBackup = 8 << 20
	n := len(p)
	if len(p) > maxBackup-len(b.data) {
		b.exceeded = true
		p = p[:maxBackup-len(b.data)]
	}
	b.data = append(b.data, p...)
	return n, nil
}

func TestLiveClipboardBackupBufferIsBounded(t *testing.T) {
	var b liveClipboardBuffer
	input := bytes.Repeat([]byte("x"), 9<<20)
	n, err := b.Write(input)
	if err != nil || n != len(input) || len(b.data) != 8<<20 || !b.exceeded {
		t.Fatal("backup buffer exceeded bound or stopped draining")
	}
	n, err = b.Write([]byte("tail"))
	if n != 4 || err != nil || len(b.data) != 8<<20 {
		t.Fatal("full backup buffer stopped draining")
	}
}
