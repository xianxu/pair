package couchcore

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/artifactpath"
)

func TestOSSlugReaderReadsTheSuggestionUnfenced(t *testing.T) {
	data := t.TempDir()
	address := validThreadRecord(t).Address
	paths, err := artifactpath.Resolve(artifactpath.Address{DataDir: data, RepoScope: address.RepoScope, Tag: string(address.Tag)})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(paths.ScopeDir(), 0700); err != nil {
		t.Fatal(err)
	}
	reader := OSSlugReader{DataDir: data}
	read := func() string {
		t.Helper()
		got, err := reader.Read(context.Background(), address)
		if err != nil {
			t.Fatal(err)
		}
		return got
	}
	if got := read(); got != "" {
		t.Fatalf("missing suggestion = %q, want empty", got)
	}
	// The draft-mirrored slug-<tag> carries the operator's edits; it is never read.
	if err := os.WriteFile(paths.Slug(), []byte("=== main | operator edit ===\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if got := read(); got != "" {
		t.Fatalf("read the draft mirror: %q", got)
	}
	if err := os.WriteFile(paths.SlugProposed(), []byte("=== main-slot3 | couch switcher click-select ===\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if got := read(); got != "main-slot3 | couch switcher click-select" {
		t.Fatalf("suggestion = %q", got)
	}
	if err := os.WriteFile(paths.SlugProposed(), []byte("KEEP\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if got := read(); got != "" {
		t.Fatalf("malformed suggestion = %q, want empty", got)
	}
	if err := os.WriteFile(paths.SlugProposed(), []byte("=== a | "+strings.Repeat("x", slugMaxBytes)+" ===\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := reader.Read(context.Background(), address); err == nil {
		t.Fatal("oversized suggestion read without error")
	}
}

func TestApplySlugsReadsLiveRowsOnly(t *testing.T) {
	live := ActionableThreadSummary{Address: ThreadAddress{RepoScope: "816fc349d3faebf8", Tag: "couch-0000000000000001"}, State: ThreadLive}
	parked := ActionableThreadSummary{Address: ThreadAddress{RepoScope: "816fc349d3faebf8", Tag: "couch-0000000000000002"}, State: ThreadParked, Slug: "stale"}
	failing := ActionableThreadSummary{Address: ThreadAddress{RepoScope: "816fc349d3faebf8", Tag: "couch-0000000000000003"}, State: ThreadLive}
	var asked []ThreadAddress
	rows := ApplySlugs(context.Background(), []ActionableThreadSummary{live, parked, failing}, func(_ context.Context, address ThreadAddress) (string, error) {
		asked = append(asked, address)
		if address == failing.Address {
			return "ignored", errors.New("unreadable")
		}
		return "main | busy", nil
	})
	if rows[0].Slug != "main | busy" || rows[1].Slug != "" || rows[2].Slug != "" {
		t.Fatalf("slugs = %q %q %q", rows[0].Slug, rows[1].Slug, rows[2].Slug)
	}
	if len(asked) != 2 || asked[0] != live.Address || asked[1] != failing.Address {
		t.Fatalf("reader asked for %v, want only the live rows", asked)
	}
	if rows := ApplySlugs(context.Background(), []ActionableThreadSummary{live}, nil); rows[0].Slug != "" {
		t.Fatal("nil reader produced a slug")
	}
}
