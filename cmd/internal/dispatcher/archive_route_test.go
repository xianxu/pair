package dispatcher

import "testing"

func TestArchiveRoutesToStreamingRuntime(t *testing.T) {
	f, rest, ok := Resolve([]string{"archive", "inventory"})
	if !ok || f.Name != "archive" || !f.Streaming || len(rest) != 1 || rest[0] != "inventory" {
		t.Fatalf("route=%+v %v %v", f, rest, ok)
	}
}
