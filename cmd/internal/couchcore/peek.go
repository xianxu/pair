package couchcore

import "context"

// DefaultPeekLines is how much of a slot's recent terminal a peek shows when
// the caller does not say: about one screen, which holds the composer.
const DefaultPeekLines = 40

// PeekResult is a read-only look at another slot (pair#362): its recent
// terminal as plain text, and where its prompt log and the agent's own
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
	SentPrompts string   `json:"sent_prompts,omitempty"`
	Transcripts []string `json:"transcripts,omitempty"`
	Unavailable []string `json:"unavailable,omitempty"`
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
	switch {
	case result.Agent == "":
		result.Unavailable = append(result.Unavailable, "terminal recording: the thread's agent is unknown")
	case c.SlotTerminal == nil:
		result.Unavailable = append(result.Unavailable, "terminal recording: no reader is configured")
	default:
		n := lines
		if n <= 0 {
			n = DefaultPeekLines
		}
		rendered, err := c.SlotTerminal(address, result.Agent, n)
		if err != nil {
			result.Unavailable = append(result.Unavailable, "terminal recording: "+err.Error())
			break
		}
		result.Lines = peekTail(rendered, n)
	}
	return result, nil
}
