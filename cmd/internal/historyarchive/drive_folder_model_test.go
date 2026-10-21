package historyarchive

import "testing"

func TestDriveFolderSequencesRequirePositiveUniqueObservation(t *testing.T) {
	events := []DriveFolderEvent{{Kind: FolderBegin}, {Kind: FolderAbsent}, {Kind: FolderUnique, ID: "child"}, {Kind: FolderUnique}, {Kind: FolderAmbiguous}, {Kind: FolderCreate}, {Kind: FolderCreated}, {Kind: FolderFailure}, {Kind: FolderCancel}}
	queue := []DriveFolderState{{}}
	seen := map[DriveFolderState]bool{}
	for len(queue) > 0 {
		s := queue[0]
		queue = queue[1:]
		if seen[s] {
			continue
		}
		seen[s] = true
		for _, e := range events {
			next, effect, err := StepDriveFolder(s, e)
			if err != nil {
				if next != s || effect != FolderNoEffect {
					t.Fatal("invalid event changed state")
				}
				continue
			}
			if next.Phase() == FolderReady && s.Phase() != FolderReady && (e.Kind != FolderUnique || e.ID != "child") {
				t.Fatal("ready without positive evidence")
			}
			if effect == FolderMake && (s.creates != 0 || next.creates != 1) {
				t.Fatal("repeated create")
			}
			if e.Kind == FolderFailure || e.Kind == FolderCancel {
				if effect != FolderNoEffect {
					t.Fatal("effect after failure/cancel")
				}
			}
			if s.Phase() == FolderUnknown || s.Phase() == FolderConflict || s.Phase() == FolderCancelled || s.Phase() == FolderReady {
				if effect != FolderNoEffect {
					t.Fatal("late event emitted effect")
				}
			}
			if !seen[next] {
				queue = append(queue, next)
			}
		}
	}
	if len(seen) < 8 {
		t.Fatalf("only explored %d states", len(seen))
	}
}

func FuzzDriveFolderEvents(f *testing.F) {
	f.Add([]byte{0, 1, 4, 5, 2}, "child")
	f.Fuzz(func(t *testing.T, raw []byte, id string) {
		if len(raw) > 64 {
			raw = raw[:64]
		}
		s := DriveFolderState{}
		creates := 0
		for _, kind := range raw {
			next, effect, err := StepDriveFolder(s, DriveFolderEvent{Kind: DriveFolderEventKind(kind), ID: id})
			if err != nil {
				if next != s || effect != FolderNoEffect {
					t.Fatal("invalid transition mutated")
				}
				continue
			}
			if effect == FolderMake {
				creates++
				if creates > 1 {
					t.Fatal("created twice")
				}
			}
			if next.Phase() == FolderReady && !validRemoteIdentity(next.FolderID()) {
				t.Fatal("invalid ready identity")
			}
			s = next
		}
	})
}
