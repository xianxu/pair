package agentdrive

import (
	"testing"
	"time"
)

func TestReduceDriveRawCommandEnterIsControl(t *testing.T) {
	now := time.Unix(100, 0)
	s, _ := NewDrive(Request{Keys: KeySequence{{Kind: KeyText, Text: "/usage"}, {Kind: KeyEnter}}}, "claude", now.Add(time.Second))
	s, fx := ReduceDrive(s, DriveEvent{Kind: EventStart, At: now, Screen: Screen{Epoch: 1, Complete: true, Idle: true}})
	if len(fx) != 1 || fx[0].Control {
		t.Fatal("ordinary raw text acquired control authority")
	}
	s, _ = ReduceDrive(s, DriveEvent{Kind: EventWritten, At: now, Screen: Screen{Epoch: 1}})
	_, fx = ReduceDrive(s, DriveEvent{Kind: EventScreen, At: now, Screen: Screen{Epoch: 2, Complete: true, Recoverable: true, CommandKnown: true, CommandText: "/usage"}})
	if len(fx) != 1 || !fx[0].CommandEnter() || !fx[0].Control {
		t.Fatalf("proved command Enter opens conversational turn: %#v", fx)
	}
	raw, _ := NewDrive(Request{Keys: KeySequence{{Kind: KeySubmit}}}, "claude", now.Add(time.Second))
	_, fx = ReduceDrive(raw, DriveEvent{Kind: EventStart, At: now, Screen: Screen{Epoch: 1, Complete: true, Idle: true}})
	if len(fx) != 1 || fx[0].Control || fx[0].CommandEnter() {
		t.Fatal("ordinary public submit lost conversational semantics")
	}
}
