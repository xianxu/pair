package webcast

import (
	"context"
	"fmt"
	"io"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/broadcast"
	castassets "github.com/xianxu/pair/cmd/internal/webcast/assets"
	"github.com/xianxu/pair/cmd/internal/webcast/recording"
)

// A recorded cast: Launch gets the recording's options; what the recorder
// writes lands in <dir>/<id>/ and, through the Tee, reaches viewers as rec
// events after their state. FakeRecorder stands in for the recorder.
func TestCastRecordsToDirAndViewers(t *testing.T) {
	dir := t.TempDir()
	f := NewFakeBrowser()
	var got *RecordOptions
	var logged []string
	c, err := StartCast(context.Background(), CastConfig{
		Launch:        func(_ context.Context, ro *RecordOptions) (Browser, error) { got = ro; return f, nil },
		Host:          broadcast.NewHost(broadcast.HostConfig{Tunnel: &broadcast.FakeTunnel{}, ProbeTimeout: 2 * time.Second}),
		Ping:          time.Second,
		RecordingsDir: dir,
		Logf:          func(format string, args ...any) { logged = append(logged, fmt.Sprintf(format, args...)) },
		NewAssets: func(w recording.AssetWriter) castassets.Served {
			return castassets.NewRecorded(w, castassets.DefaultPolicy)
		},
		NewLive: func() (recording.Sink, recording.Live) {
			h := recording.NewHub(256)
			return h, h
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(c.End)
	if got == nil || got.ID == "" || got.Out == nil || got.Logf == nil {
		t.Fatalf("record options %+v", got)
	}
	// The recorder's reports reach the operator's Logf.
	got.Logf("lost %d", 1)
	if len(logged) != 1 || logged[0] != "lost 1" {
		t.Fatalf("Logf: %v", logged)
	}
	if entries, err := os.ReadDir(dir); err != nil || len(entries) != 0 {
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
	// The contract recording's one asset body, put through the recording's
	// store, which the cast's proxy now serves to viewers (#459).
	if o.Assets == nil {
		t.Fatal("no asset store for the recording")
	}
	a, err := o.Assets.Put(context.Background(), castassets.Capture{URL: "http://x/a.png", MIME: "image/png", Body: []byte("a")})
	if err != nil {
		t.Fatal(err)
	}
	resp, err := castClient.Get(srv.URL + "/" + c.token + "/assets/" + a.Hash)
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode != 200 || string(body) != "a" {
		t.Fatalf("proxy: %d %q", resp.StatusCode, body)
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
		for ev == "capabilities" || ev == "feedback" {
			ev, data = s.next()
		}
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

// When a recorded cast ends, the fidelity check (#478) runs on its
// recording if the recording has frames, and its summary or error reaches
// the operator; a cast never live, or with no frame, isn't checked.
func TestCastChecksItsRecordingWhenItEnds(t *testing.T) {
	type call struct{ dir string }
	for _, c := range []struct {
		name    string
		live    bool
		frame   bool
		err     error
		want    string
		checked bool
	}{
		{name: "frames", live: true, frame: true, want: "fidelity median 0.10%", checked: true},
		{name: "error", live: true, frame: true, err: fmt.Errorf("no Chrome"), want: "fidelity check: no Chrome", checked: true},
		{name: "no frame", live: true},
		{name: "never live"},
	} {
		t.Run(c.name, func(t *testing.T) {
			dir := t.TempDir()
			f := NewFakeBrowser()
			var ro *RecordOptions
			calls := make(chan call, 2)
			logged := make(chan string, 4)
			cast, err := StartCast(context.Background(), CastConfig{
				Launch:        func(_ context.Context, r *RecordOptions) (Browser, error) { ro = r; return f, nil },
				Host:          broadcast.NewHost(broadcast.HostConfig{Tunnel: &broadcast.FakeTunnel{}, ProbeTimeout: 2 * time.Second}),
				RecordingsDir: dir,
				Logf:          func(format string, args ...any) { logged <- fmt.Sprintf(format, args...) },
				Check: func(_ context.Context, d string) (string, error) {
					calls <- call{d}
					return "fidelity median 0.10%", c.err
				},
			})
			if err != nil {
				t.Fatal(err)
			}
			if c.live { // what the recorder does at go-live and at its frames
				o, err := ro.Out()
				if err != nil {
					t.Fatal(err)
				}
				for _, r := range recording.ContractRecording(0)[:1] {
					if err := o.Sink.Put(r); err != nil {
						t.Fatal(err)
					}
				}
				if c.frame {
					if err := o.Frames.PutFrame(&recording.Frame{File: "frames/0.png", W: 1, H: 1, Scale: 0.5}, []byte("png")); err != nil {
						t.Fatal(err)
					}
				}
				o.Sink.Close()
			}
			f.QuitByHand()
			<-cast.Done()
			if !c.checked {
				select {
				case got := <-calls:
					t.Fatalf("checked %s", got.dir)
				case <-time.After(100 * time.Millisecond):
				}
				return
			}
			var got call
			select {
			case got = <-calls:
			case <-time.After(5 * time.Second):
				t.Fatal("the recording was never checked")
			}
			if got.dir != filepath.Join(dir, ro.ID) {
				t.Errorf("checked %s, want the cast's recording", got.dir)
			}
			select {
			case msg := <-logged:
				if msg != c.want {
					t.Errorf("logged %q, want %q", msg, c.want)
				}
			case <-time.After(5 * time.Second):
				t.Fatal("nothing logged")
			}
		})
	}
}
