// Package opener is the Go owner of pair's two floating-pane viewer launchers
// (#93 M2, ported from bin/pair-scrollback-open + bin/pair-changelog-open):
//
//   - Alt+/ scrollback: render the agent pane's captured PTY to ANSI and open a
//     read-only nvim viewer, positioned to the user's current scroll offset.
//   - Alt+l changelog: launch a DETACHED render+distill build (survives the
//     viewer closing) and open an nvim watcher that tails the distilled log.
//
// The nvim viewers (nvim/scrollback.lua, nvim/changelog.lua) stay native (#95
// boundary); this package owns the orchestration. Following the #78 sessionwatch
// template, the pure decisions live here and IO (zellij/nvim/exec/detach/fs) sits
// behind the Runtime seam.
package opener

import (
	"encoding/json"
	"regexp"
)

// sgrEscape matches an ANSI CSI SGR (and other CSI) escape so a rendered .ansi
// line compares byte-for-byte against a zellij dump-screen plain-text line.
var sgrEscape = regexp.MustCompile("\x1b\\[[0-9;]*[a-zA-Z]")

func stripSGR(s string) string { return sgrEscape.ReplaceAllString(s, "") }

// matchViewport finds the .ansi line (1-based) the user is currently looking at,
// by matching a zellij dump-screen (the pane's actual visible content, including
// zellij's scroll offset) against the rendered scrollback. Ported from
// pair-scrollback-open's awk scorer: index ansi lines (≥ 8 chars) → line numbers,
// derive candidate start offsets from each matching dump line, score each start
// by consecutive matches, and accept the best only if ≥ 50% of non-blank dump
// lines match. On ties it prefers the smaller start (deterministic — the shell's
// map iteration order was not). Returns (line, true) on a high-confidence match;
// the caller falls back to the renderer's own .viewport otherwise.
func matchViewport(dump, ansi []string) (int, bool) {
	an := len(ansi)

	// ansi line → 1-based line numbers (only substantial lines, to avoid short
	// -line false positives).
	idx := map[string][]int{}
	for j, line := range ansi {
		if line != "" && len(line) >= 8 {
			idx[line] = append(idx[line], j+1)
		}
	}

	// Candidate starts: for each substantial dump line that matches an ansi
	// line at 1-based position p (dump index i), "dump line 0 lands at ansi
	// line p-i" is a hypothesis.
	seen := map[int]bool{}
	for i, d := range dump {
		if d == "" || len(d) < 8 {
			continue
		}
		for _, p := range idx[d] {
			s := p - i
			if s < -an {
				continue
			}
			seen[s] = true
		}
	}

	// Non-blank dump lines (the match-fraction denominator).
	nb := 0
	for _, d := range dump {
		if d != "" {
			nb++
		}
	}

	bestScore, bestStart := -1, 0
	for s := range seen {
		score := 0
		for i, d := range dump {
			if d == "" {
				continue
			}
			j := s - 1 + i // 0-based ansi index
			if j < 0 || j >= an {
				continue
			}
			if ansi[j] == d {
				score++
			}
		}
		if score > bestScore || (score == bestScore && s < bestStart) {
			bestScore, bestStart = score, s
		}
	}

	if nb > 0 && bestScore*2 >= nb {
		s := bestStart
		if s < 1 {
			s = 1 // dump starts before .ansi line 1 → user is at the very top
		}
		return s, true
	}
	return 0, false
}

// resolveSessionID implements the #63 change-log keying: an explicit
// PAIR_SESSION_ID wins; else the per-tag config's session_id; else "" (the
// legacy unsuffixed base). configJSON is the raw config-<tag>-<agent>.json bytes
// (nil/empty when absent).
func resolveSessionID(envSID string, configJSON []byte) string {
	if envSID != "" {
		return envSID
	}
	if len(configJSON) == 0 {
		return ""
	}
	var c struct {
		SessionID string `json:"session_id"`
	}
	if json.Unmarshal(configJSON, &c) != nil {
		return ""
	}
	return c.SessionID
}

// changelogBase is the per-session change-log path stem: the sid suffix is
// appended only when resolved (fresh sessions branch; a resume reuses it).
func changelogBase(dataDir, tag, agent, sid string) string {
	base := dataDir + "/changelog-" + tag + "-" + agent
	if sid != "" {
		base += "-" + sid
	}
	return base
}

// distillerInner is the detached build pipeline: render the cleaned scrollback,
// then distill it into the change log + anchor. It references PCL_* env (set by
// distillerEnv) so the paths need no shell quoting — mirrors the shell exactly.
const distillerInner = `"$PCL_BIN" scrollback-render --plain --max-lines 0 --with-timestamps "$PCL_RAW" "$PCL_EVENTS" "$PCL_CLEANED" && "$PCL_BIN" changelog --cleaned "$PCL_CLEANED" --log "$PCL_LOG" --anchor "$PCL_ANCHOR" --agent "$PCL_AGENT"`

// distillerEnv builds the PCL_* KEY=VALUE environment the detached distiller
// reads (paths passed via env, never interpolated into the sh -c string).
func distillerEnv(binPath, raw, events, cleaned, log, anchor, agent string) []string {
	return []string{
		"PCL_BIN=" + binPath,
		"PCL_RAW=" + raw,
		"PCL_EVENTS=" + events,
		"PCL_CLEANED=" + cleaned,
		"PCL_LOG=" + log,
		"PCL_ANCHOR=" + anchor,
		"PCL_AGENT=" + agent,
	}
}
