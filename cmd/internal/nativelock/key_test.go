package nativelock

import (
	"path/filepath"
	"strings"
	"testing"
	"unicode"
	"unicode/utf8"

	"github.com/xianxu/pair/cmd/internal/artifactpath"
)

func FuzzKeyAndHolder(f *testing.F) {
	for _, id := range []string{"conversation", "世界:1", "../escape", "\x1b[31m", "e\u0301", strings.Repeat("x", 1025)} {
		f.Add(id)
	}
	f.Fuzz(func(t *testing.T, id string) {
		key, err := NewKey("codex", id)
		if err == nil {
			if key.ID() != id {
				t.Fatal("identity changed")
			}
			filename, err := artifactpath.NativeLockFilename(key.Agent(), key.ID())
			if err != nil || filepath.Base(filename) != filename || len(filename) > 255 {
				t.Fatalf("unsafe filename %q %v", filename, err)
			}
		}
		holder := ParseHolder([]byte(id))
		if len(holder) > 1024 || !utf8.ValidString(holder) {
			t.Fatalf("unsafe holder %q", holder)
		}
		for _, r := range holder {
			if unicode.IsControl(r) || unicode.Is(unicode.Cf, r) {
				t.Fatalf("terminal control %q", holder)
			}
		}
	})
}
