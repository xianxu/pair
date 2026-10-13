package couchcmd

import (
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"

	core "github.com/xianxu/pair/cmd/internal/submission"
)

func TestLoadOriginPolicy(t *testing.T) {
	for _, tc := range []struct{ name, content, reason string }{
		{"valid", `{"origins":[{"origin":"https://example.com","privacy":"shareable"}]}`, ""},
		{"case alias", `{"origins":[{"origin":"https://example.com","privacy":"private","Privacy":"shareable"}]}`, "invalid JSON"},
		{"envelope alias", `{"origins":[],"Origins":[{"origin":"https://example.com","privacy":"shareable"}]}`, "invalid JSON"},
		{"invalid", `{"origins":[{"secret":"do-not-echo"}]}`, "invalid JSON"},
		{"oversized", strings.Repeat("x", core.MaxOriginPolicyBytes+1), "file exceeds 64 KiB"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "origin-privacy.json")
			if err := os.WriteFile(path, []byte(tc.content), 0600); err != nil {
				t.Fatal(err)
			}
			policy, notice := loadOriginPolicy(path)
			d := policy.Decide("https://example.com/page?secret=hidden")
			if tc.reason == "" {
				if notice != "" || d.Privacy != core.Shareable {
					t.Fatalf("decision=%+v notice=%q", d, notice)
				}
			} else if d.Privacy != core.Private || d.PrivacyRule.Invalid != tc.reason || !strings.Contains(notice, path) || !strings.Contains(notice, "private") || strings.Contains(notice, "do-not-echo") {
				t.Fatalf("decision=%+v notice=%q", d, notice)
			}
		})
	}
}

func TestLoadOriginPolicyMissingCreatesNothing(t *testing.T) {
	path := filepath.Join(t.TempDir(), "absent", "origin-privacy.json")
	policy, notice := loadOriginPolicy(path)
	d := policy.Decide("https://example.com")
	if notice != "" || d.Privacy != core.Private || d.PrivacyRule.Invalid != "" {
		t.Fatalf("decision=%+v notice=%q", d, notice)
	}
	if _, err := os.Stat(filepath.Dir(path)); !os.IsNotExist(err) {
		t.Fatalf("created parent: %v", err)
	}
}

func TestLoadOriginPolicyNonregularAndUnreadable(t *testing.T) {
	dir := t.TempDir()
	fifo := filepath.Join(dir, "fifo")
	if err := syscall.Mkfifo(fifo, 0600); err != nil {
		t.Fatal(err)
	}
	loop := filepath.Join(dir, "loop")
	if err := os.Symlink(loop, loop); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct{ path, reason string }{{dir, "not a regular file"}, {fifo, "not a regular file"}, {loop, "cannot read file"}} {
		policy, notice := loadOriginPolicy(tc.path)
		d := policy.Decide("https://example.com")
		if d.Privacy != core.Private || d.PrivacyRule.Invalid != tc.reason || !strings.Contains(notice, tc.path) {
			t.Fatalf("path=%q decision=%+v notice=%q", tc.path, d, notice)
		}
	}
}

func TestWebcastOriginPolicyReloads(t *testing.T) {
	store := t.TempDir()
	cfg, err := webcastSettings(func(string) string { return "" }, store, nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.ReadOriginPolicy == nil {
		t.Fatal("missing per-submission policy reader")
	}
	path := filepath.Join(store, "webcast", "origin-privacy.json")
	if _, err := os.Stat(filepath.Dir(path)); !os.IsNotExist(err) {
		t.Fatalf("settings created policy directory: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		t.Fatal(err)
	}
	for _, privacy := range []core.Privacy{core.Shareable, core.Private} {
		if err := os.WriteFile(path, []byte(`{"origins":[{"origin":"https://example.com","privacy":"`+string(privacy)+`"}]}`), 0600); err != nil {
			t.Fatal(err)
		}
		policy, notice := cfg.ReadOriginPolicy()
		if d := policy.Decide("https://example.com"); d.Privacy != privacy || notice != "" {
			t.Fatalf("decision=%+v notice=%q", d, notice)
		}
	}
	cfg, err = webcastSettings(func(string) string { return "" }, "", nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.ReadOriginPolicy != nil {
		policy, notice := cfg.ReadOriginPolicy()
		if policy.Decide("https://example.com").Privacy != core.Private || notice != "" {
			t.Fatalf("empty store grants sharing or warns: %q", notice)
		}
	}
}
