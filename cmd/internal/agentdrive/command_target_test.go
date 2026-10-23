package agentdrive

import (
	"testing"
	"time"
)

func TestCommandExpectationUsesProjectedTrailingSpaceRepresentation(t *testing.T) {
	cases := []struct{ authored, want string }{
		{"/model ", "/model"},
		{"/model    ", "/model"},
		{"/model opus  5.5   ", "/model opus  5.5"},
		{"/search café 猫  ", "/search café 猫"},
		{"/search 猫\u00a0  ", "/search 猫\u00a0"},
		{"/model\t ", "/model\t"},
		{" /model  ", ""},
		{"/model \n   ", ""},
		{"/model \r   ", ""},
		{"/model \x00   ", ""},
	}
	for _, tc := range cases {
		t.Run(tc.authored, func(t *testing.T) {
			got := commandExpectations(KeySequence{{Kind: KeyText, Text: tc.authored}, {Kind: KeyEnter}})
			if got[1] != tc.want {
				t.Fatalf("authored %q: target=%q want %q", tc.authored, got[1], tc.want)
			}
		})
	}
	chunks := commandExpectations(KeySequence{{Kind: KeyText, Text: "/mo"}, {Kind: KeyText, Text: "del "}, {Kind: KeyText, Text: "  "}, {Kind: KeyEnter}})
	if chunks[3] != "/model" {
		t.Fatalf("chunk target=%q", chunks[3])
	}
}

func TestReduceDriveCommandEnterMatchesProjectedTrailingSpaces(t *testing.T) {
	for _, tc := range []struct{ authored, projected string }{{"/model ", "/model"}, {"/model opus  5.5   ", "/model opus  5.5"}} {
		t.Run(tc.authored, func(t *testing.T) {
			now := time.Unix(100, 0)
			s, err := NewDrive(Request{Keys: KeySequence{{Kind: KeyText, Text: tc.authored}, {Kind: KeyEnter}}}, "claude", now.Add(time.Second))
			if err != nil {
				t.Fatal(err)
			}
			s, fx := ReduceDrive(s, DriveEvent{Kind: EventStart, At: now, Screen: Screen{Epoch: 1, Complete: true, Idle: true}})
			if len(fx) != 1 || fx[0].Key.Text != tc.authored {
				t.Fatalf("authored bytes changed: %#v", fx)
			}
			s, fx = ReduceDrive(s, DriveEvent{Kind: EventWritten, At: now, Screen: Screen{Epoch: 1}})
			for _, effect := range fx {
				if effect.Kind == EffectWrite {
					t.Fatal("Enter before echo")
				}
			}
			changed := Screen{Epoch: 2, Complete: true, Recoverable: true, CommandKnown: true, CommandText: tc.projected + "x"}
			s, fx = ReduceDrive(s, DriveEvent{Kind: EventScreen, At: now, Screen: changed})
			for _, effect := range fx {
				if effect.Kind == EffectWrite {
					t.Fatal("different command accepted")
				}
			}
			exact := Screen{Epoch: 3, Complete: true, Recoverable: true, CommandKnown: true, CommandText: tc.projected}
			_, fx = ReduceDrive(s, DriveEvent{Kind: EventScreen, At: now, Screen: exact})
			if len(fx) != 1 || !fx[0].CommandEnter() || !fx[0].Control || !fx[0].MatchesScreen("claude", exact) {
				t.Fatalf("projected echo rejected: %#v", fx)
			}
			if fx[0].MatchesScreen("claude", changed) {
				t.Fatal("effect accepted changed arguments")
			}
			for _, mutate := range []func(*Screen){func(s *Screen) { s.Recoverable = false }, func(s *Screen) { s.CommandKnown = false }, func(s *Screen) { s.Complete = false }, func(s *Screen) { s.TurnActive = true }, func(s *Screen) { s.CommandText = " " + s.CommandText }} {
				invalid := exact
				mutate(&invalid)
				if fx[0].MatchesScreen("claude", invalid) {
					t.Fatalf("invalid ownership/text accepted: %#v", invalid)
				}
			}
		})
	}
}
