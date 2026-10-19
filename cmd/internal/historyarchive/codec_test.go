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
	if !strings.Contains(args, "--batch") && !strings.Contains(args, "-T0") {
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
	os.WriteFile(filepath.Join(state, stage+".started"), []byte(args), 0600)
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
	if err := c.Encode(context.Background(), strings.NewReader(input), &out, []string{codecFP}, "zstd -19"); err != nil {
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
			err := c.Encode(ctx, in, out, []string{codecFP}, "zstd -19")
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

func TestCodecUsesAllThreads(t *testing.T) {
	c, d := fixtureCodec(t)
	if err := c.Encode(context.Background(), strings.NewReader("tar"), io.Discard, []string{codecFP}, "zstd -19"); err != nil {
		t.Fatal(err)
	}
	args, err := os.ReadFile(filepath.Join(d, "zstd.started"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(args), "-T0") || strings.Contains(string(args), "--single-thread") {
		t.Fatalf("zstd argv: %s", args)
	}
}

func TestCodecDescriptorArgumentsAndEarlyRefusal(t *testing.T) {
	for _, descriptor := range []string{"zstd -1", "zstd -9", "zstd -19", "zstd -20", "zstd -22", "zstd -23", "zstd -09", "gzip -9"} {
		t.Run(descriptor, func(t *testing.T) {
			c, d := fixtureCodec(t)
			err := c.Encode(context.Background(), strings.NewReader("tar"), io.Discard, []string{codecFP}, descriptor)
			level, parseErr := compressionLevel(descriptor)
			supported := parseErr == nil && level <= 22
			if (err == nil) != supported {
				t.Fatalf("error=%v supported=%v", err, supported)
			}
			args, readErr := os.ReadFile(filepath.Join(d, "zstd.started"))
			if !supported {
				if readErr == nil {
					t.Fatal("invalid descriptor spawned zstd")
				}
				if _, err := os.Stat(filepath.Join(d, "gpg.started")); !os.IsNotExist(err) {
					t.Fatal("invalid descriptor spawned gpg")
				}
				return
			}
			if readErr != nil {
				t.Fatal(readErr)
			}
			if !strings.Contains(string(args), "-- "+strings.TrimPrefix(descriptor, "zstd ")+" -T0 ") || strings.Contains(string(args), "--ultra") != (level > 19) {
				t.Fatalf("argv=%s", args)
			}
		})
	}
}
