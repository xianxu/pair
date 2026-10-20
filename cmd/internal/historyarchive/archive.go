package historyarchive

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/sys/unix"
)

// Archiver creates backups synchronously. Success is evidence for the exact
// manifest and sink, not a grant to delete any source.
type Archiver struct {
	Codec Codec
	Sink  Sink
	Now   func() time.Time
}

func (a Archiver) Create(ctx context.Context, req ArchiveRequest) ([]CatalogRecord, error) {
	return a.create(ctx, req, false)
}

// create is Create; collected admits a snapshot kind whose files the
// continuous pass snapshotted itself.
func (a Archiver) create(ctx context.Context, req ArchiveRequest, collected bool) (records []CatalogRecord, err error) {
	if err = validateRequest(req, collected); err != nil {
		return nil, fmt.Errorf("archive request: %w; correct request", err)
	}
	if a.Codec == nil {
		return nil, errors.New("archive codec unavailable; configure codec")
	}
	if overlaps(req.Source.Root, req.Sink) || overlaps(req.Source.Root, filepath.Dir(req.Catalog)) {
		return nil, errors.New("archive destination overlaps source root; choose a separate destination")
	}
	if overlaps(req.Catalog+".staging", req.Sink) {
		return nil, errors.New("archive sink overlaps staging namespace; choose a separate destination")
	}
	if err = a.Codec.CheckRecipients(ctx, req.Recipients); err != nil {
		return nil, fmt.Errorf("archive recipients: %w; configure a usable public recipient", err)
	}
	catalog, err := OpenCatalog(ctx, req.Catalog)
	if err != nil {
		return nil, fmt.Errorf("archive catalog: %w; retry after resolving catalog error", err)
	}
	defer func() { err = errors.Join(err, catalog.Close()) }()
	sink := a.Sink
	if sink == nil {
		sink = &LocalSink{Root: req.Sink}
	}
	files := &SelectedFiles{Root: req.Source.Root, Limit: rawBundleLimit(req.Kind)}
	described, err := files.Describe(ctx, req.Files)
	if err != nil {
		return nil, fmt.Errorf("archive source: %w; retry with stable regular files", err)
	}
	manifests, err := planBundles(req, described, collected)
	if err != nil {
		return nil, err
	}
	inventory, err := Defaults()
	if err != nil {
		return nil, err
	}
	for _, manifest := range manifests {
		age := a.now().Sub(manifest.To)
		if age < 0 {
			age = 0
		}
		route, e := inventory.Route(req.Kind, age)
		if e != nil {
			return records, e
		}
		if route != "local" {
			return records, errors.New("archive sink route is unsupported; configure a local sink")
		}
		record, e := a.createOne(ctx, req, manifest, files, sink, catalog)
		if e != nil {
			return records, e
		}
		records = append(records, record)
	}
	return records, nil
}
func overlaps(a, b string) bool {
	a = filepath.Clean(a)
	b = filepath.Clean(b)
	return a == b || a == string(filepath.Separator) || b == string(filepath.Separator) || strings.HasPrefix(a, b+string(filepath.Separator)) || strings.HasPrefix(b, a+string(filepath.Separator))
}
func (a Archiver) now() time.Time {
	if a.Now != nil {
		return a.Now().UTC()
	}
	return time.Now().UTC()
}
func (a Archiver) createOne(ctx context.Context, req ArchiveRequest, m Manifest, files *SelectedFiles, sink Sink, catalog *Catalog) (CatalogRecord, error) {
	id, err := m.ID()
	if err != nil {
		return CatalogRecord{}, err
	}
	bkey, lkey, err := m.Keys()
	if err != nil {
		return CatalogRecord{}, err
	}
	existing, found, err := catalog.Find(ctx, id)
	if err != nil {
		return CatalogRecord{}, err
	}
	label, observation, err := inspectBundle(ctx, sink, bkey, lkey, m, req.Sink, a.now())
	if err != nil {
		return CatalogRecord{}, fmt.Errorf("archive verification: %w; inspect stored bundle", err)
	}
	decision := DecidePublication(observation)
	if found {
		if decision != PublicationVerifiedExisting {
			return CatalogRecord{}, errors.New("catalog record conflicts with stored bundle; inspect stored bundle")
		}
		candidate := CatalogRecord{Label: label, Sink: req.Sink, BundleKey: bkey, LabelKey: lkey, VerifiedAt: existing.VerifiedAt, Manifest: m}
		if !sameCatalogRecord(existing, candidate) {
			return CatalogRecord{}, errors.New("catalog record conflicts with stored bundle; inspect stored bundle")
		}
		return existing, nil
	}
	if decision == PublicationConflict {
		return CatalogRecord{}, errors.New("incomplete or conflicting bundle publication; inspect stored bundle")
	}
	if decision == PublicationAbsent {
		label, err = a.writeBundle(ctx, req, m, files, sink, bkey, lkey)
		if err != nil {
			return CatalogRecord{}, err
		}
		verified, obs, e := inspectBundle(ctx, sink, bkey, lkey, m, req.Sink, a.now())
		if e != nil {
			return CatalogRecord{}, fmt.Errorf("archive verification: %w; retry", e)
		}
		if DecidePublication(obs) != PublicationVerifiedExisting {
			return CatalogRecord{}, errors.New("archive verification incomplete; retry")
		}
		want, _ := json.Marshal(label)
		got, _ := json.Marshal(verified)
		if !bytes.Equal(want, got) {
			return CatalogRecord{}, errors.New("archive label changed during verification; inspect stored bundle")
		}
	}
	record := CatalogRecord{Label: label, Sink: req.Sink, BundleKey: bkey, LabelKey: lkey, VerifiedAt: a.now(), Manifest: m}
	if err = catalog.Append(ctx, record); err != nil {
		return CatalogRecord{}, fmt.Errorf("archive catalog append: %w; retry", err)
	}
	return record, nil
}
func inspectBundle(ctx context.Context, sink Sink, bkey, lkey string, m Manifest, sinkName string, now time.Time) (BundleLabel, PublicationObservation, error) {
	var label BundleLabel
	var obs PublicationObservation
	r, err := sink.Get(ctx, lkey)
	if errors.Is(err, os.ErrNotExist) {
		b, e := sink.Get(ctx, bkey)
		if e == nil {
			obs.BundlePresent = true
			b.Close()
		} else if !errors.Is(e, os.ErrNotExist) {
			return label, obs, e
		}
		return label, obs, nil
	}
	if err != nil {
		return label, obs, err
	}
	obs.LabelPresent = true
	raw, readErr := io.ReadAll(io.LimitReader(r, MaxMetadataBytes+1))
	err = errors.Join(readErr, r.Close())
	if err != nil {
		return label, obs, err
	}
	if label, err = DecodeLabel(raw); err != nil {
		return label, obs, err
	}
	record := CatalogRecord{Label: label, Sink: sinkName, BundleKey: bkey, LabelKey: lkey, VerifiedAt: now, Manifest: m}
	if err = ValidateRecord(record); err != nil {
		return label, obs, err
	}
	if err = sink.Verify(ctx, bkey, label.SHA256, label.Bytes); err != nil {
		return label, obs, err
	}
	obs.BundlePresent = true
	hash := sha256.Sum256(raw)
	if err = sink.Verify(ctx, lkey, hex.EncodeToString(hash[:]), int64(len(raw))); err != nil {
		return label, obs, err
	}
	obs.Matches = true
	obs.Durable = true
	return label, obs, nil
}
func (a Archiver) writeBundle(ctx context.Context, req ArchiveRequest, m Manifest, files *SelectedFiles, sink Sink, bkey, lkey string) (BundleLabel, error) {
	var label BundleLabel
	stageDir := req.Catalog + ".staging"
	dir, err := openSafeDir(stageDir, true)
	if err != nil {
		return label, err
	}
	defer dir.Close()
	// The catalog lock covers this private staging namespace. It is separate
	// from sink objects and is never allowed inside the source root.
	names, err := dir.Readdirnames(-1)
	if err != nil {
		return label, err
	}
	for _, name := range names {
		if !strings.HasPrefix(name, "cipher-") {
			return label, errors.New("unexpected archive staging entry; inspect staging directory")
		}
		f, e := openRegularAt(dir, name, unix.O_RDONLY)
		if e != nil {
			return label, e
		}
		f.Close()
		if e = unix.Unlinkat(int(dir.Fd()), name, 0); e != nil {
			return label, e
		}
	}
	var token [16]byte
	if _, err = rand.Read(token[:]); err != nil {
		return label, err
	}
	name := "cipher-" + hex.EncodeToString(token[:])
	f, err := openRegularAt(dir, name, unix.O_RDWR|unix.O_CREAT|unix.O_EXCL)
	if err != nil {
		return label, err
	}
	defer func() { f.Close(); unix.Unlinkat(int(dir.Fd()), name, 0) }()
	pr, pw := io.Pipe()
	done := make(chan error, 1)
	go func() { e := files.WriteTar(ctx, pw, m); pw.CloseWithError(e); done <- e }()
	bounded := &boundedCipherWriter{writer: f, remaining: maxCiphertext}
	encodeErr := a.Codec.Encode(ctx, pr, bounded, m.Recipients, m.Compression)
	pr.CloseWithError(encodeErr)
	tarErr := <-done
	// The codec may not wrap the writer's error; the writer remembers.
	if bounded.exceeded {
		return label, ErrCiphertextLimit
	}
	if err = errors.Join(encodeErr, tarErr); err != nil {
		return label, fmt.Errorf("archive transform: %w; retry", err)
	}
	if err = ctx.Err(); err != nil {
		return label, err
	}
	if err = f.Sync(); err != nil {
		return label, err
	}
	if _, err = f.Seek(0, io.SeekStart); err != nil {
		return label, err
	}
	hash := sha256.New()
	size, err := io.Copy(hash, f)
	if err != nil {
		return label, err
	}
	if _, err = f.Seek(0, io.SeekStart); err != nil {
		return label, err
	}
	id, err := m.ID()
	if err != nil {
		return label, err
	}
	var rawBytes int64
	for _, entry := range m.Files {
		rawBytes += entry.Size
	}
	label = BundleLabel{Version: Version, ID: id, Kind: m.Kind, From: m.From, To: m.To, FileCount: len(m.Files), RawBytes: rawBytes, Bytes: size, Compression: m.Compression, Recipients: m.Recipients, SHA256: hex.EncodeToString(hash.Sum(nil))}
	if err = ValidateLabel(label); err != nil {
		return label, err
	}
	if err = sink.Put(ctx, bkey, f, MaxCiphertextBytes); err != nil {
		return label, fmt.Errorf("archive bundle publication: %w; retry", err)
	}
	raw, err := json.Marshal(label)
	if err != nil {
		return label, err
	}
	if err = sink.Put(ctx, lkey, bytes.NewReader(raw), MaxMetadataBytes); err != nil {
		return label, fmt.Errorf("archive label publication: %w; inspect stored bundle", err)
	}
	return label, nil
}

// ErrCiphertextLimit means a bundle's compressed, encrypted form exceeds
// MaxCiphertextBytes: the input is too large to archive as one bundle.
var ErrCiphertextLimit = errors.New("compressed bundle exceeds the ciphertext bound")

// maxCiphertext is MaxCiphertextBytes; tests lower it.
var maxCiphertext int64 = MaxCiphertextBytes

type boundedCipherWriter struct {
	writer    io.Writer
	remaining int64
	exceeded  bool
}

func (w *boundedCipherWriter) Write(p []byte) (int, error) {
	if int64(len(p)) > w.remaining {
		w.exceeded = true
		return 0, ErrCiphertextLimit
	}
	n, e := w.writer.Write(p)
	w.remaining -= int64(n)
	return n, e
}
