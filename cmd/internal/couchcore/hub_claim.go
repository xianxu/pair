package couchcore

import (
	"bytes"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

const MaxHubClaimBytes = 64 << 10
const MaxHubClaimLines = 1024

// HubClaim is positive configuration owned by a hub. Slot -1 claims a repo.
type HubClaim struct {
	Repository string
	Slot       int
}
type hubClaimLine struct {
	raw   string
	claim *HubClaim
}
type HubClaimDocument struct {
	raw   []byte
	lines []hubClaimLine
}

func (d HubClaimDocument) Bytes() []byte { return bytes.Clone(d.raw) }
func (d HubClaimDocument) Claims() []HubClaim {
	var out []HubClaim
	for _, line := range d.lines {
		if line.claim != nil {
			out = append(out, *line.claim)
		}
	}
	return out
}
func ParseHubClaims(raw []byte) (HubClaimDocument, error) {
	if len(raw) > MaxHubClaimBytes || !utf8.Valid(raw) {
		return HubClaimDocument{}, fmt.Errorf(".hub exceeds 64 KiB or is not UTF-8; shorten or repair the file and retry")
	}
	d := HubClaimDocument{raw: bytes.Clone(raw)}
	if len(raw) == 0 {
		return d, nil
	}
	lines := strings.SplitAfter(string(raw), "\n")
	if lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	if len(lines) > MaxHubClaimLines {
		return HubClaimDocument{}, fmt.Errorf(".hub exceeds %d lines; shorten the file and retry", MaxHubClaimLines)
	}
	for i, line := range lines {
		entry := hubClaimLine{raw: line}
		text := strings.TrimSpace(line)
		if text != "" && !strings.HasPrefix(text, "#") {
			claim, err := parseHubClaim(text)
			if err != nil {
				return HubClaimDocument{}, fmt.Errorf(".hub line %d: %w", i+1, err)
			}
			entry.claim = &claim
		}
		d.lines = append(d.lines, entry)
	}
	return d, nil
}
func parseHubClaim(text string) (HubClaim, error) {
	c := HubClaim{Repository: text, Slot: -1}
	if at := strings.LastIndexByte(text, ':'); at >= 0 {
		c.Repository = text[:at]
		n, e := strconv.Atoi(text[at+1:])
		if e != nil || n < 0 || strconv.Itoa(n) != text[at+1:] {
			return c, fmt.Errorf("invalid exact slot %q", text)
		}
		c.Slot = n
	}
	p := c.Repository
	if p == "" || filepath.IsAbs(p) || filepath.Clean(p) != p || p == "." || p == ".." || strings.HasPrefix(p, "../") || strings.ContainsAny(p, "\\:!#") || strings.IndexFunc(p, func(r rune) bool { return unicode.IsSpace(r) || unicode.IsControl(r) }) >= 0 {
		return c, fmt.Errorf("invalid workspace-relative repository %q", p)
	}
	return c, nil
}
func claimSlot(h HubDefinition, c HubClaim) HubSlot {
	return HubSlot{Repository: filepath.Join(h.Workspace, c.Repository), Number: c.Slot}
}

type HubClaimEdit struct {
	Root          string
	Before, After []byte
}

// PlanHubClaimEdits reasons only about durable claims, never runtime overrides.
// Target additions are returned first; each adapter write is independently atomic.
func PlanHubClaimEdits(hubs []HubDefinition, slots []HubSlot, target string) ([]HubClaimEdit, error) {
	state, err := NewHubOwnership(hubs)
	if err != nil {
		return nil, err
	}
	if !state.IsHub(target) {
		return nil, fmt.Errorf("unknown target hub %q", target)
	}
	selected := map[HubSlot]bool{}
	for _, slot := range slots {
		if err := slot.Validate(); err != nil {
			return nil, err
		}
		if state.hubForSlot(slot) != "" {
			return nil, fmt.Errorf("hub slots cannot be moved")
		}
		selected[slot] = true
	}
	if len(selected) == 0 || len(selected) > MaxHubSlots {
		return nil, fmt.Errorf("move requires 1..%d slots", MaxHubSlots)
	}
	changed := state.Hubs()
	// Remove exact claims for selected slots, preserving every other byte.
	for i, h := range changed {
		var b strings.Builder
		for _, line := range h.Claims.lines {
			if line.claim != nil && line.claim.Slot >= 0 && selected[claimSlot(h, *line.claim)] {
				continue
			}
			b.WriteString(line.raw)
		}
		changed[i].Claims, err = ParseHubClaims([]byte(b.String()))
		if err != nil {
			return nil, err
		}
	}
	base, _ := NewHubOwnership(changed)
	targetIndex := -1
	for i, h := range changed {
		if h.Root == target {
			targetIndex = i
		}
	}
	// Retain an already necessary target claim in its original position.
	var preserved strings.Builder
	seen := map[HubSlot]bool{}
	for _, line := range hubs[targetIndex].Claims.lines {
		if line.claim != nil && line.claim.Slot >= 0 {
			slot := claimSlot(hubs[targetIndex], *line.claim)
			if selected[slot] {
				if base.Resolve(slot).Hub == target || seen[slot] {
					continue
				}
				seen[slot] = true
			}
		}
		preserved.WriteString(line.raw)
	}
	raw := []byte(preserved.String())
	for _, slot := range slots {
		if seen[slot] {
			continue
		}
		seen[slot] = true
		if base.Resolve(slot).Hub == target {
			continue
		}
		relative, e := filepath.Rel(changed[targetIndex].Workspace, slot.Repository)
		if e != nil {
			return nil, e
		}
		claim, e := parseHubClaim(relative + ":" + strconv.Itoa(slot.Number))
		if e != nil {
			return nil, e
		}
		if len(raw) > 0 && raw[len(raw)-1] != '\n' {
			raw = append(raw, '\n')
		}
		raw = append(raw, []byte(claim.Repository+":"+strconv.Itoa(claim.Slot)+"\n")...)
	}
	changed[targetIndex].Claims, err = ParseHubClaims(raw)
	if err != nil {
		return nil, err
	}
	var out []HubClaimEdit
	order := []int{targetIndex}
	for i := range hubs {
		if i != targetIndex {
			order = append(order, i)
		}
	}
	for _, i := range order {
		before, after := hubs[i].Claims.Bytes(), changed[i].Claims.Bytes()
		if !bytes.Equal(before, after) {
			out = append(out, HubClaimEdit{Root: hubs[i].Root, Before: before, After: after})
		}
	}
	return out, nil
}
