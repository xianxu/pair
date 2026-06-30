package launcher

import (
	"fmt"
	"path/filepath"
	"strings"
)

// NormalizeTag returns Pair's canonical bare tag form.
func NormalizeTag(raw string) (string, error) {
	tag := strings.TrimPrefix(raw, "pair-")
	if tag == "" {
		return "", fmt.Errorf("empty tag")
	}
	for _, r := range tag {
		if (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			continue
		}
		return "", fmt.Errorf("tag %q contains invalid character %q", raw, r)
	}
	return tag, nil
}

// DefaultTag derives Pair's create-flow default tag from a cwd path.
func DefaultTag(cwd string) string {
	base := filepath.Base(cwd)
	if base == "." || base == string(filepath.Separator) {
		return "pair"
	}
	var b strings.Builder
	for _, r := range base {
		if (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			b.WriteRune(r)
		} else {
			b.WriteByte('_')
		}
	}
	if b.Len() == 0 {
		return "pair"
	}
	return b.String()
}
