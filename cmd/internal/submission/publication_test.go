package submission

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestObservePublicationRecoveryRequiresCompleteDurableBundle(t *testing.T) {
	s, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	want := fixture()
	_, obs, err := s.ObservePublication(want.ID)
	if err != nil || obs.State != PublicationAbsent {
		t.Fatal(obs, err)
	}
	if err = os.Mkdir(filepath.Dir(s.ManifestPath(want.ID)), 0700); err != nil {
		t.Fatal(err)
	}
	_, obs, err = s.ObservePublication(want.ID)
	if err == nil || obs.State != PublicationUnknown {
		t.Fatal("incomplete directory is not absence", obs, err)
	}
	if err = os.Remove(filepath.Dir(s.ManifestPath(want.ID))); err != nil {
		t.Fatal(err)
	}
	if _, err = s.Prepare(want, nil); err != nil {
		t.Fatal(err)
	}
	s.fail = func(string) error { return errors.New("sync failed") }
	_, obs, err = s.ObservePublication(want.ID)
	if err == nil || obs.State != PublicationUnknown {
		t.Fatal(obs, err)
	}
	s.fail = nil
	got, obs, err := s.ObservePublication(want.ID)
	if err != nil || obs.State != PublicationConfirmed || got.ID != want.ID {
		t.Fatal(got, obs, err)
	}
}

func TestConfirmPreparedDoesNotPublishAbsentBundle(t *testing.T) {
	s, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	got, err := s.ConfirmPrepared(fixture(), nil)
	if err != nil || got.State != PublicationAbsent {
		t.Fatal(got, err)
	}
	if _, err = s.Read(fixture().ID); err == nil {
		t.Fatal("confirmation published bundle")
	}
}
func TestConfirmPreparedReestablishesDurabilityAndRejectsMismatch(t *testing.T) {
	s, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	sub := fixture()
	if _, err = s.Prepare(sub, nil); err != nil {
		t.Fatal(err)
	}
	s.fail = func(string) error { return errors.New("sync failed") }
	got, err := s.ConfirmPrepared(sub, nil)
	if err == nil || got.State != PublicationUnknown {
		t.Fatal(got, err)
	}
	s.fail = nil
	got, err = s.ConfirmPrepared(sub, nil)
	if err != nil || got.State != PublicationConfirmed || got.Path != s.ManifestPath(sub.ID) {
		t.Fatal(got, err)
	}
	sub.Notes[0].Text = "different"
	if got, err = s.ConfirmPrepared(sub, nil); err == nil || got.State != PublicationUnknown {
		t.Fatal(got, err)
	}
}
