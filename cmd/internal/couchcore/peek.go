package couchcore

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"
)

// DefaultPeekLines is how much of a slot's recent terminal a peek shows when
// the caller does not say: about one screen, which holds the composer.
const DefaultPeekLines = 40

// PeekResult is a read-only look at another slot (pair#362): its recent
// terminal -- live from the wrapper's memory with light markup when it
// answers (pair#425), else the recording as plain text -- and where its prompt log and the agent's own
// transcript live. A coordinator reads it to tell whether a peer message
// reached the composer and whether the turn was submitted, without asking the
// recipient. Transcripts are paths, never parsed: the reading agent knows its
// own format, and Couch stays agent-agnostic. Every evidence source that could
// not be read is named in Unavailable with its reason; a failed read is never
// an empty answer.
type PeekResult struct {
	Ref         string   `json:"ref"`
	Tag         string   `json:"tag"`
	Agent       string   `json:"agent,omitempty"`
	WorkingPath string   `json:"working_path,omitempty"`
	Lines       []string `json:"lines"`
	// Source says where Lines came from: "live" (the wrapper's in-memory
	// terminal, with markup and Cursor, pair#425) or "recording" (the plain
	// render of the terminal recording, the fallback).
	Source string `json:"source,omitempty"`
	// Cursor summarizes the live cursor ("7,3 bar", "hidden at 7,3").
	Cursor string `json:"cursor,omitempty"`
	// Truncated counts the oldest live lines dropped to bound the answer.
	Truncated   int      `json:"truncated,omitempty"`
	SentPrompts string   `json:"sent_prompts,omitempty"`
	Transcripts []string `json:"transcripts,omitempty"`
	Unavailable []string `json:"unavailable,omitempty"`
}

// TerminalTail is a slot's live tail (pair#425): its wrapper's in-memory
// terminal with faint, reverse and cursor marked. Pair renders it and never
// classifies it; whether the slot is busy is the reader's call.
type TerminalTail struct {
	Lines     []string
	Cursor    string
	Truncated int
}

// SlotTailReader reads a thread's live tail from its running wrapper.
type SlotTailReader func(ctx context.Context, address ThreadAddress, maxLines int) (TerminalTail, error)

// PeekSnapshot is a multi-slot peek: one section per requested slot, in
// request order. A slot that could not be read says why in its section.
type PeekSnapshot struct {
	Slots []PeekResult `json:"slots"`
}

// MaxPeekSlots bounds one multi-slot peek, and with it the reads it runs at once.
const MaxPeekSlots = 32

// ExpandPeekReferences splits a peek's reference into slots: "," separates
// groups, and "repo:1:2:3" is repo:1, repo:2, repo:3. Anything else is one
// reference, passed through for the thread resolver.
func ExpandPeekReferences(raw string) ([]string, error) {
	var refs []string
	for _, group := range strings.Split(raw, ",") {
		group = strings.TrimSpace(group)
		if group == "" {
			return nil, fmt.Errorf("peek reference %q has an empty slot", raw)
		}
		parts := strings.Split(group, ":")
		numbers := len(parts) > 2
		for _, part := range parts[1:] {
			if _, err := strconv.Atoi(part); err != nil {
				numbers = false
			}
		}
		if !numbers {
			refs = append(refs, group)
			continue
		}
		for _, number := range parts[1:] {
			refs = append(refs, parts[0]+":"+number)
		}
	}
	if len(refs) > MaxPeekSlots {
		return nil, fmt.Errorf("peek reference %q names %d slots; at most %d", raw, len(refs), MaxPeekSlots)
	}
	return refs, nil
}

// PeekSlots peeks every reference concurrently. A reference that does not
// resolve, or whose peek fails, becomes a section naming why; the snapshot as
// a whole never fails.
func (c *Couch) PeekSlots(ctx context.Context, resolve func(ref string) (ThreadAddress, error), refs []string, lines int) PeekSnapshot {
	snapshot := PeekSnapshot{Slots: make([]PeekResult, len(refs))}
	var wg sync.WaitGroup
	for i, ref := range refs {
		wg.Add(1)
		go func() {
			defer wg.Done()
			failed := func(err error) {
				snapshot.Slots[i] = PeekResult{Ref: ref, Lines: []string{}, Unavailable: []string{err.Error()}}
			}
			address, err := resolve(ref)
			if err != nil {
				failed(err)
				return
			}
			result, err := c.PeekThread(ctx, ref, address, lines)
			if err != nil {
				failed(err)
				return
			}
			snapshot.Slots[i] = result
		}()
	}
	wg.Wait()
	return snapshot
}

// SlotTerminalReader renders a thread's live terminal recording as plain lines,
// keeping at most the given number of history rows plus the visible screen.
type SlotTerminalReader func(address ThreadAddress, agent string, maxLines int) ([]string, error)

// peekTail is the last n lines (n<=0 means DefaultPeekLines).
func peekTail(lines []string, n int) []string {
	if n <= 0 {
		n = DefaultPeekLines
	}
	if len(lines) > n {
		lines = lines[len(lines)-n:]
	}
	return append([]string{}, lines...)
}

// PeekThread looks at one thread read-only: the transcript resolver (the same
// one the agent switcher uses) names the agent and its logs, and the terminal
// reader renders the recording.
func (c *Couch) PeekThread(ctx context.Context, ref string, address ThreadAddress, lines int) (PeekResult, error) {
	record, err := c.Threads.GetThread(address)
	if err != nil {
		return PeekResult{}, err
	}
	result := PeekResult{Ref: ref, Tag: string(address.Tag), WorkingPath: record.WorkingPath, Lines: []string{}}
	if c.SwitchContext == nil {
		result.Unavailable = append(result.Unavailable, "transcript resolver is not configured")
	} else {
		orientation, err := c.SwitchContext.Resolve(ctx, record)
		if err != nil {
			if ctx.Err() != nil {
				return PeekResult{}, ctx.Err()
			}
			result.Unavailable = append(result.Unavailable, "transcripts: "+err.Error())
		}
		result.Agent = orientation.SourceAgent
		result.SentPrompts = orientation.PairLog
		result.Transcripts = orientation.NativeTranscripts
		result.Unavailable = append(result.Unavailable, orientation.Unavailable...)
	}
	n := lines
	if n <= 0 {
		n = DefaultPeekLines
	}
	if c.SlotTail != nil {
		tail, err := c.SlotTail(ctx, address, n)
		if err == nil {
			result.Lines, result.Source, result.Cursor, result.Truncated = peekTail(tail.Lines, n), "live", tail.Cursor, tail.Truncated
			return result, nil
		}
		if ctx.Err() != nil {
			return PeekResult{}, ctx.Err()
		}
		result.Unavailable = append(result.Unavailable, "live tail: "+err.Error())
	}
	switch {
	case result.Agent == "":
		result.Unavailable = append(result.Unavailable, "terminal recording: the thread's agent is unknown")
	case c.SlotTerminal == nil:
		result.Unavailable = append(result.Unavailable, "terminal recording: no reader is configured")
	default:
		rendered, err := c.SlotTerminal(address, result.Agent, n)
		if err != nil {
			result.Unavailable = append(result.Unavailable, "terminal recording: "+err.Error())
			break
		}
		result.Lines, result.Source = peekTail(rendered, n), "recording"
	}
	return result, nil
}
