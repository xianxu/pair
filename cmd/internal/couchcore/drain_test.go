package couchcore

import (
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestDrainFlagsAddClearAndStewardExclusion(t *testing.T) {
	for _, n := range []int{8, 16} {
		selected := []string{"pair:0", "", "pair:1"}
		for i := 1; i <= n; i++ {
			selected = append(selected, fmt.Sprintf("pair:%d", i))
		}
		old := DrainFlags{"other:1": true, "pair:0": true}
		flags := AddDrainFlags(old, selected, "pair:0")
		if len(flags) != n+1 || flags["pair:0"] || !flags["other:1"] {
			t.Fatalf("flags: %#v", flags)
		}
		flags["other:1"] = false
		if !old["other:1"] {
			t.Fatal("add mutated source")
		}
		flags = AddDrainFlags(flags, selected, "pair:0")
		cleared := ClearDrainFlags(flags, []string{"pair:1"})
		if cleared["pair:1"] || !flags["pair:1"] {
			t.Fatal("clear mutated source or left selected flag")
		}
		if len(ClearDrainFlags(flags, nil)) != 0 {
			t.Fatal("clear all retained flags")
		}
	}
}

func TestDrainBoardUnknownClockAndIsolation(t *testing.T) {
	now := time.Unix(100, 0)
	yes, no := true, false
	flags := DrainFlags{"pair:3": true, "pair:1": true, "pair:2": true, "pair:4": true, "pair:5": false}
	obs := map[string]DrainObservation{
		"pair:1": {Connected: true, LastOutputAt: now.Add(-1500 * time.Millisecond), Settled: &yes, Picker: &no, Background: &no},
		"pair:2": {Connected: true, LastOutputAt: now.Add(time.Second), Settled: &yes, Picker: &no, Background: &no},
		"pair:3": {Connected: false, LastOutputAt: now.Add(-time.Minute), Settled: &yes, Picker: &no, Background: &no},
		"pair:4": {Connected: true, Settled: &yes, Picker: &no, Background: &no},
	}
	board := ProjectDrainBoard(flags, obs, now)
	if len(board.Rows) != 4 {
		t.Fatal(board)
	}
	row := board.Rows[0]
	if row.Workspace != "pair:1" || row.IdleMillis == nil || *row.IdleMillis != 1500 || row.Settled == nil || !*row.Settled {
		t.Fatal(row)
	}
	for _, r := range board.Rows[1:] {
		if r.IdleMillis != nil {
			t.Fatalf("invented idle: %+v", r)
		}
	}
	*row.Settled = false
	*row.Picker = true
	*row.Background = false
	if !yes || no {
		t.Fatal("observation pointer leaked")
	}
	*row.IdleMillis = 0
	if got := ProjectDrainBoard(flags, obs, now); *got.Rows[0].IdleMillis != 1500 {
		t.Fatal("board aliased")
	}
	if !reflect.DeepEqual(flags, DrainFlags{"pair:3": true, "pair:1": true, "pair:2": true, "pair:4": true, "pair:5": false}) {
		t.Fatal("projection mutated flags")
	}
}

func TestGracefulDrainNoticeLeavesQueuedWorkOutstanding(t *testing.T) {
	want := "Drain: finish your current atomic step and wait for background work that is already running to finish. Leave queued, not-started work queued and note it in the issue Log as outstanding. Write where you stopped and the next step in the issue Log; commit and push the checkpoint, then end your turn and stay idle. Start no new work. While idle, keep receiving messages. For new work, reply once: 'acknowledge message received, will resume processing after drain is lifted'. Do not create courtesy acknowledgements or acknowledgement loops."
	if got := DrainNotice(false); got != want {
		t.Fatalf("graceful notice must distinguish running work from outstanding queued work:\ngot:  %s\nwant: %s", got, want)
	}
}

func TestDrainNoticesAndUndrainValidation(t *testing.T) {
	for _, fast := range []bool{false, true} {
		notice := DrainNotice(fast)
		for _, phrase := range []string{"issue Log", "next step", "commit and push", "stay idle", "keep receiving messages", "acknowledge message received, will resume processing after drain is lifted", "courtesy acknowledgements"} {
			if !strings.Contains(notice, phrase) {
				t.Fatalf("missing %q in %q", phrase, notice)
			}
		}
		if strings.Contains(notice, "--ack") {
			t.Fatal("retained ack command")
		}
	}
	if !strings.Contains(DrainNotice(false), "wait for background work that is already running") {
		t.Fatal("graceful background guidance missing")
	}
	for _, phrase := range []string{"stop your own background work", "supported controls", "unfinished"} {
		if !strings.Contains(DrainNotice(true), phrase) {
			t.Fatal("fast missing " + phrase)
		}
	}
	if got, err := UndrainNotice(""); err != nil || got != "drain lifted; continue your current work\nIf you received no drain notice, no checkpoint or recovery action is required." {
		t.Fatal(got, err)
	}
	if got, err := UndrainNotice("next task"); err != nil || !strings.HasSuffix(got, "\nNote: \"next task\"") {
		t.Fatal(got, err)
	}
	for _, s := range []string{"\x1b[2J", "a\nb", string([]byte{0xff}), strings.Repeat("é", 513), "\u0085"} {
		if _, err := UndrainNotice(s); err == nil {
			t.Fatalf("accepted %q", s)
		}
	}
	if _, err := UndrainNotice(strings.Repeat("é", 512)); err != nil {
		t.Fatal(err)
	}
}

func TestDrainBoardIdleRequiresKnownQuiescentFacts(t *testing.T) {
	now := time.Unix(100, 0)
	yes, no := true, false
	for _, facts := range []struct {
		name                        string
		settled, picker, background *bool
	}{
		{"active", &no, &no, &no},
		{"picker", &yes, &yes, &no},
		{"background", &yes, &no, &yes},
		{"unknown settled", nil, &no, &no},
		{"unknown picker", &yes, nil, &no},
		{"unknown background", &yes, &no, nil},
	} {
		t.Run(facts.name, func(t *testing.T) {
			board := ProjectDrainBoard(DrainFlags{"pair:1": true}, map[string]DrainObservation{"pair:1": {Connected: true, LastOutputAt: now.Add(-time.Minute), Settled: facts.settled, Picker: facts.picker, Background: facts.background}}, now)
			if board.Rows[0].IdleMillis != nil {
				t.Fatalf("active or unknown facts fabricated idle: %+v", board.Rows[0])
			}
		})
	}
}
