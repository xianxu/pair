package drive

import (
	"errors"
	"strings"
	"testing"
)

func TestParseBatchAcceptsEveryKind(t *testing.T) {
	body := `{"seg":3,"acts":[
		{"a":"down","id":12,"x":10.5,"y":20},
		{"a":"move","x":11,"y":21,"button":"left"},
		{"a":"up","x":12,"y":22},
		{"a":"click","id":12,"x":10,"y":20,"count":2},
		{"a":"wheel","id":4,"x":1,"y":1,"dy":120},
		{"a":"key","type":"down","key":"z","code":"KeyZ","mods":["meta"]},
		{"a":"text","v":"héllo"},
		{"a":"ime","v":"にほ","sel":[0,2]}]}`
	b, err := ParseBatch([]byte(body))
	if err != nil {
		t.Fatal(err)
	}
	if b.Seg != 3 || len(b.Acts) != 8 || b.Acts[0].Button != "left" || b.Acts[3].Count != 2 || b.Acts[2].Button != "left" {
		t.Fatalf("parsed %+v", b)
	}
	if b.Acts[4].Count != 0 || b.Acts[5].Mods[0] != ModMeta || *b.Acts[7].Sel != [2]int{0, 2} {
		t.Fatalf("parsed %+v", b)
	}
}

func TestParseBatchRefuses(t *testing.T) {
	many := `{"seg":1,"acts":[` + strings.TrimSuffix(strings.Repeat(`{"a":"text","v":"x"},`, MaxActs+1), ",") + `]}`
	long := `{"seg":1,"acts":[{"a":"text","v":"` + strings.Repeat("x", MaxText+1) + `"}]}`
	for _, c := range []struct {
		name, body string
		want       error
	}{
		{"not json", `{`, ErrBatchShape},
		{"unknown field", `{"seg":1,"acts":[{"a":"text","v":"x"}],"x":1}`, ErrBatchShape},
		{"unknown act field", `{"seg":1,"acts":[{"a":"text","v":"x","eval":"1"}]}`, ErrBatchShape},
		{"two objects", `{"seg":1,"acts":[{"a":"text","v":"x"}]} {}`, ErrBatchShape},
		{"no seg", `{"acts":[{"a":"text","v":"x"}]}`, ErrBatchShape},
		{"no acts", `{"seg":1,"acts":[]}`, ErrBatchShape},
		{"unknown kind", `{"seg":1,"acts":[{"a":"paste"}]}`, ErrBatchShape},
		{"click without target", `{"seg":1,"acts":[{"a":"click","x":1,"y":1}]}`, ErrBatchShape},
		{"negative coordinate", `{"seg":1,"acts":[{"a":"move","x":-1,"y":1}]}`, ErrBatchShape},
		{"huge coordinate", `{"seg":1,"acts":[{"a":"move","x":1e12,"y":1}]}`, ErrBatchShape},
		{"triple click", `{"seg":1,"acts":[{"a":"click","id":1,"x":1,"y":1,"count":3}]}`, ErrBatchShape},
		{"key without type", `{"seg":1,"acts":[{"a":"key","key":"a"}]}`, ErrBatchShape},
		{"unknown modifier", `{"seg":1,"acts":[{"a":"key","type":"down","key":"a","mods":["hyper"]}]}`, ErrBatchShape},
		{"empty text", `{"seg":1,"acts":[{"a":"text","v":""}]}`, ErrBatchShape},
		{"too many acts", many, ErrBatchLimit},
		{"text too long", long, ErrBatchLimit},
		{"body too large", `{"seg":1,"acts":[{"a":"text","v":"` + strings.Repeat("x", MaxBody) + `"}]}`, ErrBatchLimit},
	} {
		_, err := ParseBatch([]byte(c.body))
		if !errors.Is(err, c.want) {
			t.Errorf("%s: got %v, want %v", c.name, err, c.want)
		}
		if err != nil && strings.Contains(err.Error(), "eval") {
			t.Errorf("%s: the error echoes the input: %v", c.name, err)
		}
	}
}
