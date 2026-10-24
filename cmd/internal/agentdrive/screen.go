package agentdrive

import (
	"github.com/xianxu/pair/cmd/internal/tailmarkup"
	"regexp"
	"strings"
	"unicode/utf8"
)

type Screen struct {
	Epoch uint64
	Lines []string
	// History is retained tail evidence, never an actionable picker viewport.
	// Nil falls back to Lines for standalone pure callers and old fixtures.
	History                    []string
	Idle, TurnActive, Complete bool
	Recoverable                bool
	// Dismissable names a recognized panel with no composer; only DismissOnly
	// may use it.
	Dismissable bool
	// Interruptible names a working turn the wrapper may stop; only an
	// interrupt request may use it (pair#717).
	Interruptible bool
	ComposerText  string
	// CommandText is the complete recognized composer region, independent of model cleanup.
	CommandText  string
	CommandKnown bool
}

func (s Screen) Admission() Admission {
	return Admission{Idle: s.Idle, Recoverable: s.Recoverable, Dismissable: s.Dismissable, Interruptible: s.Interruptible}
}

func screenHistory(screen Screen) []string {
	if screen.History != nil {
		return screen.History
	}
	return screen.Lines
}

type ScreenPhase string

const (
	ScreenUnknown  ScreenPhase = ""
	ScreenModel    ScreenPhase = "model"
	ScreenEffort   ScreenPhase = "effort"
	ScreenAdvanced ScreenPhase = "advanced"
	ScreenStatus   ScreenPhase = "status"
	ScreenCombined ScreenPhase = "combined"
	ScreenCommand  ScreenPhase = "command"
)

type ScreenEvidence struct {
	Phase                   ScreenPhase
	Labels                  []string
	Selected, Model, Effort string
	Scope                   ModelScope
	Confirmation            string
	ConfirmationCount       int
	Footer                  string
	ComposerText            string
}

var codexRow = regexp.MustCompile(`^[›]?\s*\d+\.\s+(GPT-[A-Za-z0-9.-]+)(?:\s|$)`)
var effortRow = regexp.MustCompile(`^[›]?\s*\d+\.\s+(Extra high|Low|Medium|High|More reasoning…)(?:\s|$)`)
var advancedRow = regexp.MustCompile(`^[›]?\s*\d+\.\s+(Max|Ultra)(?:\s|$)`)
var claudeRow = regexp.MustCompile(`^[❯↓]?\s*\d+\.\s+(Default|(?:Opus|Sonnet|Haiku|Fable) [0-9.]+)(?:\s|$)`)
var codexStatus = regexp.MustCompile(`^• Model changed to (gpt-[a-z0-9.-]+) (low|medium|high|xhigh|max|ultra)( for this session only)?$`)
var claudeStatus = regexp.MustCompile(`^⎿\s+Set model to ((?:Opus|Sonnet|Haiku|Fable) [0-9.]+) for this session only$`)

func confirmationCounts(lines []string) map[string]int {
	counts := map[string]int{}
	if len(lines) > 200 {
		lines = lines[len(lines)-200:]
	}
	for _, line := range lines {
		line = tailmarkup.Strip(line)
		if codexStatus.MatchString(line) || claudeStatus.MatchString(line) {
			counts[line]++
		}
	}
	return counts
}

// checkedMarkup rejects incomplete/unknown styling. Native Codex may paint its
// explicit selection prefix without reverse video; balanced markup is still
// required before any row can supply evidence.
func checkedMarkup(s string) (bool, bool) {
	stack := []string{}
	rev := false
	for len(s) > 0 {
		i := strings.Index(s, "‹")
		if i < 0 {
			break
		}
		s = s[i:]
		if strings.HasPrefix(s, "‹‹") {
			s = s[len("‹‹"):]
			continue
		}
		end := strings.Index(s, "›")
		if end < 0 {
			return false, false
		}
		tag := s[len("‹"):end]
		s = s[end+len("›"):]
		switch tag {
		case "cursor":
		case "rev", "dim":
			stack = append(stack, tag)
			rev = rev || tag == "rev"
		case "/rev", "/dim":
			if len(stack) == 0 || stack[len(stack)-1] != tag[1:] {
				return false, false
			}
			stack = stack[:len(stack)-1]
		default:
			return false, false
		}
	}
	return rev, len(stack) == 0
}
func canonicalEffort(s string) string {
	switch strings.ToLower(s) {
	case "extra high":
		return "xhigh"
	case "more reasoning…":
		return "more"
	default:
		return strings.ToLower(s)
	}
}

// ReadModelScreen recognizes only complete bounded captured recipe shapes.
func ReadModelScreen(agent string, screen Screen) ScreenEvidence {
	unknown := ScreenEvidence{}
	if !screen.Complete || screen.TurnActive || len(screen.Lines) > 200 {
		return unknown
	}
	plain := make([]string, len(screen.Lines))
	for i, line := range screen.Lines {
		if !utf8.ValidString(line) || len(line) > MaxResultTailBytes {
			return unknown
		}
		_, ok := checkedMarkup(line)
		if !ok {
			return unknown
		}
		plain[i] = tailmarkup.Strip(line)
	}
	heading := -1
	if (agent == "codex" || agent == "claude") && screen.Recoverable && screen.ComposerText == "/model" {
		return ScreenEvidence{Phase: ScreenCommand, ComposerText: screen.ComposerText}
	}
	phase := ScreenUnknown
	model := ""
	for i, line := range plain {
		p := ScreenUnknown
		m := ""
		if agent == "codex" {
			if line == "Advanced Reasoning" {
				p = ScreenAdvanced
			}
			if line == "Select Model and Effort" {
				p = ScreenModel
			}
			if strings.HasPrefix(line, "Select Reasoning Level for ") {
				p = ScreenEffort
				m = strings.ToLower(strings.TrimPrefix(line, "Select Reasoning Level for "))
			}
		}
		if agent == "claude" && line == "Select model" {
			p = ScreenCombined
		}
		if p != ScreenUnknown {
			if heading >= 0 {
				return unknown
			}
			heading = i
			phase = p
			model = m
		}
	}
	if heading >= 0 {
		e := ScreenEvidence{Phase: phase, Model: model}
		seen := map[string]bool{}
		selected := 0
		footer := false
		for i := heading + 1; i < len(plain); i++ {
			line := plain[i]
			label := ""
			sel := false
			switch phase {
			case ScreenModel:
				if m := codexRow.FindStringSubmatch(line); m != nil {
					label = strings.ToLower(m[1])
					sel = strings.HasPrefix(line, "›")
				}
				if line == "enter select · esc back" {
					if e.Footer != "" {
						return unknown
					}
					e.Footer = line
					footer = true
				}
			case ScreenEffort:
				if m := effortRow.FindStringSubmatch(line); m != nil {
					label = canonicalEffort(m[1])
					sel = strings.HasPrefix(line, "›")
				}
				if line == "enter default · s session · esc back" || line == "enter select · esc back" {
					if e.Footer != "" {
						return unknown
					}
					e.Footer = line
					footer = true
				}
			case ScreenAdvanced:
				if m := advancedRow.FindStringSubmatch(line); m != nil {
					label = canonicalEffort(m[1])
					sel = strings.HasPrefix(line, "›")
				}
				if line == "enter default · s session · esc back" {
					if e.Footer != "" {
						return unknown
					}
					e.Footer = line
					footer = true
				}
			case ScreenCombined:
				if m := claudeRow.FindStringSubmatch(line); m != nil {
					label = strings.ToLower(m[1])
					sel = strings.HasPrefix(line, "❯")
				}
				if strings.Contains(line, " effort") && strings.HasSuffix(line, "←/→ to adjust") {
					fields := strings.Fields(line)
					if len(fields) > 2 {
						e.Effort = canonicalEffort(fields[1])
					}
				}
				if line == "Enter to set as default · s to use this session only · Esc to cancel" {
					if e.Footer != "" {
						return unknown
					}
					e.Footer = line
					footer = true
				}
			}
			if label == "" {
				continue
			}
			if len(label) > MaxModelLabelBytes || seen[label] || len(e.Labels) >= MaxObservedLabels {
				return unknown
			}
			seen[label] = true
			e.Labels = append(e.Labels, label)
			if sel {
				selected++
				e.Selected = label
			}
		}
		if !footer || selected != 1 {
			return unknown
		}
		if phase == ScreenEffort && (!strings.HasPrefix(model, "gpt-") || len(model) > MaxModelLabelBytes) {
			return unknown
		}
		if phase == ScreenEffort {
			expected := "enter default · s session · esc back"
			if e.Selected == "more" {
				expected = "enter select · esc back"
			}
			if e.Footer != expected {
				return unknown
			}
		}
		if phase == ScreenAdvanced && (len(e.Labels) != 2 || !seen["max"] || !seen["ultra"]) {
			return unknown
		}
		if phase == ScreenCombined && e.Effort == "" {
			return unknown
		}
		return e
	}
	if agent == "codex" && screen.Idle {
		var e ScreenEvidence
		count := 0
		for _, line := range plain {
			if m := codexStatus.FindStringSubmatch(line); m != nil {
				count++
				e = ScreenEvidence{Phase: ScreenStatus, Model: m[1], Effort: m[2], Scope: ScopeDefault}
				if m[3] != "" {
					e.Scope = ScopeSession
				}
				e.Confirmation = line
			}
		}
		if count == 0 {
			return unknown
		}
		footer := false
		composer := false
		for _, line := range plain {
			if strings.HasPrefix(strings.ToLower(line), e.Model+" "+e.Effort+" · ") {
				footer = true
			}
			if line == "› Ask Codex to do anything" || line == "›" {
				composer = true
			}
		}
		if footer && composer {
			e.ConfirmationCount = confirmationCounts(screenHistory(screen))[e.Confirmation]
			return e
		}
	}
	if agent == "claude" && screen.Idle {
		var e ScreenEvidence
		lastCommand, lastConfirmation := -1, -1
		for i, line := range plain {
			if line == "❯ /model" {
				lastCommand = i
			}
			if m := claudeStatus.FindStringSubmatch(line); m != nil {
				lastConfirmation = i
				e = ScreenEvidence{Phase: ScreenStatus, Model: strings.ToLower(m[1]), Scope: ScopeSession, Confirmation: line}
			}
		}
		if lastConfirmation <= lastCommand || lastCommand < 0 {
			return unknown
		}
		composer := false
		for _, line := range plain[lastConfirmation+1:] {
			if strings.TrimSpace(line) == "❯" {
				composer = true
			}
			if idx := strings.Index(line, " · /effort"); idx >= 0 {
				fields := strings.Fields(line[:idx])
				if len(fields) > 0 {
					e.Effort = canonicalEffort(fields[len(fields)-1])
				}
			}
		}
		if composer && e.Effort != "" {
			e.ConfirmationCount = confirmationCounts(screenHistory(screen))[e.Confirmation]
			return e
		}
	}
	return unknown
}
