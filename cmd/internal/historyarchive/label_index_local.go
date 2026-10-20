package historyarchive

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"golang.org/x/sys/unix"
)

// localObservationReport counts work at the actual scanner/read boundaries.
// It is intentionally internal; future diagnostics can use the same observation.
type localObservationReport struct {
	Catalog        catalogScanReport
	LabelReads     int
	DirectoryScans int
}

// ObserveLocalLabels uses existing metadata only. It cannot publish or repair a backup.
func ObserveLocalLabels(ctx context.Context, cfg ContinuousConfig, at time.Time) (IndexRefresh, error) {
	r, _, err := observeLocalLabels(ctx, cfg, at)
	return r, err
}

func observeLocalLabels(ctx context.Context, cfg ContinuousConfig, at time.Time) (r IndexRefresh, report localObservationReport, err error) {
	r = IndexRefresh{Backend: "local", Locator: cfg.Sink, At: at, Complete: true}
	if !filepath.IsAbs(cfg.Sink) || !filepath.IsAbs(cfg.Catalog) || !validTime(at) {
		return r, report, errors.New("invalid local inventory paths or time")
	}
	labels := map[string]IndexObservation{}
	bytesUsed := 0
	add := func(label BundleLabel, proof LocationProof) error {
		if len(labels) >= MaxIndexRecords {
			return errors.New("local labels exceed record bound")
		}
		b, k, e := BundleKeys(label.Kind, label.From, label.To, label.ID)
		if e != nil {
			return e
		}
		loc := BundleLocation{Backend: "local", Locator: cfg.Sink, BundleKey: b, LabelKey: k, Evidence: "observed", FirstObserved: at, LastObserved: at, Proof: proof}
		if proof.Method != "" {
			loc.FirstObserved = proof.VerifiedAt
			loc.LastObserved = proof.VerifiedAt
		}
		root, e := openSafeDir(cfg.Sink, false)
		if e != nil && !errors.Is(e, os.ErrNotExist) {
			return e
		}
		if e == nil {
			defer root.Close()
			raw, e := readLocalLabel(root, k, &report)
			if e == nil {
				disk, e := DecodeLabel(raw)
				if e != nil {
					return e
				}
				a, z := label, disk
				a.Extensions = nil
				z.Extensions = nil
				x, _ := json.Marshal(a)
				y, _ := json.Marshal(z)
				if string(x) != string(y) {
					return errors.New("catalog and stored label conflict")
				}
				label.Extensions, e = mergeExtensions(label.Extensions, disk.Extensions)
				if e != nil {
					return e
				}
				f, e := openRegularAt(root, b, unix.O_RDONLY)
				if e == nil {
					st, se := f.Stat()
					ce := f.Close()
					if se != nil || ce != nil {
						return errors.Join(se, ce)
					}
					if st.Size() != label.Bytes {
						return errors.New("local ciphertext size conflicts with label")
					}
					loc.Available = true
					loc.LastObserved = at
				}
				if e != nil && !errors.Is(e, os.ErrNotExist) {
					return e
				}
			} else if !errors.Is(e, os.ErrNotExist) {
				return e
			}
		}
		if prior, ok := labels[label.ID]; ok {
			a, z := prior.Label, label
			a.Extensions = nil
			z.Extensions = nil
			x, _ := json.Marshal(a)
			y, _ := json.Marshal(z)
			if string(x) != string(y) {
				return errors.New("conflicting local labels")
			}
			return nil
		}
		raw, e := json.Marshal(label)
		if e != nil {
			return e
		}
		bytesUsed += len(raw)
		if bytesUsed > MaxIndexBytes {
			return errors.New("local labels exceed 64 MiB")
		}
		labels[label.ID] = IndexObservation{Label: label, Location: loc}
		return nil
	}
	catalog, e := openCatalogMode(ctx, cfg.Catalog, catalogShardBytes, true, &report.Catalog)
	if e == nil {
		e = catalog.Each(ctx, func(record CatalogRecord) error {
			if record.Sink != cfg.Sink {
				return nil
			}
			return add(record.Label, LocationProof{Method: "local-catalog", SHA256: record.Label.SHA256, VerifiedAt: record.VerifiedAt})
		})
		e = errors.Join(e, catalog.Close())
		if e != nil {
			return r, report, e
		}
	} else if errors.Is(e, os.ErrNotExist) {
		if e = discoverLocalLabels(ctx, cfg.Sink, &report, func(label BundleLabel) error { return add(label, LocationProof{}) }); e != nil {
			return r, report, e
		}
	} else {
		return r, report, e
	}
	for _, ob := range labels {
		r.Observations = append(r.Observations, ob)
	}
	return r, report, ctx.Err()
}

func readLocalLabel(root *os.File, key string, report *localObservationReport) ([]byte, error) {
	if report != nil {
		report.LabelReads++
	}
	f, err := openRegularAt(root, key, unix.O_RDONLY)
	if err != nil {
		return nil, err
	}
	raw, err := io.ReadAll(io.LimitReader(f, MaxMetadataBytes+1))
	err = errors.Join(err, f.Close())
	if len(raw) > MaxMetadataBytes {
		return nil, errors.New("local label exceeds 4 MiB")
	}
	return raw, err
}

// discoverLocalLabels traverses only inventory-kind/year/month, never arbitrary roots.
func discoverLocalLabels(ctx context.Context, rootPath string, report *localObservationReport, visit func(BundleLabel) error) error {
	root, err := openSafeDir(rootPath, false)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	defer root.Close()
	inventory, err := Defaults()
	if err != nil {
		return err
	}
	remaining := MaxIndexRecords * 4
	names := func(d *os.File) ([]string, error) {
		if report != nil {
			report.DirectoryScans++
		}
		list, e := d.Readdirnames(remaining + 1)
		if e != nil && !errors.Is(e, io.EOF) {
			return nil, e
		}
		remaining -= len(list)
		if remaining < 0 {
			return nil, errors.New("local inventory entry limit exceeded")
		}
		return list, nil
	}
	for _, rule := range inventory.Sources {
		kind, e := childDir(root, rule.Kind, false)
		if errors.Is(e, os.ErrNotExist) {
			continue
		}
		if e != nil {
			return e
		}
		years, e := names(kind)
		if e != nil {
			kind.Close()
			return e
		}
		for _, year := range years {
			if err := ctx.Err(); err != nil {
				kind.Close()
				return err
			}
			n, e := strconv.Atoi(year)
			if e != nil || len(year) != 4 || n < 1 || n > 9999 {
				continue
			}
			yd, e := childDir(kind, year, false)
			if e != nil {
				kind.Close()
				return e
			}
			months, e := names(yd)
			if e != nil {
				yd.Close()
				kind.Close()
				return e
			}
			for _, month := range months {
				m, e := strconv.Atoi(month)
				if e != nil || len(month) != 2 || m < 1 || m > 12 {
					continue
				}
				md, e := childDir(yd, month, false)
				if e != nil {
					yd.Close()
					kind.Close()
					return e
				}
				files, e := names(md)
				if e != nil {
					md.Close()
					yd.Close()
					kind.Close()
					return e
				}
				for _, name := range files {
					if !strings.HasSuffix(name, ".json") {
						continue
					}
					key := fmt.Sprintf("%s/%s/%s/%s", rule.Kind, year, month, name)
					raw, e := readLocalLabel(root, key, report)
					if e != nil {
						md.Close()
						yd.Close()
						kind.Close()
						return e
					}
					label, e := DecodeLabel(raw)
					if e == nil {
						_, expected, ke := BundleKeys(label.Kind, label.From, label.To, label.ID)
						if ke != nil || expected != key {
							e = errors.New("local label name does not match identity")
						}
					}
					if e == nil {
						e = visit(label)
					}
					if e != nil {
						md.Close()
						yd.Close()
						kind.Close()
						return e
					}
				}
				md.Close()
			}
			yd.Close()
		}
		kind.Close()
	}
	return ctx.Err()
}
