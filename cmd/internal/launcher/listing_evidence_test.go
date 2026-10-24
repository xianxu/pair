package launcher

import (
	"context"
	"crypto/sha256"
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestListingEvidenceKeepsExactBoundedBytes(t *testing.T) {
	at := time.Unix(1000, 123)
	for _, size := range []int{0, 8192, 9000} {
		raw := []byte(strings.Repeat("x", size))
		want := append([]byte(nil), raw...)
		got := newListingEvidence(raw, at)
		if got.At != at || got.Bytes != size || got.SHA256 != fmt.Sprintf("%x", sha256.Sum256(want)) || got.Truncated != (size > 8192) || string(got.Raw) != string(want[:min(size, 8192)]) {
			t.Fatalf("size %d: %+v", size, got)
		}
		if size > 0 {
			raw[0] = 'z'
			if got.Raw[0] != 'x' {
				t.Fatal("diagnostic aliases the source bytes")
			}
		}
	}
}

func TestListingEvidenceUsesTheJudgedPollSnapshot(t *testing.T) {
	for _, n := range []int{8, 16} {
		t.Run(fmt.Sprint(n), func(t *testing.T) {
			now := time.Unix(1000, 0)
			var rows strings.Builder
			for i := range n {
				fmt.Fprintf(&rows, "pair-%d [Created 1s ago]\n", i)
			}
			f := &fakeListings{raw: rows.String()}
			p := testPoller(f, &now)
			p.Refresh(context.Background())
			source := ZellijSource{Path: "/nonexistent/zellij", Listings: p}
			sessions, before, err := source.LivenessEvidenceContext(t.Context())
			if err != nil || len(sessions) != n || before.At != now || string(before.Raw) != rows.String() {
				t.Fatalf("judged snapshot: sessions=%d evidence=%+v err=%v", len(sessions), before, err)
			}
			for range 2 * n {
				if _, _, err := source.LivenessEvidenceContext(t.Context()); err != nil {
					t.Fatal(err)
				}
			}
			if f.calls != 1 {
				t.Fatalf("evidence reads spawned %d queries, want one refresh", f.calls)
			}
			now = now.Add(time.Second)
			f.raw = "pair-restored [Created 1s ago]\n"
			p.Refresh(t.Context())
			if string(before.Raw) != rows.String() || before.At != time.Unix(1000, 0) {
				t.Fatal("later refresh rewrote the judged evidence")
			}
		})
	}
}
