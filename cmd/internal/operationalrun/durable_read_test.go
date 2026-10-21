package operationalrun

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"syscall"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/durablefile"
)

func TestReadDurableRepairsPublicationAcknowledgement(t *testing.T) {
	for _, leaf := range []string{"run", "runs", "task", "hub", "schedule", "namespace"} {
		t.Run(leaf, func(t *testing.T) {
			s := Store{Namespace: t.TempDir()}
			r := sample("one", time.Now().UTC())
			r.Unknown = map[string]json.RawMessage{"future": json.RawMessage(`{"keep":true}`)}
			r.Key.Unknown = map[string]json.RawMessage{"key_future": json.RawMessage(`42`)}
			task, _ := s.task(r.Key)
			run := filepath.Join(task, "runs", r.Key.ID)
			failPath := map[string]string{"run": run, "runs": filepath.Dir(run), "task": task, "hub": filepath.Dir(task), "schedule": filepath.Join(s.Namespace, "schedule"), "namespace": s.Namespace}[leaf]
			// Fail only after publication, so pre-publication cleanup is unaffected.
			s.SyncDirectory = func(path string) error {
				if path == failPath {
					if _, err := os.Lstat(run); err == nil {
						return syscall.EIO
					}
				}
				return durablefile.SyncDirectory(path)
			}
			if err := s.Publish(t.Context(), r.Key, r, []byte("canonical output")); !errors.Is(err, syscall.EIO) {
				t.Fatalf("publication hid directory failure: %v", err)
			}
			want, output, err := s.Read(t.Context(), r.Key)
			if err != nil || !bytes.Equal(output, []byte("canonical output")) {
				t.Fatal(want, string(output), err)
			}
			if _, _, err = s.ReadDurable(t.Context(), r.Key); !errors.Is(err, syscall.EIO) {
				t.Fatalf("durable read hid uncertainty: %v", err)
			}
			var synced []string
			s.SyncDirectory = func(path string) error {
				synced = append(synced, path)
				return durablefile.SyncDirectory(path)
			}
			got, out, err := s.ReadDurable(t.Context(), r.Key)
			if err != nil || !reflect.DeepEqual(got, want) || !bytes.Equal(output, out) {
				t.Fatal(got, string(out), err)
			}
			expectedPaths := []string{run, filepath.Dir(run), task, filepath.Dir(task), filepath.Join(s.Namespace, "schedule"), s.Namespace}
			if !reflect.DeepEqual(synced, expectedPaths) {
				t.Fatalf("sync inventory escaped/missed authority boundary: %v", synced)
			}
		})
	}
}

func TestReadDurableNeverCreatesOrRepairsMissingHistory(t *testing.T) {
	for _, kind := range []string{"namespace", "task", "run", "record.json", "output"} {
		t.Run(kind, func(t *testing.T) {
			root := t.TempDir()
			s := Store{Namespace: filepath.Join(root, "absent")}
			r := sample("one", time.Now().UTC())
			task, _ := s.task(r.Key)
			path := s.Namespace
			switch kind {
			case "task":
				if err := os.MkdirAll(s.Namespace, 0700); err != nil {
					t.Fatal(err)
				}
				path = task
			case "run", "record.json", "output":
				if err := s.Publish(t.Context(), r.Key, r, []byte("keep")); err != nil {
					t.Fatal(err)
				}
				path = filepath.Join(task, "runs", r.Key.ID)
				if kind == "run" {
					if err := os.RemoveAll(path); err != nil {
						t.Fatal(err)
					}
				} else {
					path = filepath.Join(path, kind)
					if err := os.Remove(path); err != nil {
						t.Fatal(err)
					}
				}
			}
			s.SyncDirectory = func(string) error { t.Fatal("synced absent/damaged history"); return nil }
			if _, _, err := s.ReadDurable(t.Context(), r.Key); err == nil {
				t.Fatal("accepted absent/damaged history")
			}
			if _, err := os.Lstat(path); !errors.Is(err, os.ErrNotExist) {
				t.Fatal("created missing history", err)
			}
		})
	}
}

func TestReadDurableHoldsTaskLockAndHonorsCancellation(t *testing.T) {
	s := Store{Namespace: t.TempDir()}
	r := sample("one", time.Now().UTC())
	if err := s.Publish(t.Context(), r.Key, r, nil); err != nil {
		t.Fatal(err)
	}
	task, _ := s.task(r.Key)
	s.SyncDirectory = func(path string) error {
		if path == filepath.Join(task, "runs", r.Key.ID) {
			if _, _, err := (Store{Namespace: s.Namespace}).Read(t.Context(), r.Key); !errors.Is(err, ErrBusy) {
				t.Fatal("durable read released lock before sync", err)
			}
		}
		return durablefile.SyncDirectory(path)
	}
	if _, _, err := s.ReadDurable(t.Context(), r.Key); err != nil {
		t.Fatal(err)
	}
	unlock, err := taskLock(t.Context(), task, false)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := s.ReadDurable(t.Context(), r.Key); !errors.Is(err, ErrBusy) {
		t.Fatal(err)
	}
	unlock()
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if _, _, err := s.ReadDurable(ctx, r.Key); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
}

func TestPublicationAndFinalizationConfirmAllBundleDirectories(t *testing.T) {
	for _, operation := range []string{"publish retry", "occurrence", "finalize", "finalize retry"} {
		for _, leaf := range []string{"run", "runs", "task", "hub", "schedule", "namespace"} {
			t.Run(operation+"/"+leaf, func(t *testing.T) {
				s := Store{Namespace: t.TempDir()}
				r := sample("one", time.Now().UTC())
				r.Notice = "pending"
				if err := s.Publish(t.Context(), r.Key, r, []byte("keep")); err != nil {
					t.Fatal(err)
				}
				if operation == "finalize retry" {
					s.fault = func(step string) error {
						if step == "notice-renamed" {
							return syscall.EIO
						}
						return nil
					}
					if err := s.FinalizeNotice(t.Context(), r.Key, "rev", "submitted"); !errors.Is(err, syscall.EIO) {
						t.Fatal(err)
					}
					s.fault = nil
				}
				task, _ := s.task(r.Key)
				run := filepath.Join(task, "runs", r.Key.ID)
				failPath := map[string]string{"run": run, "runs": filepath.Dir(run), "task": task, "hub": filepath.Dir(task), "schedule": filepath.Join(s.Namespace, "schedule"), "namespace": s.Namespace}[leaf]
				s.SyncDirectory = func(path string) error {
					if path == failPath {
						return syscall.EIO
					}
					return durablefile.SyncDirectory(path)
				}
				var err error
				switch operation {
				case "publish retry":
					err = s.Publish(t.Context(), r.Key, r, []byte("keep"))
				case "occurrence":
					_, _, err = s.PublishOccurrence(t.Context(), r.Key, r, []byte("replacement"))
				default:
					err = s.FinalizeNotice(t.Context(), r.Key, "rev", "submitted")
				}
				if !errors.Is(err, syscall.EIO) {
					t.Fatalf("hid %s directory sync failure: %v", leaf, err)
				}
				s.SyncDirectory = nil
				if operation == "finalize" || operation == "finalize retry" {
					if err := s.FinalizeNotice(t.Context(), r.Key, "rev", "submitted"); err != nil {
						t.Fatal(err)
					}
					got, out, err := s.ReadDurable(t.Context(), r.Key)
					if err != nil || got.Notice != "submitted" || string(out) != "keep" {
						t.Fatal(got, string(out), err)
					}
				}
			})
		}
	}
}

func TestFinalizeNoticeRefusesDamagedOutput(t *testing.T) {
	for _, damage := range []string{"missing", "oversized"} {
		t.Run(damage, func(t *testing.T) {
			for _, notice := range []string{"pending", "submitted"} {
				t.Run(notice, func(t *testing.T) {
					s := Store{Namespace: t.TempDir()}
					r := sample("one", time.Now().UTC())
					r.Notice = notice
					if err := s.Publish(t.Context(), r.Key, r, []byte("keep")); err != nil {
						t.Fatal(err)
					}
					task, _ := s.task(r.Key)
					run := filepath.Join(task, "runs", r.Key.ID)
					recordPath := filepath.Join(run, "record.json")
					before, err := os.ReadFile(recordPath)
					if err != nil {
						t.Fatal(err)
					}
					if damage == "missing" {
						if err := os.Remove(filepath.Join(run, "output")); err != nil {
							t.Fatal(err)
						}
					} else if err := os.Truncate(filepath.Join(run, "output"), OutputLimit+1); err != nil {
						t.Fatal(err)
					}
					if err := s.FinalizeNotice(t.Context(), r.Key, "rev", "submitted"); err == nil {
						t.Fatal("accepted damaged immutable bundle")
					}
					after, err := os.ReadFile(recordPath)
					if err != nil || !bytes.Equal(before, after) {
						t.Fatal("rewrote damaged bundle", err)
					}
					if damage == "missing" {
						if _, err := os.Lstat(filepath.Join(run, "output")); !errors.Is(err, os.ErrNotExist) {
							t.Fatal("recreated output", err)
						}
					}
				})
			}
		})
	}
}
