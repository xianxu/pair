package diagnosticlog

import (
	"context"
	"errors"
	"testing"
)

type failedSweep struct {
	Inspection
	err error
}

func (s failedSweep) Snapshot(context.Context, []string) (Inspection, error) { return nil, s.err }

func TestInspectionFailureIsDistinctFromLocalHolderBlock(t *testing.T) {
	denied := errors.New("native inventory incomplete")
	for _, tt := range []struct {
		name   string
		probe  Inspection
		reader bool
	}{
		{"files", &observedInspection{fileErr: denied}, false},
		{"processes", &observedInspection{holders: map[string][]Registration{"/log": nil}, runtimeErr: denied}, false},
		{"reader files", &observedInspection{fileErr: denied}, true},
		{"snapshot", failedSweep{err: denied}, false},
		{"reader snapshot", failedSweep{err: denied}, true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ctx := SweepContext(context.Background(), []string{"/log"})
			var err error
			if tt.reader {
				err = InspectionReaderProof(tt.probe)(ctx, "/log")
			} else {
				err = InspectionProof(tt.probe)(ctx, "/log", nil)
			}
			if !errors.Is(err, ErrInspectionUnavailable) || !errors.Is(err, denied) {
				t.Fatalf("native failure lost class or cause: %v", err)
			}
		})
	}
	held := &observedInspection{holders: map[string][]Registration{"/log": {{PID: 123, Birth: "reader"}}}}
	for _, err := range []error{InspectionProof(held)(context.Background(), "/log", nil), InspectionReaderProof(held)(context.Background(), "/log")} {
		if !errors.Is(err, ErrUnknownWriters) || errors.Is(err, ErrInspectionUnavailable) {
			t.Fatalf("complete holder observation misclassified: %v", err)
		}
	}
	for _, err := range []error{InspectionProof(nil)(context.Background(), "/log", nil), InspectionReaderProof(nil)(context.Background(), "/log")} {
		if !errors.Is(err, ErrInspectionUnavailable) {
			t.Fatalf("missing inspector treated as local block: %v", err)
		}
	}
}
