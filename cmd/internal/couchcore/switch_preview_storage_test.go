package couchcore

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/storagegc"
)

// Exercise the form/CLI preparation boundary, rather than only the storage
// helper: opening or cancelling an offer must never finish a lifecycle write.
func TestSwitchPreparationPreservesPendingJournals(t *testing.T) {
	for _, kind := range []string{"global", "local", "routed-local"} {
		for _, preparation := range []string{"form", "known", "force"} {
			t.Run(kind+"/"+preparation, func(t *testing.T) {
				store, backend, record := previewThreadFixture(t, kind)
				interrupted := errors.New("interrupted lifecycle transaction")
				backend.hooks.AfterJournal = func() error { return interrupted }
				if _, err := backend.updateExistingThread(record.Address, record.Revision, func(next *ThreadRecord) error {
					next.Description = "must remain pending until an explicit lifecycle action"
					return nil
				}); !errors.Is(err, interrupted) {
					t.Fatal(err)
				}
				before, err := snapshotDirectoryForTest(store.namespace.Dir())
				if err != nil {
					t.Fatal(err)
				}
				backendBefore, err := snapshotDirectoryForTest(backend.root)
				if err != nil {
					t.Fatal(err)
				}
				c := &Couch{Threads: store, Runner: NewFakeRunner()}
				prepare := func(ctx context.Context) error {
					var err error
					switch preparation {
					case "form":
						_, err = c.PrepareAgentSwitch(ctx, record.Address, "codex", nil)
					case "known":
						_, err = c.PrepareKnownAgentSwitch(ctx, record.Address, "codex")
					case "force":
						_, err = c.PrepareForcedAgentSwitch(ctx, record.Address, "codex", nil)
					}
					return err
				}
				if err := prepare(context.Background()); err == nil || !strings.Contains(err.Error(), "recovery pending") {
					t.Errorf("preparation must refuse pending recovery: %v", err)
				}
				ctx, cancel := context.WithCancel(context.Background())
				cancel()
				if err := prepare(ctx); !errors.Is(err, context.Canceled) {
					t.Errorf("cancelled preparation: %v", err)
				}
				after, err := snapshotDirectoryForTest(store.namespace.Dir())
				if err != nil {
					t.Fatal(err)
				}
				if !reflect.DeepEqual(before, after) {
					t.Fatal("opening/cancelling switch preparation changed stored bytes or removed the pending journal")
				}
				backendAfter, err := snapshotDirectoryForTest(backend.root)
				if err != nil || !reflect.DeepEqual(backendBefore, backendAfter) {
					t.Fatalf("opening/cancelling preparation changed routed backend bytes: %v", err)
				}
			})
		}
	}
}

func TestSwitchPreparationDoesNotInitializeOrEnrollRetention(t *testing.T) {
	for _, preparation := range []string{"form", "known", "force"} {
		t.Run(preparation, func(t *testing.T) {
			store, _ := newTestThreadStore(t)
			coordinator, err := storagegc.NewCoordinator(t.TempDir())
			if err != nil {
				t.Fatal(err)
			}
			store.coordinator = coordinator
			before, err := snapshotDirectoryForTest(store.namespace.Dir())
			if err != nil {
				t.Fatal(err)
			}
			retentionBefore, err := snapshotDirectoryForTest(coordinator.Root)
			if err != nil {
				t.Fatal(err)
			}
			c := &Couch{Threads: store, Runner: NewFakeRunner()}
			address := validThreadRecord(t).Address
			switch preparation {
			case "form":
				_, err = c.PrepareAgentSwitch(context.Background(), address, "codex", nil)
			case "known":
				_, err = c.PrepareKnownAgentSwitch(context.Background(), address, "codex")
			case "force":
				_, err = c.PrepareForcedAgentSwitch(context.Background(), address, "codex", nil)
			}
			if !errors.Is(err, ErrThreadNotFound) {
				t.Fatalf("missing source: %v", err)
			}
			after, err := snapshotDirectoryForTest(store.namespace.Dir())
			if err != nil {
				t.Fatal(err)
			}
			retentionAfter, err := snapshotDirectoryForTest(coordinator.Root)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(before, after) || !reflect.DeepEqual(retentionBefore, retentionAfter) {
				t.Fatal("preparation initialized storage or enrolled retention")
			}
		})
	}
}
