package couchtty

import (
	"fmt"
	"reflect"
	"testing"
)

func TestSplitCompletionPathPreservesEditableSpelling(t *testing.T) {
	tests := []struct {
		path      string
		directory string
		prefix    string
		name      string
		hidden    bool
		immediate string
		entry     string
		completed string
	}{
		{path: "", directory: ".", entry: "src", completed: "src/"},
		{path: ".", immediate: "./"},
		{path: "..", immediate: "../"},
		{path: "./", directory: "./", prefix: "./", entry: "src", completed: "./src/"},
		{path: "../sr", directory: "../", prefix: "../", name: "sr", entry: "src", completed: "../src/"},
		{path: "/repo/sr", directory: "/repo/", prefix: "/repo/", name: "sr", entry: "src", completed: "/repo/src/"},
		{path: "foo//ba", directory: "foo//", prefix: "foo//", name: "ba", entry: "bar", completed: "foo//bar/"},
		{path: "/", directory: "/", prefix: "/", entry: "src", completed: "/src/"},
		{path: "~", directory: ".", name: "~", entry: "~repo", completed: "~repo/"},
		{path: ".ca", directory: ".", name: ".ca", hidden: true, entry: ".cache", completed: ".cache/"},
	}

	for _, test := range tests {
		t.Run(test.path, func(t *testing.T) {
			got := SplitCompletionPath(test.path)
			if got.Directory != test.directory || got.EditablePrefix != test.prefix || got.NamePrefix != test.name || got.IncludeHidden != test.hidden || got.Immediate != test.immediate {
				t.Fatalf("SplitCompletionPath(%q) = %+v", test.path, got)
			}
			if test.entry != "" && got.CompletedPath(test.entry) != test.completed {
				t.Fatalf("CompletedPath(%q) = %q, want %q", test.entry, got.CompletedPath(test.entry), test.completed)
			}
		})
	}
}

func TestCompletionAccumulatorFiltersSortsAndBounds(t *testing.T) {
	query := SplitCompletionPath("s")
	accumulator := NewCompletionAccumulator(query, menuCompletionLimit)
	accumulator.Add([]CompletionEntry{
		{Name: "src", Directory: true},
		{Name: "sample", Directory: true},
		{Name: "setup", Directory: false},
		{Name: ".secret", Directory: true},
	})
	if got := accumulator.Result(); !reflect.DeepEqual(got, CompletionMatches{Paths: []string{"sample/", "src/"}}) {
		t.Fatalf("filtered matches = %+v", got)
	}

	hidden := NewCompletionAccumulator(CompletionQuery{Directory: ".", NamePrefix: ".", IncludeHidden: true}, menuCompletionLimit)
	hidden.Add([]CompletionEntry{{Name: ".zeta", Directory: true}, {Name: "plain", Directory: true}})
	hidden.Add([]CompletionEntry{{Name: ".alpha", Directory: true}, {Name: ".env", Directory: false}})
	if got := hidden.Result(); !reflect.DeepEqual(got.Paths, []string{".alpha/", ".zeta/"}) || got.Truncated {
		t.Fatalf("hidden matches = %+v", got)
	}

	bounded := NewCompletionAccumulator(SplitCompletionPath(""), menuCompletionLimit)
	for i := menuCompletionLimit; i >= 0; i-- {
		bounded.Add([]CompletionEntry{{Name: fmt.Sprintf("d%03d", i), Directory: true}})
	}
	got := bounded.Result()
	if len(got.Paths) != menuCompletionLimit || got.Paths[0] != "d000/" || got.Paths[len(got.Paths)-1] != "d199/" || !got.Truncated {
		t.Fatalf("bounded matches = len %d first/last %q/%q truncated %v", len(got.Paths), got.Paths[0], got.Paths[len(got.Paths)-1], got.Truncated)
	}
}
