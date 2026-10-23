package couchtty

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/xianxu/pair/cmd/internal/webcast"
	"github.com/xianxu/pair/cmd/internal/webcast/drive"
	"github.com/xianxu/pair/cmd/internal/webcast/recording"
)

func noticeClipboard(t *testing.T) func() string {
	t.Helper()
	dir := t.TempDir()
	clipboard := filepath.Join(dir, "clipboard")
	if err := os.WriteFile(filepath.Join(dir, "pbcopy"), []byte("#!/bin/sh\n/bin/cat > \"$TEST_CHROME_CLIPBOARD\"\n"), 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir)
	t.Setenv("TEST_CHROME_CLIPBOARD", clipboard)
	for _, key := range []string{"SSH_CONNECTION", "SSH_CLIENT", "SSH_TTY"} {
		t.Setenv(key, "")
	}
	return func() string {
		t.Helper()
		data, err := os.ReadFile(clipboard)
		if err != nil {
			t.Fatal(err)
		}
		return string(data)
	}
}

func TestChromeCapabilityDiagnosticCopiesFullCurrentText(t *testing.T) {
	copied := noticeClipboard(t)
	var b *cleanupBrowser
	r := newWebcastRig(t, func(r *webcastRig) {
		r.wrap = func(f *webcast.FakeBrowser, _ *webcast.RecordOptions) webcast.Browser {
			d, err := webcast.NewFakeDriven(f, drive.DrivenPage(100), &recording.MemSink{})
			if err != nil {
				t.Fatal(err)
			}
			b = &cleanupBrowser{FakeDriven: d, entered: make(chan struct{}), release: make(chan struct{}), terminal: true}
			return b
		}
	})
	r.open(t)
	r.con.mu.Lock()
	cast := r.con.wcast.cast
	r.con.mu.Unlock()
	if err := cast.SetCapability(context.Background(), webcast.CapabilityAnnotation, true); err != nil {
		t.Fatal(err)
	}
	done := r.con.onWebCapabilityClick(webcast.CapabilityAnnotation, false)
	<-b.entered
	close(b.release)
	<-done
	if !cast.Capabilities().Broken() {
		t.Fatal("capability did not break")
	}
	full := capabilityNotice(cast.Capabilities())
	r.con.setNotice("Unrelated notice")
	r.con.copyLastNotice()
	got := copied()
	if got != full {
		t.Fatalf("copied %q; want full current capability diagnostic %q", got, full)
	}
	for _, held := range []bool{false, true} {
		want := "selected operation error"
		r.con.mu.Lock()
		r.con.menu.Notice = errorMenuNotice(want)
		if held {
			want = "held original copy payload"
			r.con.menu.CopyFeedback = CopyFeedback{Kind: CopyFeedbackFailed, payload: want}
		}
		r.con.mu.Unlock()
		r.con.copyLastNotice()
		got := copied()
		if got != want {
			t.Fatalf("held=%v copied %q; want priority payload %q", held, got, want)
		}
	}
}
