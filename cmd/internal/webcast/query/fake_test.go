package query

import (
	"testing"

	"github.com/xianxu/pair/cmd/internal/webcast/notes"
	"github.com/xianxu/pair/cmd/internal/webcast/recording"
)

func TestFakeSurfaceMeetsTheSurfaceContract(t *testing.T) {
	RunSurfaceContract(t, func(recs []recording.Record) Surface {
		return FakeSurface{Records: recs, Model: notes.FakeModel{}}
	})
}
