package sessioninventory_test

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/sessioninventory"
	"github.com/xianxu/pair/cmd/internal/sessioninventorytest"
)

func cursorFixture(t *testing.T) (*sessioninventorytest.FakeRuntime, sessioninventory.Artifact, sessioninventory.TranscriptCursor) {
	t.Helper()
	at := time.Now()
	rt, a, _ := proofBackedCodexFixtureWithGeneration(t, "019d1111-1111-7111-8111-111111111111", &at, "")
	inv := sessioninventory.NewIncrementalInventory(rt, sessioninventory.Catalog{})
	v, _, err := inv.ValidateNamedContent(sessioninventory.AgentCodex, "019d1111-1111-7111-8111-111111111111")
	if err != nil {
		t.Fatal(err)
	}
	c, err := sessioninventory.NewTranscriptCursor(rt, v)
	if err != nil {
		t.Fatal(err)
	}
	return rt, a, c
}
func TestTranscriptCursorIdleAndDiscardedCandidate(t *testing.T) {
	rt, a, c := cursorFixture(t)
	key := a.StorageRoot + ":" + a.RelativePath
	reads := rt.OperationCount(sessioninventorytest.OperationReadAt, key)
	stats := rt.OperationCount(sessioninventorytest.OperationStatFile, key)
	_, batch, err := c.Next(context.Background(), rt, 256<<10)
	if err != nil || len(batch.Records) != 0 || batch.BytesRead != 0 || batch.Backlog {
		t.Fatalf("idle=%+v %v", batch, err)
	}
	if rt.OperationCount(sessioninventorytest.OperationReadAt, key) != reads || rt.OperationCount(sessioninventorytest.OperationStatFile, key) != stats+1 {
		t.Fatal("idle reread body or used more than one stat")
	}
	rt.AppendFile(a, []byte("{\"type\":\"event_msg\",\"payload\":{\"type\":\"task_started\"}}\n"), "append")
	candidate, first, err := c.Next(context.Background(), rt, 256<<10)
	if err != nil || len(first.Records) != 1 {
		t.Fatalf("first=%+v %v", first, err)
	}
	_, again, err := c.Next(context.Background(), rt, 256<<10)
	if err != nil || !reflect.DeepEqual(first, again) {
		t.Fatalf("discarded candidate advanced: %+v %v", again, err)
	}
	_, idle, err := candidate.Next(context.Background(), rt, 256<<10)
	if err != nil || idle.BytesRead != 0 {
		t.Fatalf("committed candidate replayed: %+v %v", idle, err)
	}
}
func TestTranscriptCursorBoundedSnapshotAndPartial(t *testing.T) {
	fake, a, c := cursorFixture(t)
	rt := &titleObservedRuntime{FakeRuntime: fake, transcript: a}
	fake.AppendFile(a, []byte(strings.Repeat("x", 600<<10)), "partial")
	entry, _ := fake.StatFile(a)
	appended := 0
	rt.afterRead = func(offset, limit int64) {
		if offset >= entry.Size-(600<<10) {
			appended++
			fake.AppendFile(a, []byte("x"), "racing")
		}
	}
	candidate, batch, err := c.Next(context.Background(), rt, 256<<10)
	if err != nil || batch.BytesRead != 256<<10 || !batch.Backlog || len(batch.Records) != 0 {
		t.Fatalf("batch=%+v %v", batch, err)
	}
	payload := 0
	for _, r := range rt.reads {
		if r.limit == 64<<10 {
			payload++
		}
	}
	if payload != 4 {
		t.Fatalf("payload chunks=%d reads=%v", payload, rt.reads)
	}
	rt.afterRead = nil
	fake.AppendFile(a, []byte("\n"), "finish")
	var records []sessioninventory.FramedJSONLRecord
	for i := 0; i < 4; i++ {
		next, b, err := candidate.Next(context.Background(), rt, 256<<10)
		if err != nil {
			t.Fatal(err)
		}
		candidate = next
		records = append(records, b.Records...)
		if !b.Backlog {
			break
		}
	}
	if len(records) != 1 || len(records[0].Bytes) != 600<<10+appended || records[0].Offset != entry.Size-(600<<10) {
		t.Fatalf("partial result count=%d", len(records))
	}
}
func TestTranscriptCursorCancellationAndMutationRollback(t *testing.T) {
	for _, mode := range []string{"cancel", "replace", "truncate", "same-size"} {
		t.Run(mode, func(t *testing.T) {
			fake, a, c := cursorFixture(t)
			fake.AppendFile(a, []byte("{}\n"), "append")
			rt := &titleObservedRuntime{FakeRuntime: fake, transcript: a}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			rt.afterRead = func(offset, limit int64) {
				rt.afterRead = nil
				entry, _ := fake.StatFile(a)
				switch mode {
				case "cancel":
					cancel()
				case "replace":
					fake.ReplaceFile(entry, []byte("{}\n"), "new", "changed")
				case "truncate":
					fake.TruncateFile(a, 0, "changed")
				case "same-size":
					raw, _ := fake.ReadFile(a, -1)
					raw[0] = ' '
					fake.PutFile(sessioninventory.FileEntry{Artifact: a, StableFileID: entry.StableFileID, GenerationToken: entry.GenerationToken, MutationToken: "changed", ModTime: entry.ModTime}, raw)
				}
			}
			original, batch, err := c.Next(ctx, rt, 256<<10)
			if err == nil || batch.BytesRead != 0 || len(batch.Records) != 0 || !reflect.DeepEqual(original, c) {
				t.Fatalf("mutation committed: %+v %v", batch, err)
			}
			if mode == "cancel" {
				if !errors.Is(err, context.Canceled) {
					t.Fatal(err)
				}
				_, again, err := c.Next(context.Background(), fake, 256<<10)
				if err != nil || len(again.Records) != 1 {
					t.Fatalf("cancel retry=%+v %v", again, err)
				}
			}
		})
	}
}

func TestTranscriptCursorPartialCancellationRetainsPrefix(t *testing.T) {
	fake, a, c := cursorFixture(t)
	fake.AppendFile(a, []byte(strings.Repeat("p", 180<<10)), "partial")
	partial, first, err := c.Next(context.Background(), fake, 180<<10)
	if err != nil || first.BytesRead != 180<<10 || len(first.Records) != 0 {
		t.Fatalf("first=%+v %v", first, err)
	}
	fake.AppendFile(a, []byte(strings.Repeat("q", 180<<10)+"\n"), "finish")
	rt := &titleObservedRuntime{FakeRuntime: fake, transcript: a}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	payloads := 0
	rt.afterRead = func(offset, limit int64) {
		if limit == 64<<10 {
			payloads++
			if payloads == 2 {
				cancel()
			}
		}
	}
	unchanged, batch, err := partial.Next(ctx, rt, 256<<10)
	if !errors.Is(err, context.Canceled) || !reflect.DeepEqual(unchanged, partial) || len(batch.Records) != 0 {
		t.Fatalf("canceled=%+v %v", batch, err)
	}
	_, again, err := partial.Next(context.Background(), fake, 256<<10)
	if err != nil || len(again.Records) != 1 || string(again.Records[0].Bytes) != strings.Repeat("p", 180<<10)+strings.Repeat("q", 180<<10) {
		t.Fatalf("retry records=%d err=%v", len(again.Records), err)
	}
}

func TestTranscriptCursorIdentityAndMetadataGuards(t *testing.T) {
	for _, mode := range []string{"same-size", "birthtime", "generation", "boundary", "paired-payload", "paired-boundary"} {
		t.Run(mode, func(t *testing.T) {
			fake, a, c := cursorFixture(t)
			rt := &titleObservedRuntime{FakeRuntime: fake, transcript: a}
			mutate := func() {
				entry, _ := fake.StatFile(a)
				raw, _ := fake.ReadFile(a, -1)
				switch mode {
				case "birthtime":
					at := time.Now().Add(time.Hour)
					entry.BirthTime = &at
				case "generation":
					entry.GenerationToken = "new-generation"
				default:
					raw[0] = ' '
					entry.MutationToken = "rewrite"
				}
				fake.PutFile(entry, raw)
			}
			if mode == "same-size" || mode == "birthtime" || mode == "generation" {
				mutate()
			} else {
				fake.AppendFile(a, []byte("{}\n"), "append")
				switch mode {
				case "boundary":
					mutate()
				case "paired-boundary":
					rt.afterRead = func(offset, limit int64) { rt.afterRead = nil; mutate() }
				case "paired-payload":
					reads := 0
					rt.afterRead = func(offset, limit int64) {
						reads++
						if reads == 2 {
							mutate()
						}
					}
				}
			}
			original, batch, err := c.Next(context.Background(), rt, 256<<10)
			if !errors.Is(err, sessioninventory.ErrArtifactChanged) || !reflect.DeepEqual(original, c) || batch.BytesRead != 0 || len(batch.Records) != 0 {
				t.Fatalf("guard=%+v %v", batch, err)
			}
		})
	}
}

func TestTranscriptCursorOversizedPartialFailsWithoutAdvancing(t *testing.T) {
	rt, a, c := cursorFixture(t)
	rt.AppendFile(a, []byte(strings.Repeat("x", (32<<20)+1)), "oversize")
	for pass := 0; pass < 129; pass++ {
		next, batch, err := c.Next(context.Background(), rt, 256<<10)
		if pass == 128 {
			if !errors.Is(err, sessioninventory.ErrReadLimit) || !reflect.DeepEqual(next, c) || batch.BytesRead != 0 || len(batch.Records) != 0 {
				t.Fatalf("oversize=%+v %v", batch, err)
			}
			return
		}
		if err != nil || len(batch.Records) != 0 || batch.BytesRead != 256<<10 {
			t.Fatalf("pass=%d batch=%+v err=%v", pass, batch, err)
		}
		c = next
	}
	t.Fatal("oversize accepted")
}
