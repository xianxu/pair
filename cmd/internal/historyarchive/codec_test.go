package historyarchive

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

const codecFP = "0123456789ABCDEF0123456789ABCDEF01234567"

// The executable fixture reads persistent state shared with its parent; command
// invocations exercise real OS pipes and child lifecycle, without real keys.
func TestCodecProcess(t *testing.T) {
	state := os.Getenv("PAIR_CODEC_FIXTURE")
	if state == "" {
		return
	}
	args := strings.Join(os.Args[1:], " ")
	if !strings.Contains(args, "--batch") && !strings.Contains(args, "-19") {
		os.Exit(90)
	}
	if strings.Contains(args, "--list-keys") {
		b, _ := os.ReadFile(filepath.Join(state, "keys"))
		os.Stdout.Write(b)
		os.Exit(0)
	}
	stage := "zstd"
	if strings.Contains(args, "--encrypt") {
		stage = "gpg"
	}
	os.WriteFile(filepath.Join(state, stage+".started"), []byte("yes"), 0600)
	if _, err := os.Stat(filepath.Join(state, stage+".fail")); err == nil {
		os.Stderr.WriteString(strings.Repeat("failure", 10000))
		os.Exit(7)
	}
	io.Copy(os.Stdout, os.Stdin)
	os.Exit(0)
}
func fixtureCodec(t *testing.T) (CommandCodec, string) {
	t.Helper()
	d := t.TempDir()
	t.Setenv("PAIR_CODEC_FIXTURE", d)
	// Go's test binary accepts our command arguments through a tiny executable
	// wrapper; the production codec itself never uses a shell.
	exe, _ := os.Executable()
	script := filepath.Join(d, "tool")
	if err := os.WriteFile(script, []byte("#!/bin/sh\nexec '"+exe+"' -test.run=TestCodecProcess -- \"$@\"\n"), 0700); err != nil {
		t.Fatal(err)
	}
	return CommandCodec{GPG: script, Zstd: script}, d
}
func TestCodecRecipients(t *testing.T) {
	c, d := fixtureCodec(t)
	for _, tc := range []struct {
		name, key string
		ok        bool
	}{
		{"usable", "pub:u:2048:1:key:1:0:::::e:\nfpr:::::::::" + codecFP + ":\n", true},
		{"missing", "", false},
		{"expired encryption subkey", "pub:u:2048:1:key:1:0:::::sE:\nfpr:::::::::" + codecFP + ":\nsub:e:2048:1:key:1:2:::::e:\nfpr:::::::::AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA:\n", false},
		{"usable encryption subkey", "pub:u:2048:1:key:1:0:::::sE:\nfpr:::::::::" + codecFP + ":\nsub:u:2048:1:key:1:0:::::e:\nfpr:::::::::AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA:\n", true},
		{"expired", "pub:u:2048:1:key:1:2:::::e:\nfpr:::::::::" + codecFP + ":\n", false},
		{"revoked", "pub:r:2048:1:key:1:0:::::e:\nfpr:::::::::" + codecFP + ":\n", false},
		{"signing only", "pub:u:2048:1:key:1:0:::::s:\nfpr:::::::::" + codecFP + ":\n", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			os.WriteFile(filepath.Join(d, "keys"), []byte(tc.key), 0600)
			err := c.CheckRecipients(context.Background(), []string{codecFP})
			if (err == nil) != tc.ok {
				t.Fatalf("error=%v", err)
			}
		})
	}
	if c.CheckRecipients(context.Background(), []string{"email@example.com"}) == nil {
		t.Fatal("accepted email")
	}
	if c.CheckRecipients(context.Background(), nil) == nil {
		t.Fatal("accepted no recipients")
	}
}
func TestCodecStream(t *testing.T) {
	c, _ := fixtureCodec(t)
	var out bytes.Buffer
	input := strings.Repeat("tar", 100000)
	if err := c.Encode(context.Background(), strings.NewReader(input), &out, []string{codecFP}); err != nil {
		t.Fatal(err)
	}
	if out.String() != input {
		t.Fatal("stream differs")
	}
}
func TestCodecFailures(t *testing.T) {
	for _, stage := range []string{"zstd", "gpg", "startup", "writer", "reader", "cancel", "failed stage blocked input"} {
		t.Run(stage, func(t *testing.T) {
			c, d := fixtureCodec(t)
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()
			var in io.Reader = strings.NewReader(strings.Repeat("a", 1<<20))
			var out io.Writer = io.Discard
			switch stage {
			case "zstd", "gpg":
				os.WriteFile(filepath.Join(d, stage+".fail"), nil, 0600)
			case "failed stage blocked input":
				os.WriteFile(filepath.Join(d, "zstd.fail"), nil, 0600)
				r, w := io.Pipe()
				defer w.Close()
				in = r
			case "startup":
				c.Zstd = filepath.Join(d, "absent")
			case "writer":
				out = codecBadWriter{}
			case "reader":
				in = codecBadReader{}
			case "cancel":
				r, w := io.Pipe()
				defer w.Close()
				in = r
				time.AfterFunc(50*time.Millisecond, cancel)
			}
			start := time.Now()
			err := c.Encode(ctx, in, out, []string{codecFP})
			if err == nil {
				t.Fatal("expected failure")
			}
			if len(err.Error()) > 12000 {
				t.Fatal("unbounded diagnostics")
			}
			if time.Since(start) > 2*time.Second {
				t.Fatalf("slow shutdown: %v", err)
			}
		})
	}
}

type codecBadWriter struct{}

func (codecBadWriter) Write([]byte) (int, error) { return 0, errors.New("writer failure") }

type codecBadReader struct{}

func (codecBadReader) Read([]byte) (int, error) { return 0, errors.New("reader failure") }
