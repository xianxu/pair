package mockserve

import (
	"strings"
	"testing"
)

func TestDevelopmentPageVisitsEachInputNodeOnce(t *testing.T) {
	previous := 0
	for _, n := range []int{64, 128} {
		visits := 0
		input := []byte("<html><head></head><body>" + strings.Repeat("<p>screen</p>", n) + "</body></html>")
		body, err := transformPageWithWork(input, Bootstrap{Version: "mock-1", Screen: "index", Revision: "1"}, "MOCK", func() { visits++ })
		if err != nil || !strings.Contains(string(body), "couch-mock-progress") {
			t.Fatalf("transform: %v", err)
		}
		if visits > 3*n+10 || visits < 2*n || previous > 0 && visits > 2*previous {
			t.Fatalf("%d paragraphs: %d visits (previous %d)", n, visits, previous)
		}
		previous = visits
	}
}
