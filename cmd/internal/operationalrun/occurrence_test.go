package operationalrun

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"syscall"
	"testing"
	"time"
)

func TestPublishOccurrenceReturnsExistingCanonicalFacts(t *testing.T) {
	s := Store{Namespace: t.TempDir()}
	original := sample("one", time.Now().UTC())
	original.Notice = "uncertain"
	original.Unknown = map[string]json.RawMessage{"future": json.RawMessage(`{ "retain": 2 }`)}
	original.Key.Unknown = map[string]json.RawMessage{"future_key": json.RawMessage(`true`)}
	if err := s.Publish(t.Context(), original.Key, original, []byte("first output")); err != nil {
		t.Fatal(err)
	}
	expected, out, err := s.Read(t.Context(), original.Key)
	if err != nil {
		t.Fatal(err)
	}
	proposed := original
	proposed.StartedAt = proposed.StartedAt.Add(time.Second)
	proposed.EndedAt = proposed.EndedAt.Add(2 * time.Second)
	proposed.Notice = "pending"
	proposed.Unknown = nil
	got, output, err := s.PublishOccurrence(t.Context(), original.Key, proposed, []byte("different output"))
	if err != nil || !reflect.DeepEqual(got, expected) || !bytes.Equal(output, out) {
		t.Fatalf("did not adopt original: %#v %q %v", got, output, err)
	}
	if err := s.Publish(t.Context(), original.Key, proposed, []byte("different output")); err == nil {
		t.Fatal("legacy Publish became permissive")
	}
}
func TestPublishOccurrenceAdoptsConcurrentWinner(t *testing.T) {
	s := Store{Namespace: t.TempDir()}
	winner := sample("one", time.Now().UTC())
	proposed := winner
	proposed.EndedAt = proposed.EndedAt.Add(time.Minute)
	absent, release := make(chan struct{}), make(chan struct{})
	competing := s
	competing.fault = func(step string) error {
		if step == "occurrence-absent" {
			close(absent)
			<-release
		}
		return nil
	}
	type result struct {
		record Record
		output []byte
		err    error
	}
	done := make(chan result, 1)
	go func() {
		r, out, err := competing.PublishOccurrence(t.Context(), winner.Key, proposed, []byte("loser"))
		done <- result{r, out, err}
	}()
	defer func() {
		select {
		case <-release:
		default:
			close(release)
		}
	}()
	select {
	case <-absent:
	case <-time.After(5 * time.Second):
		t.Fatal("did not observe initial absent bundle")
	}
	if err := s.Publish(t.Context(), winner.Key, winner, []byte("winner")); err != nil {
		t.Fatal(err)
	}
	close(release)
	var got result
	select {
	case got = <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("concurrent publisher did not return")
	}
	if got.err != nil || !got.record.EndedAt.Equal(winner.EndedAt) || string(got.output) != "winner" {
		t.Fatal(got)
	}
}
func TestPublishOccurrenceAdoptsBundleAfterRenameError(t *testing.T) {
	s := Store{Namespace: t.TempDir()}
	r := sample("one", time.Now().UTC())
	task, _ := s.task(r.Key)
	injected := errors.New("publication durability acknowledgement lost")
	s.fault = func(step string) error {
		if step != "publish-ready" {
			return nil
		}
		entries, err := os.ReadDir(task)
		if err != nil {
			return err
		}
		for _, entry := range entries {
			if stageName.MatchString(entry.Name()) {
				if err := os.Rename(filepath.Join(task, entry.Name()), filepath.Join(task, "runs", r.Key.ID)); err != nil {
					return err
				}
				return injected
			}
		}
		return errors.New("stage absent")
	}
	got, out, err := s.PublishOccurrence(t.Context(), r.Key, r, []byte("durable candidate"))
	if err != nil || got.Key.ID != r.Key.ID || string(out) != "durable candidate" {
		t.Fatal(got, string(out), err)
	}
}
func TestPublishOccurrenceRefusesDamagedBundle(t *testing.T) {
	for _, member := range []string{"record.json", "output"} {
		t.Run(member, func(t *testing.T) {
			s := Store{Namespace: t.TempDir()}
			r := sample("one", time.Now().UTC())
			if err := s.Publish(t.Context(), r.Key, r, []byte("keep")); err != nil {
				t.Fatal(err)
			}
			task, _ := s.task(r.Key)
			path := filepath.Join(task, "runs", r.Key.ID, member)
			if err := os.Remove(path); err != nil {
				t.Fatal(err)
			}
			if _, _, err := s.PublishOccurrence(t.Context(), r.Key, r, []byte("replacement")); err == nil {
				t.Fatal("accepted damaged record")
			}
			if _, err := os.Lstat(path); !os.IsNotExist(err) {
				t.Fatal("rewrote damaged immutable bundle", err)
			}
		})
	}
}

func TestPublishOccurrencePreservesSyncUncertaintyAndBusy(t *testing.T) {
	for _, failure := range []string{"sync", "busy"} {
		t.Run(failure, func(t *testing.T) {
			s := Store{Namespace: t.TempDir()}
			r := sample("one", time.Now().UTC())
			if err := s.Publish(t.Context(), r.Key, r, []byte("original")); err != nil {
				t.Fatal(err)
			}
			task, _ := s.task(r.Key)
			if failure == "sync" {
				s.fault = func(step string) error {
					if step == "occurrence-sync" {
						return syscall.EIO
					}
					return nil
				}
			} else {
				release, err := taskLock(t.Context(), task, false)
				if err != nil {
					t.Fatal(err)
				}
				defer release()
			}
			proposed := r
			proposed.EndedAt = proposed.EndedAt.Add(time.Hour)
			_, _, err := s.PublishOccurrence(t.Context(), r.Key, proposed, []byte("overwrite"))
			want := error(syscall.EIO)
			if failure == "busy" {
				want = ErrBusy
			}
			if !errors.Is(err, want) {
				t.Fatalf("uncertainty hidden: %v", err)
			}
			raw, err := os.ReadFile(filepath.Join(task, "runs", r.Key.ID, "output"))
			if err != nil || string(raw) != "original" {
				t.Fatal("rewrote uncertain bundle", string(raw), err)
			}
		})
	}
}
