package vt

import (
	"bytes"
	"errors"
	uv "github.com/charmbracelet/ultraviolet"
	"github.com/charmbracelet/x/ansi"
	"io"
	"strings"
	"testing"
)

func TestPairLimitsAndEffects(t *testing.T) {
	l := DefaultLimits()
	l.StringBytes = 32
	l.MetadataBytes = 12
	l.GraphemeBytes = 16
	l.KeyboardStack = 2
	e, err := NewEmulatorWithLimits(4, 2, l)
	if err != nil {
		t.Fatal(err)
	}
	defer e.Close()
	var out bytes.Buffer
	e.SetReplyWriter(&out)
	var titles, clips []string
	e.SetCallbacks(Callbacks{Title: func(s string) { titles = append(titles, s) }, ClipboardWrite: func(_ string, b []byte) { clips = append(clips, string(b)) }})
	e.WriteString("\x1b]2;" + strings.Repeat("a", 50) + "\a\x1b]2;ok;yes\a\x1b]52;c;eA==\a")
	if len(titles) != 1 || titles[0] != "ok;yes" || len(clips) != 1 || clips[0] != "x" {
		t.Fatalf("effects %q %q", titles, clips)
	}
	for i := 0; i < 1000; i++ {
		e.WriteString("\x1b[?12345h")
	}
	if len(e.modes) > 100 {
		t.Fatal("unknown modes retained")
	}
	if e.ResizeChecked(1000000, 1000000) == nil || e.Width() != 4 {
		t.Fatal("resize limit")
	}
	if e.Mode(ansi.DECMode(12345)) != ansi.ModeNotRecognized {
		t.Fatal("unknown mode recognized")
	}
}

type brokenWriter struct{}

func (brokenWriter) Write([]byte) (int, error) { return 0, io.ErrClosedPipe }
func TestPairReplyError(t *testing.T) {
	e := NewEmulator(4, 2)
	defer e.Close()
	e.SetReplyWriter(brokenWriter{})
	e.WriteString("\x1b[5n")
	if !errors.Is(e.TakeReplyError(), io.ErrClosedPipe) {
		t.Fatal("lost error")
	}
	if e.TakeReplyError() != nil {
		t.Fatal("error not consumed")
	}
}
func TestPairKeyboardStack(t *testing.T) {
	e := NewEmulator(4, 2)
	defer e.Close()
	var out bytes.Buffer
	e.SetReplyWriter(&out)
	e.WriteString("\x1b[>1u\x1b[>3u\x1b[<u\x1b[?u")
	if out.String() != "\x1b[?1u" {
		t.Fatalf("stack %q", out.String())
	}
	out.Reset()
	e.WriteString("\x1b[=2;2u")
	e.SendKey(uv.KeyReleaseEvent{Code: uv.KeyEnter, Mod: uv.ModCtrl})
	if out.String() != "\x1b[13;5:3u" {
		t.Fatalf("release %q", out.String())
	}
	e.WriteString("\x1bc")
	if e.KeyboardFlags() != 0 {
		t.Fatal("reset keyboard")
	}
}
