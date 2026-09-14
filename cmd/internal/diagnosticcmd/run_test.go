package diagnosticcmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAppendCommandUsesExactPath(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "trace")
	var stderr bytes.Buffer
	if code := Run([]string{"--path", path}, func(string) string { return "" }, strings.NewReader("record\n"), &stderr); code != 0 {
		t.Fatalf("%d %s", code, &stderr)
	}
	got, err := os.ReadFile(path)
	if err != nil || string(got) != "record\n" {
		t.Fatalf("%q %v", got, err)
	}
	for _, args := range [][]string{nil, {"--path"}, {"--path", path, "extra"}} {
		if Run(args, func(string) string { return "" }, strings.NewReader("bad"), &stderr) == 0 {
			t.Fatal("accepted invalid command")
		}
	}
}
