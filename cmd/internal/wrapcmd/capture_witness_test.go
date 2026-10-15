package wrapcmd

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/artifactpath"
)

func TestPublishCaptureWitnessGenerationAndAppendRefusal(t *testing.T) {
	dir := t.TempDir()
	raw, _ := os.Create(filepath.Join(dir, "raw"))
	defer raw.Close()
	events, _ := os.Create(filepath.Join(dir, "events"))
	defer events.Close()
	w := artifactpath.CaptureWitness{ScopeKey: "scope", Tag: "work", Agent: "codex", LaunchOrdinal: 18, Nonce: "start-a"}
	if e := PublishCaptureWitness(raw, events, w); e != nil {
		t.Fatal(e)
	}
	data, _ := os.ReadFile(events.Name())
	first, e := artifactpath.DecodeCaptureWitness(data)
	if e != nil {
		t.Fatal(e)
	}
	if e = PublishCaptureWitness(raw, events, w); e == nil {
		t.Fatal("allowed append to witnessed generation")
	}
	if e = raw.Truncate(0); e != nil {
		t.Fatal(e)
	}
	events.Truncate(0)
	events.Seek(0, 0)
	if e = PublishCaptureWitness(raw, events, w); e != nil {
		t.Fatal(e)
	}
	data, _ = os.ReadFile(events.Name())
	second, _ := artifactpath.DecodeCaptureWitness(data)
	if first.Generation == second.Generation {
		t.Fatal("reopen reused generation")
	}
	raw.Write([]byte("existing"))
	events.Truncate(0)
	events.Seek(0, 0)
	if e = PublishCaptureWitness(raw, events, w); e == nil {
		t.Fatal("certified preexisting raw")
	}
}
func TestPublishCaptureWitnessClosedEvents(t *testing.T) {
	dir := t.TempDir()
	raw, _ := os.Create(filepath.Join(dir, "raw"))
	defer raw.Close()
	events, _ := os.Create(filepath.Join(dir, "events"))
	events.Close()
	if e := PublishCaptureWitness(raw, events, artifactpath.CaptureWitness{ScopeKey: "scope", Tag: "work", Agent: "codex", LaunchOrdinal: 1, Nonce: "n"}); e == nil {
		t.Fatal("accepted failed publication")
	}
}

func TestCaptureOpenFailureCannotRetainOldAuthority(t *testing.T) {
	d := t.TempDir()
	rawPath := filepath.Join(d, "raw")
	eventsPath := filepath.Join(d, "events")
	env := func(k string) string {
		return map[string]string{"PAIR_LAUNCH_ORDINAL": "18", "PAIR_LAUNCH_NONCE": "n", "PAIR_SCOPE_KEY": "scope", "PAIR_TAG": "work", "PAIR_AGENT": "codex"}[k]
	}
	raw, events, e := openWitnessedCapture(rawPath, eventsPath, env)
	if e != nil {
		t.Fatal(e)
	}
	raw.Write([]byte("old"))
	raw.Close()
	events.Close()
	os.Rename(rawPath, rawPath+".saved")
	os.Mkdir(rawPath, 0700)
	if _, _, e = openWitnessedCapture(rawPath, eventsPath, env); e == nil {
		t.Fatal("opened directory as raw")
	}
	data, _ := os.ReadFile(eventsPath)
	if _, e = artifactpath.DecodeCaptureWitness(data); e != artifactpath.ErrCaptureWitnessAbsent {
		t.Fatalf("retained old authority: %s %v", data, e)
	}
	os.Remove(rawPath)
	os.Rename(rawPath+".saved", rawPath)
	os.Remove(eventsPath)
	os.Mkdir(eventsPath, 0700)
	if _, _, e = openWitnessedCapture(rawPath, eventsPath, env); e == nil {
		t.Fatal("opened directory as events")
	}
	data, _ = os.ReadFile(rawPath)
	if string(data) != "old" {
		t.Fatal("changed raw before events invalidation")
	}
}

func TestCaptureOpenWithoutLaunchAuthorityIsLegacy(t *testing.T) {
	d := t.TempDir()
	raw, events, e := openWitnessedCapture(filepath.Join(d, "raw"), filepath.Join(d, "events"), func(string) string { return "" })
	if e != nil {
		t.Fatal(e)
	}
	defer raw.Close()
	defer events.Close()
	b, _ := os.ReadFile(events.Name())
	if _, e := artifactpath.DecodeCaptureWitness(b); e != artifactpath.ErrCaptureWitnessAbsent {
		t.Fatal("invented authority")
	}
}

func TestPublishCaptureWitnessWritesCaptureOpenEpoch(t *testing.T) {
	dir := t.TempDir()
	raw, err := os.Create(filepath.Join(dir, "raw"))
	if err != nil {
		t.Fatal(err)
	}
	defer raw.Close()
	events, err := os.Create(filepath.Join(dir, "events"))
	if err != nil {
		t.Fatal(err)
	}
	defer events.Close()
	before := time.Now().UTC()
	input := artifactpath.CaptureWitness{ScopeKey: "scope", Tag: "work", Agent: "codex", LaunchOrdinal: 18, Nonce: "same-reexec-nonce", OpenedAt: time.Date(2001, 1, 1, 0, 0, 0, 0, time.UTC)}
	if err := PublishCaptureWitness(raw, events, input); err != nil {
		t.Fatal(err)
	}
	bytes, err := os.ReadFile(events.Name())
	if err != nil {
		t.Fatal(err)
	}
	w, err := artifactpath.DecodeCaptureWitness(bytes)
	if err != nil {
		t.Fatal(err)
	}
	if w.Version != 2 || w.OpenedAt.Before(before) || w.OpenedAt.After(time.Now().UTC()) {
		t.Fatalf("producer failed fresh epoch: %+v", w)
	}
	if rawStat, err := raw.Stat(); err != nil || rawStat.Size() != 0 {
		t.Fatal("witness did not precede child output")
	}
}
