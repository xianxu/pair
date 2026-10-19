package historyarchive

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"golang.org/x/sys/unix"
	"os"
	"path/filepath"
	"testing"
)

func TestReadUploadReceiptAdoptionReestablishesDurability(t *testing.T) {
	receipt := receiptFixture()
	raw, err := json.Marshal(receipt)
	if err != nil {
		t.Fatal(err)
	}
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	sink := LocalSink{Root: root}
	if err = sink.Put(context.Background(), "receipt.json", bytes.NewReader(raw), MaxUploadReceiptBytes); err != nil {
		t.Fatal(err)
	}
	verified := 0
	sink.Fault = func(stage string) error {
		if stage == "verify" {
			verified++
		}
		return nil
	}
	_, found, err := readUploadReceipt(context.Background(), sink, "receipt.json")
	if err != nil || !found || verified != 1 {
		t.Fatalf("adoption found=%v verify=%d error=%v", found, verified, err)
	}
}

func TestVerifyRemoteEntryRequiresMatchingChecksum(t *testing.T) {
	want := UploadObject{Name: "cipher.gpg", Size: 3, SHA256: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", MD5: "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"}
	for _, tc := range []struct {
		hashes map[string]string
		method string
		bad    bool
	}{
		{map[string]string{"SHA-256": want.SHA256}, "backend-sha256", false},
		{map[string]string{"MD5": want.MD5}, "backend-md5", false},
		{nil, "downloaded-sha256", false},
		{map[string]string{"SHA-256": "wrong"}, "", true},
		{map[string]string{"SHA256": want.SHA256, "SHA-256": want.SHA256}, "", true},
		{map[string]string{"MD5": "wrong"}, "", true},
	} {
		method, err := VerifyRemoteEntry(RemoteEntry{Name: want.Name, Size: want.Size, Hashes: tc.hashes}, want)
		if (err != nil) != tc.bad || method != tc.method {
			t.Fatalf("hashes%v method=%s err=%v", tc.hashes, method, err)
		}
	}
}

func TestDriveCandidateProjectionScansCatalogOnceAtNAnd2N(t *testing.T) {
	for _, n := range []int{8, 16} {
		t.Run(fmt.Sprint(n), func(t *testing.T) {
			req, codec := archiveFixture(t)
			a := Archiver{Codec: codec}
			for i := 0; i < n; i++ {
				name := fmt.Sprintf("row-%d.md", i)
				if err := os.WriteFile(filepath.Join(req.Source.Root, name), []byte(name), 0600); err != nil {
					t.Fatal(err)
				}
				req.Files = []string{name}
				if _, err := a.Create(context.Background(), req); err != nil {
					t.Fatal(err)
				}
			}
			catalog, err := OpenCatalog(context.Background(), req.Catalog)
			if err != nil {
				t.Fatal(err)
			}
			defer catalog.Close()
			scans := 0
			catalog.fault = func(stage string) error {
				if stage == "catalog-sync" {
					scans++
				}
				return nil
			}
			got, err := projectDriveCandidates(context.Background(), catalog, ContinuousConfig{Sink: req.Sink})
			if err != nil || len(got) != n || scans != 1 {
				t.Fatalf("n=%d projected=%d scans=%d err=%v", n, len(got), scans, err)
			}
		})
	}
}

func TestPrepareBatchObjectUsesVerifiedHardLinkAndCleansIt(t *testing.T) {
	req, codec := archiveFixture(t)
	records, err := (Archiver{Codec: codec}).Create(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	record := records[0]
	stage, err := openUploadStaging(filepath.Dir(req.Catalog))
	if err != nil {
		t.Fatal(err)
	}
	defer stage.Close()
	objects, err := openBatchObjects(stage)
	if err != nil {
		t.Fatal(err)
	}
	name := record.Label.ID + ".tar.zst.gpg"
	_, linked, err := PrepareBatchObject(context.Background(), LocalSink{Root: req.Sink}, record.BundleKey, UploadObject{Name: name, Size: record.Label.Bytes, SHA256: record.Label.SHA256}, objects)
	if err != nil || !linked {
		t.Fatalf("hardlink=%v err=%v", linked, err)
	}
	a, err := os.Stat(filepath.Join(req.Sink, record.BundleKey))
	if err != nil {
		t.Fatal(err)
	}
	b, err := os.Stat(filepath.Join(objects.Name(), name))
	if err != nil {
		t.Fatal(err)
	}
	if !os.SameFile(a, b) {
		t.Fatal("same-filesystem stage copied bytes")
	}
	objects.Close()
	if err = SweepUploadStaging(context.Background(), stage); err != nil {
		t.Fatal(err)
	}
	if _, err = os.Stat(filepath.Join(req.Sink, record.BundleKey)); err != nil {
		t.Fatal("cleanup removed original", err)
	}
}

func TestPrepareBatchObjectVerifiedCrossFilesystemFallback(t *testing.T) {
	req, codec := archiveFixture(t)
	records, err := (Archiver{Codec: codec}).Create(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	record := records[0]
	stage, err := openUploadStaging(filepath.Dir(req.Catalog))
	if err != nil {
		t.Fatal(err)
	}
	defer stage.Close()
	objects, err := openBatchObjects(stage)
	if err != nil {
		t.Fatal(err)
	}
	defer objects.Close()
	source := LocalSink{Root: req.Sink, Fault: func(stage string) error {
		if stage == "link-upload" {
			return unix.EXDEV
		}
		return nil
	}}
	name := record.Label.ID + ".tar.zst.gpg"
	_, linked, err := PrepareBatchObject(context.Background(), source, record.BundleKey, UploadObject{Name: name, Size: record.Label.Bytes, SHA256: record.Label.SHA256}, objects)
	if err != nil || linked {
		t.Fatalf("copyfallback linked=%v err=%v", linked, err)
	}
	a, _ := os.Stat(filepath.Join(req.Sink, record.BundleKey))
	b, _ := os.Stat(filepath.Join(objects.Name(), name))
	if os.SameFile(a, b) {
		t.Fatal("fallback retained a link")
	}
}
