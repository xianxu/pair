package diagnosticlog

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"syscall"
	"testing"
	"time"
)

// Persist an old mount's device number without changing its inode, bytes or clock.
func oldDeviceCheckpoint(t *testing.T, path string, opts Options) diskState {
	t.Helper()
	s, err := load(path)
	if err != nil {
		t.Fatal(err)
	}
	s.Current.Identity.Device++
	if err := save(path, s, true, opts); err != nil {
		t.Fatal(err)
	}
	return s
}

func TestDeviceRebindCommittedCheckpointPreservesBytes(t *testing.T) {
	path, _, opts := fixture(t)
	w, err := Open(path, opts)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = w.Write([]byte("old facts\n")); err != nil {
		t.Fatal(err)
	}
	if err = w.Close(); err != nil {
		t.Fatal(err)
	}
	reservation, err := os.ReadFile(reservationPath(path))
	if err != nil {
		t.Fatal(err)
	}
	old := oldDeviceCheckpoint(t, path, opts)
	if _, err = Preview(path, opts, 10); err != nil {
		t.Fatalf("current preview: %v", err)
	}
	previewed, err := load(path)
	if err != nil {
		t.Fatal(err)
	}
	if previewed.Current != old.Current {
		t.Fatal("preview mutated checkpoint")
	}
	w, err = Open(path, opts)
	if err != nil {
		t.Fatalf("device-only committed checkpoint: %v", err)
	}
	got, err := load(path)
	if err != nil {
		t.Fatal(err)
	}
	st, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if got.Current.Identity != fileIdentity(st) || got.Current.Size != old.Current.Size || !got.Current.ModTime.Equal(old.Current.ModTime) || !got.Current.LastWrite.Equal(old.Current.LastWrite) {
		t.Fatal("checkpoint not rebound without clock/size changes")
	}
	sameReservation, err := os.ReadFile(reservationPath(path))
	if err != nil || !bytes.Equal(reservation, sameReservation) {
		t.Fatal("old reservation rewritten to manufacture authority", err)
	}
	if _, err = w.Write([]byte("new facts\n")); err != nil {
		t.Fatal(err)
	}
	if err = w.Close(); err != nil {
		t.Fatal(err)
	}
	w, err = Open(path, opts)
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()
	body, err := os.ReadFile(path)
	if err != nil || string(body) != "old facts\nnew facts\n" {
		t.Fatalf("bytes=%q err=%v", body, err)
	}
}

func TestDeviceRebindRetainedPreviewAndCollection(t *testing.T) {
	path, now, opts := fixture(t)
	w, err := Open(path, opts)
	if err != nil {
		t.Fatal(err)
	}
	w.Close()
	name := fmt.Sprintf("segment-%032x.log", 1)
	fixtureSegment(t, path, name, now.Add(-RetentionPeriod-24*time.Hour))
	p := segmentPath(path, name)
	var g generation
	if err = readJSON(p+".json", &g); err != nil {
		t.Fatal(err)
	}
	g.Identity.Device++
	if err = writeJSON(directory(path), p+".json", g, true, opts); err != nil {
		t.Fatal(err)
	}
	rows, err := Preview(path, opts, 10)
	if err != nil {
		t.Fatalf("retained device-only preview: %v", err)
	}
	found := false
	for _, row := range rows {
		if row.Path == p && row.Eligible {
			found = true
		}
	}
	if !found {
		t.Fatal("retained generation not eligible")
	}
	removed, err := Collect(path, opts, 10)
	if err != nil {
		t.Fatalf("retained device-only collect: %v", err)
	}
	if len(removed) != 1 || removed[0].Path != p {
		t.Fatalf("removed=%v", removed)
	}
	if _, err = os.Stat(p); !os.IsNotExist(err) {
		t.Fatalf("retained payload survives: %v", err)
	}
}

func TestCheckpointComparison(t *testing.T) {
	base := generation{Identity: identity{Device: 1, Inode: 2}, Size: 3, ModTime: time.Unix(4, 5)}
	for _, tc := range []struct {
		name   string
		change func(*generation)
		want   checkpointMatch
		bad    bool
	}{
		{"exact", func(*generation) {}, checkpointExact, false},
		{"device", func(g *generation) { g.Identity.Device++ }, checkpointDeviceOnly, false},
		{"inode", func(g *generation) { g.Identity.Inode++ }, 0, true},
		{"zero", func(g *generation) { g.Identity.Inode = 0 }, 0, true},
		{"shrink", func(g *generation) { g.Size-- }, 0, true},
		{"grow", func(g *generation) { g.Size++ }, 0, true},
		{"nanosecond", func(g *generation) { g.ModTime = g.ModTime.Add(time.Nanosecond) }, 0, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			observed := base
			tc.change(&observed)
			got, err := compareCheckpoint(observed, base)
			if (err != nil) != tc.bad || (!tc.bad && got != tc.want) {
				t.Fatalf("match=%v error=%v", got, err)
			}
		})
	}
}

func TestDeviceRebindRefusesUncertainPayloadAndIntents(t *testing.T) {
	for _, name := range []string{"inode", "nanosecond", "shrink", "grow", "partial-tail", "full-tail", "rotation", "deletion", "legacy-append", "symlink"} {
		t.Run(name, func(t *testing.T) {
			path, _, opts := fixture(t)
			w, err := Open(path, opts)
			if err != nil {
				t.Fatal(err)
			}
			if _, err = w.Write([]byte("original")); err != nil {
				t.Fatal(err)
			}
			w.Close()
			s := oldDeviceCheckpoint(t, path, opts)
			switch name {
			case "inode":
				s.Current.Identity.Inode++
			case "nanosecond":
				s.Current.ModTime = s.Current.ModTime.Add(time.Nanosecond)
			case "shrink":
				s.Current.Size++
			case "grow":
				s.Current.Size--
			case "partial-tail", "full-tail":
				r, err := readReservation(path)
				if err != nil {
					t.Fatal(err)
				}
				r.Before = s.Current
				data := []byte("tail")
				if err = writeReservation(path, s.Current, data, opts.Now(), opts); err != nil {
					t.Fatal(err)
				}
				if name == "partial-tail" {
					data = data[:2]
				}
				f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0600)
				if err != nil {
					t.Fatal(err)
				}
				_, err = f.Write(data)
				f.Close()
				if err != nil {
					t.Fatal(err)
				}
			case "rotation":
				g := s.Current
				g.Name = fmt.Sprintf("segment-%032x.log", 1)
				s.Pending = &g
			case "deletion":
				g := s.Current
				s.Deleting = &g
			case "legacy-append":
				s.Appending = &appendIntent{Before: s.Current, Data: []byte("tail"), At: opts.Now()}
			case "symlink":
				if err = os.Rename(path, path+".real"); err != nil {
					t.Fatal(err)
				}
				if err = os.Symlink(path+".real", path); err != nil {
					t.Fatal(err)
				}
			}
			if err = save(path, s, true, opts); err != nil {
				t.Fatal(err)
			}
			before, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if got, err := Open(path, opts); err == nil {
				got.Close()
				t.Fatal("uncertain state admitted")
			} else if name == "inode" && !strings.Contains(err.Error(), "inode") {
				t.Fatalf("missing inode diagnosis: %v", err)
			} else if name == "nanosecond" && !strings.Contains(err.Error(), "modification time") {
				t.Fatalf("missing time diagnosis: %v", err)
			} else if (name == "rotation" || name == "deletion" || name == "legacy-append") && !strings.Contains(err.Error(), name+" recovery") {
				t.Fatalf("missing transaction diagnosis: %v", err)
			}
			after, err := os.ReadFile(path)
			if err != nil || !bytes.Equal(before, after) {
				t.Fatalf("payload changed: %v", err)
			}
		})
	}
}

func TestDeviceRebindSaveFailurePreservesPayload(t *testing.T) {
	path, _, opts := fixture(t)
	w, err := Open(path, opts)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = w.Write([]byte("facts")); err != nil {
		t.Fatal(err)
	}
	w.Close()
	s := oldDeviceCheckpoint(t, path, opts)
	stop := fmt.Errorf("sync refused")
	opts.directorySync = func(string) error { return stop }
	err = locked(path, false, func() error { return recoverReservedAppend(path, &s, opts, true) })
	if !errors.Is(err, stop) || !strings.Contains(err.Error(), "device rebind checkpoint") {
		t.Fatalf("failure=%v", err)
	}
	body, err := os.ReadFile(path)
	if err != nil || string(body) != "facts" {
		t.Fatalf("payload=%q %v", body, err)
	}
	opts.directorySync = nil
	w, err = Open(path, opts)
	if err != nil {
		t.Fatal(err)
	}
	w.Close()
}

type deviceChangedInfo struct{ os.FileInfo }

func (s deviceChangedInfo) Sys() any { st := *s.FileInfo.Sys().(*syscall.Stat_t); st.Dev++; return &st }
func TestDeviceRebindNeverWeakensLiveObservation(t *testing.T) {
	path, _, opts := fixture(t)
	w, err := Open(path, opts)
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()
	st, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	g := observedGeneration(st)
	if err := matches(deviceChangedInfo{st}, g); err == nil || !strings.Contains(err.Error(), "device") {
		t.Fatalf("live device change not specifically refused: %v", err)
	}
}

func TestDeviceRebindCreationIntentStaysStrict(t *testing.T) {
	path, _, opts := fixture(t)
	w, err := Open(path, opts)
	if err != nil {
		t.Fatal(err)
	}
	w.Close()
	s := oldDeviceCheckpoint(t, path, opts)
	g := s.Current
	s.Creating = &g
	s.Current = generation{}
	if err = save(path, s, true, opts); err != nil {
		t.Fatal(err)
	}
	if got, err := Open(path, opts); err == nil {
		got.Close()
		t.Fatal("creation remount admitted")
	} else if !strings.Contains(err.Error(), "creation recovery") || !strings.Contains(err.Error(), "device") {
		t.Fatalf("creation refusal=%v", err)
	}
	st, err := os.Stat(path)
	if err != nil || st.Size() != 0 {
		t.Fatal("creation changed payload", err)
	}
}

// Exercise the real preview -> prepare -> delete path. The first reader proof
// replaces the file and presents matching newer metadata with an old device.
// The second observation must remain bound to the first, not accept that record.
func TestDeviceRebindCollectionKeepsFirstObservation(t *testing.T) {
	path, now, opts := fixture(t)
	w, err := Open(path, opts)
	if err != nil {
		t.Fatal(err)
	}
	w.Close()
	name := fmt.Sprintf("segment-%032x.log", 1)
	fixtureSegment(t, path, name, now.Add(-RetentionPeriod-24*time.Hour))
	p := segmentPath(path, name)
	var g generation
	if err = readJSON(p+".json", &g); err != nil {
		t.Fatal(err)
	}
	changed := false
	opts.ReaderProof = func(_ context.Context, candidate string) error {
		if candidate != p || changed {
			return nil
		}
		changed = true
		body, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		if err = os.Rename(p, p+".previous"); err != nil {
			return err
		}
		if err = os.WriteFile(p, body, 0600); err != nil {
			return err
		}
		if err = os.Chtimes(p, g.ModTime, g.ModTime); err != nil {
			return err
		}
		st, err := os.Stat(p)
		if err != nil {
			return err
		}
		g.Identity = fileIdentity(st)
		g.Identity.Device++
		return writeJSON(directory(path), p+".json", g, true, opts)
	}
	if _, err = Collect(path, opts, 10); err == nil {
		t.Fatal("collector discarded preview observation and deleted replacement")
	}
	if !changed {
		t.Fatal("first observation hook not reached")
	}
	if _, err = os.Stat(p); err != nil {
		t.Fatalf("replacement was not preserved: %v", err)
	}
	s, err := load(path)
	if err != nil {
		t.Fatal(err)
	}
	if s.Deleting != nil {
		t.Fatal("admitted deletion after live identity changed")
	}
}
