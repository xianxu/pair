package dispatcher

import "testing"

func TestRetentionRoutesToStreamingRuntime(t *testing.T) {
	family, rest, ok := Resolve([]string{"retention", "register", "--pid", "123"})
	if !ok || family.Name != "retention" || !family.Streaming || len(rest) != 3 || rest[0] != "register" {
		t.Fatalf("missing internal retention route: %+v %v %v", family, rest, ok)
	}
}
