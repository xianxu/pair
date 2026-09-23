package couchcore

import "testing"

func TestSelectWorkspaceNumber(t *testing.T) {
	tests := []struct {
		name string
		rows []WorkspaceCandidate
		want int
		fail bool
	}{
		{"empty", nil, 1, false},
		{"ready before missing", []WorkspaceCandidate{{4, WorkspaceReady}, {2, WorkspaceReady}}, 2, false},
		{"sparse", []WorkspaceCandidate{{1, WorkspaceOccupied}, {3, WorkspaceOccupied}}, 2, false},
		{"absent", []WorkspaceCandidate{{1, WorkspaceOccupied}, {2, WorkspaceAbsent}}, 2, false},
		{"duplicate", []WorkspaceCandidate{{1, WorkspaceReady}, {1, WorkspaceReady}}, 0, true},
		{"conflict", []WorkspaceCandidate{{1, WorkspaceReady}, {1, WorkspaceOccupied}}, 0, true},
		{"unknown blocks", []WorkspaceCandidate{{1, WorkspaceReady}, {2, WorkspaceUnknown}}, 0, true},
		{"partial blocks", []WorkspaceCandidate{{2, WorkspacePartial}}, 0, true},
		{"zero", []WorkspaceCandidate{{0, WorkspaceReady}}, 0, true},
		{"invalid", []WorkspaceCandidate{{1, "bogus"}}, 0, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := SelectWorkspaceNumber(tt.rows)
			if (err != nil) != tt.fail || got != tt.want {
				t.Fatalf("got %d %v want %d fail=%v", got, err, tt.want, tt.fail)
			}
		})
	}
}
