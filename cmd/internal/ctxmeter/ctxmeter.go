// Package ctxmeter reads an agent's current context-window occupancy (in
// tokens) from its transcript, and humanizes a token count for display.
package ctxmeter

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"strconv"
)

// ContextTokens streams a transcript and returns the current-context token
// occupancy from the LAST qualifying record, plus whether one was found.
// Tolerant: unparseable lines are skipped.
func ContextTokens(agent string, r io.Reader) (int, bool) {
	br := bufio.NewReader(r)
	last, found := 0, false
	for {
		line, err := br.ReadBytes('\n') // ReadBytes, not Scanner — records can be MB-sized
		if len(line) > 0 {
			if n, ok := lineTokens(agent, line); ok {
				last, found = n, true
			}
		}
		if err != nil {
			break // io.EOF or read error → stop
		}
	}
	return last, found
}

func lineTokens(agent string, line []byte) (int, bool) {
	switch agent {
	case "codex":
		var r struct {
			Type    string `json:"type"`
			Payload struct {
				Type string `json:"type"`
				Info struct {
					Last struct {
						InputTokens int `json:"input_tokens"`
					} `json:"last_token_usage"`
				} `json:"info"`
			} `json:"payload"`
		}
		if json.Unmarshal(line, &r) != nil || r.Type != "event_msg" || r.Payload.Type != "token_count" {
			return 0, false
		}
		return r.Payload.Info.Last.InputTokens, true
	case "agy":
		return 0, false // no usable token source
	default: // claude
		var r struct {
			Type        string `json:"type"`
			IsSidechain bool   `json:"isSidechain"`
			Message     struct {
				Model string `json:"model"`
				Usage struct {
					Input       int `json:"input_tokens"`
					CacheCreate int `json:"cache_creation_input_tokens"`
					CacheRead   int `json:"cache_read_input_tokens"`
				} `json:"usage"`
			} `json:"message"`
		}
		if json.Unmarshal(line, &r) != nil || r.Type != "assistant" || r.IsSidechain || r.Message.Model == "<synthetic>" {
			return 0, false
		}
		return r.Message.Usage.Input + r.Message.Usage.CacheCreate + r.Message.Usage.CacheRead, true
	}
}

// Humanize formats a token count per the spec's pinned rule:
// <1000 exact; 1000≤n<1_000_000 → Nk (round half-up); ≥1_000_000 → N.NM (floor).
func Humanize(n int) string {
	switch {
	case n < 1000:
		return strconv.Itoa(n)
	case n < 1_000_000:
		return strconv.Itoa(int(math.Round(float64(n)/1000))) + "k"
	default:
		return fmt.Sprintf("%.1fM", math.Floor(float64(n)/100_000)/10)
	}
}
