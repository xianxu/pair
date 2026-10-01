package couchcmd

import (
	"reflect"
	"testing"

	"github.com/xianxu/pair/cmd/internal/couchcore"
)

func TestMessageFamiliesWithholdSharedAliases(t *testing.T) {
	got := messageFamilies([]couchcore.RepositoryName{
		{Key: "/w/xianxu.dev", Dir: "xianxu.dev", Alias: "blog"},
		{Key: "/w/pair", Dir: "pair"},
		{Key: "/a/parley", Dir: "parley", Alias: "pa"},
		{Key: "/b/parley", Dir: "parley", Alias: "pb"},
	})
	if want := map[string]string{"xianxu.dev": "blog", "pair": "", "parley": ""}; !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v want %v", got, want)
	}
}
