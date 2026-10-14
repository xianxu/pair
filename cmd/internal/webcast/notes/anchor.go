package notes

import "strings"

// AnchorCandidate is bounded identifying evidence, independent of a DOM.
// Path ends with the element itself; its preceding step is its nearest container.
type AnchorCandidate struct {
	Key     string            `json:"key,omitempty"`
	Tag     string            `json:"tag,omitempty"`
	Stable  map[string]string `json:"stable,omitempty"`
	Text    string            `json:"text,omitempty"`
	Path    []string          `json:"path,omitempty"`
	Classes []string          `json:"classes,omitempty"`
	Route   string            `json:"route,omitempty"`
}

const MaxAnchorCandidates = 2000

func meaningfulText(s string) bool {
	if len([]rune(s)) < 3 {
		return false
	}
	switch strings.ToLower(s) {
	case "save", "cancel", "ok", "yes", "no", "next", "back", "close", "delete", "edit", "more", "submit", "continue", "learn more", "click here":
		return false
	}
	return true
}

// ScoreAnchor rejects conflicting identity even for a still-connected node.
// Replacement selection separately requires a known tag and sufficient evidence.
func ScoreAnchor(a, c AnchorCandidate) int {
	if a.Tag != "" && a.Tag != c.Tag {
		return -1
	}
	score := 0
	identity := false
	for _, k := range []string{"id", "data-testid", "data-test", "data-cy"} {
		if v := a.Stable[k]; v != "" {
			if c.Stable[k] != v {
				return -1
			}
			identity = true
		}
	}
	if identity {
		score += 50
	}
	text := trim(a.Text)
	meaningful := meaningfulText(text)
	if meaningful {
		if text != trim(c.Text) {
			return -1
		}
		score += 40
	}
	if !identity && !meaningful {
		return 0
	}
	if len(a.Path) > 1 && len(c.Path) > 1 && a.Path[len(a.Path)-2] == c.Path[len(c.Path)-2] {
		score += 20
	}
	seen := map[string]bool{}
	for _, s := range a.Classes {
		if s != "" {
			seen[s] = true
		}
	}
	overlap := 0
	for _, s := range c.Classes {
		if seen[s] {
			overlap++
			delete(seen, s)
		}
	}
	return score + min(10, overlap*5)
}

// ChooseAnchor requires a strong winner and a complete bounded enumeration.
func ChooseAnchor(a AnchorCandidate, candidates []AnchorCandidate, truncated bool, route string) (AnchorCandidate, bool) {
	if truncated || len(candidates) > MaxAnchorCandidates || a.Tag == "" || (a.Route != "" && a.Route != route) {
		return AnchorCandidate{}, false
	}
	best, runner := -1, -1
	var winner AnchorCandidate
	for _, c := range candidates {
		s := ScoreAnchor(a, c)
		if s > best {
			runner = best
			best = s
			winner = c
		} else if s > runner {
			runner = s
		}
	}
	if best < 60 || best-runner < 20 {
		return AnchorCandidate{}, false
	}
	return winner, true
}
