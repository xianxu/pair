package webcast

import (
	"context"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/broadcast"
	"github.com/xianxu/pair/cmd/internal/webcast/recording"
)

// A recorded cast: Launch gets the recording's options; what the recorder
// writes lands in <dir>/<id>/ and, through the Tee, reaches viewers as rec
// events after their state. FakeRecorder stands in for the recorder.
func TestCastRecordsToDirAndViewers(t *testing.T) {
	dir := t.TempDir()
	f := NewFakeBrowser()
	var got *RecordOptions
	c, err := StartCast(context.Background(), CastConfig{
		Launch:        func(_ context.Context, ro *RecordOptions) (Browser, error) { got = ro; return f, nil },
		Host:          broadcast.NewHost(broadcast.HostConfig{Tunnel: &broadcast.FakeTunnel{}, ProbeTimeout: 2 * time.Second}),
		Ping:          time.Second,
		RecordingsDir: dir,
		NewLive: func() (recording.Sink, recording.Live) {
			h := recording.NewFakeHub(256)
			return h, h
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(c.End)
	if got == nil || got.ID == "" || got.Out == nil {
		t.Fatalf("record options %+v", got)
	}
	if entries, _ := os.ReadDir(dir); len(entries) != 0 {
		t.Fatal("nothing is created before go-live")
	}
	srv := httptest.NewServer(c.server)
	t.Cleanup(srv.Close)
	s := openSSE(t, srv.URL+"/"+c.token+"/events")
	s.nextState()

	o, err := got.Out()
	if err != nil {
		t.Fatal(err)
	}
	// The contract recording's one asset body, as the asset store would
	// persist it through the DirWriter.
	if _, err := o.Frames.(*recording.DirWriter).PutAsset([]byte("a")); err != nil {
		t.Fatal(err)
	}
	recs := recording.ContractRecording(3)
	if err := (recording.FakeRecorder{Records: recs}).Run(context.Background(), o.Sink); err != nil {
		t.Fatal(err)
	}
	if err := o.Sink.Close(); err != nil {
		t.Fatal(err)
	}
	for i := range recs {
		ev, data := s.next()
		if ev != "rec" {
			t.Fatalf("event %d: %s %s", i, ev, data)
		}
		r, err := recording.Decode([]byte(data))
		if err != nil || r.Kind() != recs[i].Kind() {
			t.Fatalf("record %d: %s (%v), want %s", i, data, err, recs[i].Kind())
		}
	}
	fs, _, err := recording.ValidateDir(filepath.Join(dir, got.ID))
	if err != nil || len(recording.Errors(fs)) != 0 {
		t.Fatalf("recording: %v %v", fs, err)
	}
}
