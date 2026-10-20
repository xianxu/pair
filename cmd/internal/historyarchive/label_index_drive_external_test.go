package historyarchive_test

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	ha "github.com/xianxu/pair/cmd/internal/historyarchive"
	"github.com/xianxu/pair/cmd/internal/historyarchive/archivetest"
)

type indexInventoryRunner struct {
	fake  *archivetest.FakeRclone
	alter func([]ha.RemoteEntry) []ha.RemoteEntry
}

func (r indexInventoryRunner) Run(ctx context.Context, args []string, limit int) ([]byte, error) {
	raw, err := r.fake.Run(ctx, args, limit)
	if err != nil || args[0] != "lsjson" {
		return raw, err
	}
	var entries []ha.RemoteEntry
	if err = json.Unmarshal(raw, &entries); err != nil {
		return nil, err
	}
	return json.Marshal(r.alter(entries))
}

var indexSyncTime = time.Date(2030, 1, 2, 3, 4, 5, 0, time.UTC)

func uploadedIndexFixture(t *testing.T, n int, keepReceipts bool) (ha.ContinuousConfig, []ha.CatalogRecord, *archivetest.FakeRclone) {
	t.Helper()
	cfg, records, root := driveFixture(t, n)
	fake := &archivetest.FakeRclone{}
	report, err := (ha.DriveMirror{Rclone: ha.Rclone{Runner: fake}}).Run(context.Background(), cfg, filepath.Join(root, "archive"))
	if err != nil || report.Verified != n {
		t.Fatalf("upload: %+v %v", report, err)
	}
	if err = os.RemoveAll(cfg.Sink); err != nil {
		t.Fatal(err)
	}
	if err = os.Remove(cfg.Catalog); err != nil {
		t.Fatal(err)
	}
	if !keepReceipts {
		if err = os.RemoveAll(cfg.Catalog + ".uploads"); err != nil {
			t.Fatal(err)
		}
	}
	fake.Invocations = 0
	fake.Lists = 0
	fake.Downloads = 0
	fake.HashBatches = 0
	return cfg, records, fake
}
func runIndexSync(ctx context.Context, cfg ha.ContinuousConfig, fake *archivetest.FakeRclone) (ha.IndexSyncReport, error) {
	return (ha.DriveIndexSync{Rclone: ha.Rclone{Runner: fake}, Now: func() time.Time { return indexSyncTime }}).Run(ctx, cfg)
}

func TestDriveIndexSyncRemoteOnlyBoundedCalls(t *testing.T) {
	for _, n := range []int{8, 16} {
		t.Run(fmt.Sprint(n), func(t *testing.T) {
			cfg, records, fake := uploadedIndexFixture(t, n, false)
			report, err := runIndexSync(context.Background(), cfg, fake)
			if err != nil || report.LabelCount != n || report.Verified != n || report.Observed != 0 {
				t.Fatalf("sync: %+v %v", report, err)
			}
			idx, err := ha.ReadLabelIndex(cfg.Catalog + ".index.json")
			if err != nil || len(idx.Records) != n {
				t.Fatalf("index: %d %v", len(idx.Records), err)
			}
			for _, record := range records {
				r, ok := idx.Records[record.Label.ID]
				if !ok {
					t.Fatal("missing original ID")
				}
				for _, location := range r.Locations {
					if !location.Available || location.Proof.Method != "backend-sha256" {
						t.Fatalf("location: %+v", location)
					}
				}
			}
			if fake.Invocations > 4 || fake.Lists != 2 || fake.DownloadBatches != 1 || fake.DownloadedCipherBytes != 0 || fake.HashBatches != 0 {
				t.Fatalf("unbounded/network ciphertext work: %+v", fake)
			}
			for _, p := range []string{cfg.Catalog, cfg.Sink, cfg.Catalog + ".uploads"} {
				if _, err = os.Stat(p); !os.IsNotExist(err) {
					t.Fatalf("manufactured local authority %s: %v", p, err)
				}
			}
		})
	}
}

func TestDriveIndexSyncObservedWithoutHashesAndReceiptProof(t *testing.T) {
	for _, receipts := range []bool{false, true} {
		t.Run(fmt.Sprint(receipts), func(t *testing.T) {
			cfg, _, fake := uploadedIndexFixture(t, 1, receipts)
			fake.MissingHashes = true
			report, err := runIndexSync(context.Background(), cfg, fake)
			if err != nil {
				t.Fatal(err)
			}
			idx, err := ha.ReadLabelIndex(cfg.Catalog + ".index.json")
			if err != nil {
				t.Fatal(err)
			}
			for _, record := range idx.Records {
				for _, loc := range record.Locations {
					if receipts && (loc.Evidence != "verified" || loc.Proof.Method != "upload-receipt" || report.Verified != 1) {
						t.Fatalf("receipt proof missing: %+v %+v", loc, report)
					}
					if !receipts && (loc.Evidence != "observed" || loc.Proof.Method != "" || report.Observed != 1) {
						t.Fatalf("invented proof: %+v %+v", loc, report)
					}
				}
			}
			if fake.HashBatches != 0 || fake.DownloadedCipherBytes != 0 {
				t.Fatal("sync downloaded ciphertext/checksum fallback")
			}
		})
	}
}

func TestDriveIndexSyncRefusalsRetainSnapshot(t *testing.T) {
	for _, kind := range []string{"duplicate", "orphan", "corrupt", "changed", "new-pair", "partial", "oversize", "timeout", "bundle-sha", "receipt-id"} {
		t.Run(kind, func(t *testing.T) {
			cfg, records, fake := uploadedIndexFixture(t, 1, true)
			if _, err := runIndexSync(context.Background(), cfg, fake); err != nil {
				t.Fatal(err)
			}
			before, err := os.ReadFile(cfg.Catalog + ".index.json")
			if err != nil {
				t.Fatal(err)
			}
			bname := records[0].Label.ID + ".tar.zst.gpg"
			lname := records[0].Label.ID + ".json"
			ctx := context.Background()
			var cancel context.CancelFunc
			switch kind {
			case "duplicate":
				fake.Objects[lname] = append(fake.Objects[lname], fake.Objects[lname][0])
			case "orphan":
				delete(fake.Objects, bname)
			case "corrupt":
				fake.Objects[lname][0].Data = []byte("invalid JSON")
			case "changed":
				fake.Before = func(_ context.Context, op, _ string) error {
					if op == "lsjson" && fake.Lists == 3 {
						fake.Objects[bname][0].ID = "changed"
					}
					return nil
				}
			case "new-pair":
				fake.Before = func(_ context.Context, op, _ string) error {
					if op == "lsjson" && fake.Lists == 3 {
						fake.Objects[strings.Repeat("b", 64)+".json"] = []archivetest.FakeRemoteObject{{ID: "new", Data: []byte("{}")}}
					}
					return nil
				}
			case "partial":
				fake.FailAfterObjects = 1
			case "oversize":
				fake.Objects[lname][0].Data = make([]byte, ha.MaxMetadataBytes+1)
			case "timeout":
				ctx, cancel = context.WithTimeout(ctx, 20*time.Millisecond)
				defer cancel()
				fake.Before = func(ctx context.Context, op, _ string) error {
					if op == "copy" {
						<-ctx.Done()
						return ctx.Err()
					}
					return nil
				}
			case "bundle-sha":
				data := fake.Objects[bname][0].Data
				data[0] ^= 1
			case "receipt-id":
				fake.MissingHashes = true
				fake.Objects[bname][0].ID = "different-object"
			}
			if report, err := runIndexSync(ctx, cfg, fake); err == nil {
				t.Fatalf("accepted %s: %+v", kind, report)
			}
			after, err := os.ReadFile(cfg.Catalog + ".index.json")
			if err != nil || string(after) != string(before) {
				t.Fatalf("failed refresh changed snapshot: %v", err)
			}
			if fake.HashBatches != 0 || fake.DownloadedCipherBytes != 0 {
				t.Fatal("failure downloaded ciphertext")
			}
		})
	}
}

func TestDriveIndexSyncUnknownMembersAndCompleteEmpty(t *testing.T) {
	cfg, records, fake := uploadedIndexFixture(t, 1, false)
	name := records[0].Label.ID + ".json"
	var label map[string]json.RawMessage
	if err := json.Unmarshal(fake.Objects[name][0].Data, &label); err != nil {
		t.Fatal(err)
	}
	label["future_label"] = json.RawMessage(`{"preserve":true}`)
	raw, err := json.Marshal(label)
	if err != nil {
		t.Fatal(err)
	}
	fake.Objects[name][0].Data = raw
	fake.Objects["unrelated.txt"] = []archivetest.FakeRemoteObject{{ID: "other", Data: []byte("ignore")}}
	if _, err = runIndexSync(context.Background(), cfg, fake); err != nil {
		t.Fatal(err)
	}
	store, err := ha.OpenLabelIndexStore(context.Background(), cfg.Catalog)
	if err != nil {
		t.Fatal(err)
	}
	idx, err := store.Read()
	if err != nil {
		t.Fatal(err)
	}
	idx.Extensions = map[string]json.RawMessage{"future_index": json.RawMessage(`42`)}
	if err = store.Commit(idx); err != nil {
		t.Fatal(err)
	}
	if err = store.Close(); err != nil {
		t.Fatal(err)
	}
	fake.Objects = map[string][]archivetest.FakeRemoteObject{}
	report, err := runIndexSync(context.Background(), cfg, fake)
	if err != nil || report.LabelCount != 0 {
		t.Fatalf("empty refresh: %+v %v", report, err)
	}
	idx, err = ha.ReadLabelIndex(cfg.Catalog + ".index.json")
	if err != nil {
		t.Fatal(err)
	}
	if string(idx.Extensions["future_index"]) != "42" || len(idx.Records) != 1 {
		t.Fatalf("index history lost: %+v", idx)
	}
	for _, record := range idx.Records {
		if string(record.Label.Extensions["future_label"]) != `{"preserve":true}` {
			t.Fatal("label extension lost")
		}
		for _, loc := range record.Locations {
			if loc.Available {
				t.Fatal("missing object stayed available")
			}
		}
	}
}

func TestDriveIndexSyncValidatesDownloadedLabelMD5(t *testing.T) {
	for _, bad := range []bool{false, true} {
		t.Run(fmt.Sprint(bad), func(t *testing.T) {
			cfg, records, fake := uploadedIndexFixture(t, 1, false)
			name := records[0].Label.ID + ".json"
			runner := indexInventoryRunner{fake: fake, alter: func(entries []ha.RemoteEntry) []ha.RemoteEntry {
				for i := range entries {
					if entries[i].Name == name {
						sum := md5.Sum(fake.Objects[name][0].Data)
						value := hex.EncodeToString(sum[:])
						if bad {
							value = strings.Repeat("0", 32)
						}
						entries[i].Hashes = map[string]string{"MD5": value}
					}
				}
				return entries
			}}
			_, err := (ha.DriveIndexSync{Rclone: ha.Rclone{Runner: runner}, Now: func() time.Time { return indexSyncTime }}).Run(context.Background(), cfg)
			if (err != nil) != bad {
				t.Fatalf("bad=%v error=%v", bad, err)
			}
			if bad {
				if _, err = os.Stat(cfg.Catalog + ".index.json"); !os.IsNotExist(err) {
					t.Fatal("bad MD5 published index")
				}
			}
		})
	}
}

func TestDriveIndexSyncAggregateLabelBudgetBeforeDownload(t *testing.T) {
	cfg, records, fake := uploadedIndexFixture(t, 1, false)
	lname := records[0].Label.ID + ".json"
	bname := records[0].Label.ID + ".tar.zst.gpg"
	runner := indexInventoryRunner{fake: fake, alter: func(entries []ha.RemoteEntry) []ha.RemoteEntry {
		var label, bundle ha.RemoteEntry
		for _, entry := range entries {
			if entry.Name == lname {
				label = entry
			}
			if entry.Name == bname {
				bundle = entry
			}
		}
		if label.Name == "" {
			return entries
		}
		out := make([]ha.RemoteEntry, 0, 34)
		for i := 0; i < 17; i++ {
			id := fmt.Sprintf("%064x", i)
			l, b := label, bundle
			l.Name = id + ".json"
			l.ID = "label-" + id
			l.Size = ha.MaxMetadataBytes
			b.Name = id + ".tar.zst.gpg"
			b.ID = "bundle-" + id
			out = append(out, l, b)
		}
		return out
	}}
	_, err := (ha.DriveIndexSync{Rclone: ha.Rclone{Runner: runner}}).Run(context.Background(), cfg)
	if err == nil || fake.DownloadBatches != 0 {
		t.Fatalf("aggregate admitted: %v downloads=%d", err, fake.DownloadBatches)
	}
}
