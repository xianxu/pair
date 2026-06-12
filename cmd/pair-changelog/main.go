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
	maxSliceLines = 800  // hard cap on lines fed to the model (timeout guard, #58)
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
	var cleanedPath, logPath, anchorPath, agent, today, modelName string
	flag.StringVar(&cleanedPath, "cleaned", "", "path to the cleaned-TTY text file")
	flag.StringVar(&logPath, "log", "", "path to the change-log markdown file")
	flag.StringVar(&anchorPath, "anchor", "", "path to the content-anchor sidecar")
	flag.StringVar(&agent, "agent", "claude", "session agent (claude|codex|agy)")
	flag.StringVar(&today, "today", time.Now().Format("2006-01-02"), "press date (testing hook)")
	flag.StringVar(&modelName, "model", "", "model override; default per-agent")
	flag.Parse()

	if cleanedPath == "" || logPath == "" || anchorPath == "" {
		fail("usage: pair-changelog --cleaned F --log F --anchor F [--agent A] [--today D]")
	}

	cleanedBytes, err := os.ReadFile(cleanedPath)
	if err != nil {
		fail("read cleaned: %v", err)
	}
	// Trim the volatile live UI (empty prompt box + rule/status) so the anchor,
	// slice, and turn-count work on stable committed scrollback (#58).
	lines := trimLiveTail(splitLines(string(cleanedBytes)), agent)
	if len(lines) == 0 {
		return // nothing captured yet; leave the log untouched
	}

	priorLog := readFileOr(logPath)
	priorTurns, anchor := parseAnchor(readFileOr(anchorPath))
	boundaries := scanTurnBoundaries(lines, agent)
	hasPrior := strings.TrimSpace(priorLog) != ""

	// Turn-count no-op: the change log only gains entries when the agent
	// completes a new turn (a new user-prompt boundary). The volatile trailing
	// prompt/status lines churn between presses, so a byte-level check would
	// re-distill every press; counting completed turns is robust to that noise
	// and means "nothing changed → no model call".
	if hasPrior && len(boundaries) <= priorTurns {
		fmt.Fprintln(os.Stderr, "pair-changelog: up to date (no new turn)")
		return
	}

	res := locate(lines, anchor, boundaries, lookbackTurns, lineCap)
	if res.Kind == NoOp {
		return // belt-and-suspenders; shouldn't fire once a new turn exists
	}

	sliceStart := 0
	if hasPrior {
		sliceStart = res.Start
	}
	slice := lines[sliceStart:]

	// Batch the slice into maxSliceLines-sized chunks and distill each in order,
	// accumulating the log as memory — so a long slice is never truncated to the
	// last maxSliceLines (#58). 800 is just the per-call batch size (a timeout
	// bound); this applies equally to a long first run AND a large gap on a later
	// press. distillStep switches first-run vs incremental on the running log.
	newLog := priorLog // "" on a first-ever run
	chunks := chunkLines(slice, maxSliceLines)
	for i, chunk := range chunks {
		if len(chunks) > 1 {
			fmt.Fprintf(os.Stderr, "pair-changelog: distilling batch %d/%d (%d lines)\n", i+1, len(chunks), len(chunk))
		} else {
			fmt.Fprintf(os.Stderr, "pair-changelog: distilling %d lines\n", len(chunk))
		}
		nl, err := distillStep(newLog, strings.Join(chunk, "\n"), agent, modelName, today)
		if err != nil {
			fail("model: %v", err)
		}
		newLog = nl
	}

	if strings.TrimSpace(newLog) == "" || newLog == priorLog {
		return // nothing produced / no change
	}

	// Write the log first, then the anchor (crash-safety): a crash between leaves
	// the anchor one-behind → next press re-processes a delta already covered by
	// the frozen-prefix dedup; it never skips content.
	if err := atomicWrite(logPath, newLog); err != nil {
		fail("write log: %v", err)
	}
	if err := writeAnchor(anchorPath, len(boundaries), anchorSnippet(lines, anchorLines)); err != nil {
		fail("write anchor: %v", err)
	}
}

// distillStep runs one model distill — first-run (priorLog empty) or incremental
// (revise the last entry + append) — and returns the new full log. Returns
// priorLog unchanged when the model produces nothing; errors on a non-distill
// response (a hijacked continuation). Used per-chunk by the first-run batcher.
func distillStep(priorLog, sliceText, agent, modelName, today string) (string, error) {
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
	return assemble(frozen, ekPrime, newEntries, today, lastHeaderDate(priorLog)), nil
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
