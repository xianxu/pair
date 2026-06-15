// pair-changelog — distill a pair session's TTY into an append-mostly change
// log (issue #53). Invoked on demand by bin/pair-changelog-open (Alt+l): it
// reads the cleaned TTY the shell produced, decides what is new since the last
// run (a content anchor + turn-based lookback), asks the session's agent model
// to distill it, and assembles the new log — preserving prior entries
// byte-for-byte (only the last entry is ever model-revised).
//
// All decision logic is pure (distill.go); this file is the thin IO seam:
// read files → model.Run → atomic write (log first, then anchor).
package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/xianxu/pair/cmd/internal/model"
)

const (
	lookbackTurns = 2    // turns of context to re-include before the boundary
	lineCap       = 200  // safety cap on the lookback slice (verbose-turn guard)
	anchorLines   = 3    // K: verbatim cleaned-lines stored as the content anchor
	maxTokens     = 2000 // generous output budget (multi-entry log)
	maxSliceLines = 2000 // hard cap on lines fed to the model per batch (timeout
	//                     guard, #58). 2000 < the ~3000+ that tripped the old 30s
	//                     timeout, and the budget is now 90s — comfortable headroom
	//                     while halving the batch count on long transcripts (#59).
	// changelogTimeout — `claude -p` has a ~28s baseline (CLI startup + model), so
	// the slug's 30s default is too tight for this heavier, on-demand distill; the
	// viewer runs it async behind a spinner, so a longer budget is fine (#58).
	changelogTimeout = 90 * time.Second
)

func fail(format string, a ...any) {
	fmt.Fprintf(os.Stderr, "pair-changelog: "+format+"\n", a...)
	os.Exit(1)
}

func main() {
	var cleanedPath, logPath, anchorPath, agent, modelName string
	flag.StringVar(&cleanedPath, "cleaned", "", "path to the cleaned-TTY text file")
	flag.StringVar(&logPath, "log", "", "path to the change-log markdown file")
	flag.StringVar(&anchorPath, "anchor", "", "path to the content-anchor sidecar")
	flag.StringVar(&agent, "agent", "claude", "session agent (claude|codex|agy)")
	flag.StringVar(&modelName, "model", "", "model override; default per-agent")
	flag.Parse()

	if cleanedPath == "" || logPath == "" || anchorPath == "" {
		fail("usage: pair-changelog --cleaned F --log F --anchor F [--agent A]")
	}

	cleanedBytes, err := os.ReadFile(cleanedPath)
	if err != nil {
		fail("read cleaned: %v", err)
	}
	// Parse the render's ⟦pair:ts DATE⟧ markers into per-line dates and strip them
	// so the anchor/slice/turn-count/model-input all see clean content (#59). Then
	// trim the volatile live UI footer (empty prompt box + rule/status) so the
	// anchor/slice/turn-count work on stable committed scrollback (#58); keep
	// `dates` aligned to the trimmed content (trimLiveTail only removes the tail).
	content, dates := parseDatedLines(splitLines(string(cleanedBytes)))
	content = trimLiveTail(content, agent)
	dates = dates[:len(content)]
	if len(content) == 0 {
		return // nothing captured yet; leave the log untouched
	}

	priorLog := readFileOr(logPath)
	priorTurns, anchor := parseAnchor(readFileOr(anchorPath))
	boundaries := scanTurnBoundaries(content, agent)
	hasPrior := strings.TrimSpace(priorLog) != ""

	res := locate(content, anchor, boundaries, lookbackTurns, lineCap)

	// Turn-count no-op: the change log only gains entries when the agent
	// completes a new turn (a new user-prompt boundary). The volatile trailing
	// prompt/status lines churn between presses, so a byte-level check would
	// re-distill every press; counting completed turns is robust to that noise
	// and means "nothing changed → no model call".
	//
	// Guarded by res.Kind != FullRedistill: a FullRedistill means the anchor is
	// gone — either a first run, OR the agent session was reset (Alt+n), which
	// re-renders a fresh screen whose turn count is BELOW the stale anchor's
	// priorTurns. Without this guard a reset reads as "fewer turns → nothing
	// new" and the new session never distills (#58 follow-up: the anchor is a
	// per-session marker, so an absent anchor can't license a no-op).
	if hasPrior && res.Kind != FullRedistill && len(boundaries) <= priorTurns {
		fmt.Fprintln(os.Stderr, "pair-changelog: up to date (no new turn)")
		return
	}

	if res.Kind == NoOp {
		return // belt-and-suspenders; shouldn't fire once a new turn exists
	}

	sliceStart := 0
	if hasPrior {
		sliceStart = res.Start
	}
	slice := content[sliceStart:]
	sliceDates := dates[sliceStart:]

	// Split the slice into per-day segments (#59) so a multi-day slice distills
	// into multiple ## DATE sections, then batch each segment into maxSliceLines
	// chunks (#58) — a long slice is never truncated; 800 is just the per-call
	// batch size. distillStep dates each batch's new entries by the segment's day
	// (real change-time from the markers; "" → undated, no header). The running
	// log is carried as memory across segments AND batches (dedup + last-entry
	// revision). The log is written after each batch for progressive viewer reload;
	// the anchor only after the loop (below): if interrupted, it stays one-behind →
	// the next press re-distills and catches up, never skipping content.
	newLog := priorLog // "" on a first-ever run
	for _, seg := range splitByDate(slice, sliceDates) {
		chunks := chunkLines(seg.lines, maxSliceLines)
		for i, chunk := range chunks {
			if len(chunks) > 1 {
				fmt.Fprintf(os.Stderr, "pair-changelog: distilling batch %d/%d (%d lines)\n", i+1, len(chunks), len(chunk))
			} else {
				fmt.Fprintf(os.Stderr, "pair-changelog: distilling %d lines\n", len(chunk))
			}
			nl, err := distillStep(newLog, strings.Join(chunk, "\n"), agent, modelName, seg.date)
			if err != nil {
				fail("model: %v", err)
			}
			if nl == newLog {
				continue // this batch added nothing — no write
			}
			newLog = nl
			if err := atomicWrite(logPath, newLog); err != nil {
				fail("write log: %v", err)
			}
		}
	}

	// A first-ever run that yielded nothing has no committed content to anchor on.
	if strings.TrimSpace(newLog) == "" {
		return
	}
	// Advance the anchor to the turns/position we just processed — even when the
	// distill produced no textual change (a trivial turn the model added nothing
	// for). The anchor tracks "processed up to here," NOT "the log text changed":
	// advancing only on a text change would leave the turn count lagging behind
	// len(boundaries), so the turn-count no-op gate could never engage and every
	// later press would re-run the model — a regression of the exact #58 symptom.
	if err := writeAnchor(anchorPath, len(boundaries), anchorSnippet(content, anchorLines)); err != nil {
		fail("write anchor: %v", err)
	}
	if newLog == priorLog {
		return // processed, but no new entry → anchor advanced; nothing to flash
	}
	// Drop the "build complete" marker the draft nvim polls (#58). Reached only on
	// a real textual change, so the draft flashes its statusline only when a
	// triggered-and-left build actually produced something — not on a no-op press
	// or a trivial turn. Best-effort: the build already succeeded; the notification
	// is a bonus, so a write failure isn't fatal.
	writeReady(logPath)
}

// writeReady drops the "<base>.ready" marker beside the log. The draft nvim
// fs-watches $PAIR_DATA_DIR for it and, on arrival, flashes its statusline then
// deletes the marker (one-shot). The timestamp body is for debugging only — the
// marker's existence is the signal.
func writeReady(logPath string) {
	readyPath := strings.TrimSuffix(logPath, ".md") + ".ready"
	_ = os.WriteFile(readyPath, []byte(time.Now().Format(time.RFC3339)+"\n"), 0o644)
}

// distillStep runs one model distill — first-run (priorLog empty) or incremental
// (revise the last entry + append) — and returns the new full log. Returns
// priorLog unchanged when the model produces nothing; errors on a non-distill
// response (a hijacked continuation). Used per-chunk by the first-run batcher.
func distillStep(priorLog, sliceText, agent, modelName, date string) (string, error) {
	firstRun := strings.TrimSpace(priorLog) == ""
	var frozen, ek, sys, input string
	if firstRun {
		sys = buildSystemPrompt(true)
		input = buildInput("", "", sliceText, true)
	} else {
		frozen, ek = splitFrozenTail(priorLog)
		sys = buildSystemPrompt(false)
		input = buildInput(frozen, ek, sliceText, false)
	}
	out, err := model.Run(model.Request{
		Agent: agent, Model: modelName, Prompt: sys, Input: input,
		MaxOutputTokens: maxTokens, Verbosity: "medium", Timeout: changelogTimeout,
	})
	if err != nil {
		return "", err
	}
	out = strings.TrimSpace(out)
	if out == "" {
		return priorLog, nil // model produced nothing; no change this step
	}
	if !looksLikeChangelog(out) {
		return "", fmt.Errorf("non-distill response (no change-log entries)")
	}
	var ekPrime, newEntries string
	if firstRun {
		newEntries = out
	} else {
		ekPrime, newEntries = splitFirstEntry(out)
	}
	return assemble(frozen, ekPrime, newEntries, date, lastHeaderDate(priorLog)), nil
}

func writeAnchor(path string, turns int, snippet []string) error {
	var b strings.Builder
	fmt.Fprintf(&b, "turns:%d\n", turns)
	for _, l := range snippet {
		b.WriteString(l)
		b.WriteString("\n")
	}
	return atomicWrite(path, b.String())
}

// splitLines splits cleaned text into lines, dropping any trailing newlines so a
// file ending in one or more blank lines doesn't yield spurious empty elements.
func splitLines(s string) []string {
	s = strings.TrimRight(s, "\n")
	if s == "" {
		return nil
	}
	return strings.Split(s, "\n")
}

func readFileOr(path string) string {
	b, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return string(b)
}

func atomicWrite(path, content string) error {
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, []byte(content), 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}
