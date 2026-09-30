package couchcore

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"github.com/xianxu/pair/cmd/internal/couchidentity"
	"github.com/xianxu/pair/cmd/internal/launcher"
	"io"
	"strings"
	"testing"
)

// Keep legacy-address fixtures exercising upgrades while production allocates
// exclusively through the durable counter store.
func readerTagAllocator(reader io.Reader) func() (string, error) {
	return func() (string, error) {
		var b [8]byte
		_, err := io.ReadFull(reader, b[:])
		return "couch-" + hex.EncodeToString(b[:]), err
	}
}

type legacyFixtureAllocator struct {
	c        *Couch
	terminal uint64
}

func (a *legacyFixtureAllocator) Allocate(ctx context.Context, r couchidentity.AllocationRequest) (couchidentity.AllocationResult, error) {
	out := couchidentity.AllocationResult{C: 1}
	if r.Conversation {
		tag, err := readerTagAllocator(a.c.Entropy)()
		if err != nil {
			return out, err
		}
		out.PairTag = tag
	}
	if r.Terminal {
		a.terminal++
		out.M = a.terminal
		out.SessionName, _ = couchidentity.FormatSessionName(1, a.terminal)
	}
	return out, nil
}

func managedIntentEnvForTest(t *testing.T, env []string) string {
	t.Helper()
	for _, entry := range env {
		if raw, ok := strings.CutPrefix(entry, launcher.CouchSessionIntentEnv+"="); ok {
			var intent launcher.CouchSessionIntent
			if err := json.Unmarshal([]byte(raw), &intent); err != nil {
				t.Fatal(err)
			}
			if err := intent.Validate(); err != nil {
				t.Fatal(err)
			}
			return entry
		}
	}
	t.Fatal("managed intent missing")
	return ""
}
