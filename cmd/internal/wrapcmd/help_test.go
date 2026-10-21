package wrapcmd

import (
	"bytes"
	"strings"
	"testing"
)

func TestRunHelpExplainsProcessLifetime(t *testing.T) {
	for _, flag := range []string{"--help", "-h"} {
		var out, diag bytes.Buffer
		if code := Run([]string{flag}, strings.NewReader(""), &out, &diag); code != 0 || diag.Len() != 0 {
			t.Fatalf("help code=%d diag=%q", code, diag.String())
		}
		for _, want := range []string{"Usage: pair wrap", "Darwin", "closes its stdio", "exits or you explicitly stop", "owned process group"} {
			if !strings.Contains(out.String(), want) {
				t.Errorf("help missing %q: %s", want, out.String())
			}
		}
	}
}
