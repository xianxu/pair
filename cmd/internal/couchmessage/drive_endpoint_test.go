package couchmessage

import (
	"context"
	"encoding/json"
	"github.com/xianxu/pair/cmd/internal/agentdrive"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"
)

func TestDriveEndpointClosedRequest(t *testing.T) {
	b := protocolBinding("pair:1")
	start := DriveStart{Request: agentdrive.Request{Keys: agentdrive.KeySequence{{Kind: agentdrive.KeyEnter}}}, Instance: "test-instance", Deadline: time.Now().Add(time.Second)}
	for _, op := range []string{"drive-observe", "drive-status", "drive-cancel", "drive-start"} {
		r := EndpointRequest{Op: op, Binding: b}
		if op != "drive-observe" {
			r.ID = "drive-test"
		}
		if op == "drive-start" {
			r.Drive = &start
		}
		if err := ValidateEndpointRequest(r); err != nil {
			t.Fatalf("%s: %v", op, err)
		}
		r.Message = &Message{}
		if ValidateEndpointRequest(r) == nil {
			t.Fatal("foreign message admitted")
		}
	}
	if ValidateEndpointRequest(EndpointRequest{Op: "tail", Binding: b, Lines: 10, Drive: &start}) == nil {
		t.Fatal("drive leaked into tail")
	}
}

func TestDriveEndpointStartOnceAndReadStatus(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	b := protocolBinding("pair:1")
	var starts atomic.Int32
	dir, err := os.MkdirTemp("/tmp", "drive-ep-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)
	server, err := StartServer(ctx, filepath.Join(dir, "drive.sock"), func(_ context.Context, raw []byte) ([]byte, error) {
		var r EndpointRequest
		if err := json.Unmarshal(raw, &r); err != nil {
			return nil, err
		}
		switch r.Op {
		case "drive-observe":
			return json.Marshal(EndpointResponse{DriveObservation: &DriveObservation{Instance: "test-instance", Idle: true}})
		case "drive-start":
			starts.Add(1)
			return json.Marshal(EndpointResponse{DriveResult: &agentdrive.Result{ID: r.ID}})
		case "drive-status", "drive-cancel":
			return json.Marshal(EndpointResponse{DriveResult: &agentdrive.Result{ID: r.ID, Outcome: agentdrive.OutcomeConfirmed, Tail: []string{"done"}}})
		}
		return nil, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	defer server.Close()
	e := RemoteEndpoint{Binding: b, socket: server.socket}
	obs, err := e.ObserveDrive(ctx)
	if err != nil || !obs.Idle {
		t.Fatalf("%+v %v", obs, err)
	}
	_, err = e.StartDrive(ctx, "drive-test", DriveStart{Request: agentdrive.Request{Keys: agentdrive.KeySequence{{Kind: agentdrive.KeyEnter}}}, Instance: obs.Instance, Deadline: time.Now().Add(time.Second)})
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 3; i++ {
		r, err := e.DriveStatus(ctx, "drive-test")
		if err != nil || !r.Terminal() {
			t.Fatalf("%+v %v", r, err)
		}
	}
	if starts.Load() != 1 {
		t.Fatalf("started %d times", starts.Load())
	}
}
