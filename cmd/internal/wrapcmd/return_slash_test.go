package wrapcmd

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	uv "github.com/charmbracelet/ultraviolet"

	"github.com/xianxu/pair/cmd/internal/adapt"
)

func slashComposerPaint(agent string, lines ...string) string {
	const rule = "\x1b[2m────────────────────────────────────────────────────────────\x1b[0m"
	row, column := 7, 3
	var paint string
	switch agent {
	case "claude":
		paint = claudeBox(5, "❯", "136;136;136", lines...)
	case "qoder":
		paint, column = qoderBox(6, ">", lines...), 4
		// qoderBox models its placeholder, including a painted cursor and
		// two placeholder spaces. Native typing replaces that decoration.
		for i, line := range lines {
			paint += fmt.Sprintf("\x1b[%d;4H\x1b[K%s", 7+i, line)
		}
	case "codex":
		row = 20
		paint = "\x1b[20;1H\x1b[1m›\x1b[22m " + lines[0]
		for i, line := range lines[1:] {
			paint += fmt.Sprintf("\x1b[%d;3H%s", 21+i, line)
		}
		paint += fmt.Sprintf("\x1b[%d;3HGPT-6.1-Sol default · /fixture", 21+len(lines))
	case "muse", "agy":
		prompt := "⟩ "
		if agent == "agy" {
			prompt = "\x1b[94m>\x1b[39m "
		}
		paint = "\x1b[6;1H" + rule + "\x1b[7;1H" + prompt + lines[0]
		for i, line := range lines[1:] {
			paint += fmt.Sprintf("\x1b[%d;3H%s", 8+i, line)
		}
		paint += fmt.Sprintf("\x1b[%d;1H%s", 7+len(lines), rule)
	case "grok":
		paint = grokMinimal(6, "❯", lines[0], lines[1:]...)
	}
	return paint + fmt.Sprintf("\x1b[?25h\x1b[%d;%dH", row, column+len(lines[0]))
}

func TestDecidePlainReturnSlashComposer(t *testing.T) {
	for agent := range harnessTTYProfiles {
		t.Run(agent, func(t *testing.T) {
			profile, _ := profileForHarness(agent, true)
			for _, tc := range []struct {
				text  string
				slash bool
			}{
				{"/login", true}, {"/model opus", true}, {"/clear", true},
				{"/Users/xianxu/workspace/brain is the repo", false}, {" /model", false},
				{"/ prose", false}, {"hello /model", false}, {"!ls", false}, {"@file", false},
			} {
				t.Run(tc.text, func(t *testing.T) {
					s := peerSnapshot(t, slashComposerPaint(agent, tc.text))
					if !profile.recognize(s).Active {
						t.Fatal("fixture must preserve active composer")
					}
					got := decidePlainReturn(profile, false, &s)
					want := profile.keymap.plainCR
					outcome := adapt.Fired
					if tc.slash {
						want, outcome = profile.keymap.altCR, adapt.Bypass
					}
					if !bytes.Equal(got.bytes, want) || got.submits != tc.slash || got.outcome != outcome {
						t.Fatalf("decision bytes=%q submits=%t outcome=%s; want %q submits=%t outcome=%s", got.bytes, got.submits, got.outcome, want, tc.slash, outcome)
					}
					s.Complete = false
					if partial := decidePlainReturn(profile, false, &s); !bytes.Equal(partial.bytes, profile.keymap.plainCR) || partial.submits {
						t.Fatalf("partial frame submitted: %+v", partial)
					}
				})
			}
			for _, lines := range [][]string{{"/model", "more"}, {"/model", ""}, {"", "/model"}} {
				s := peerSnapshot(t, slashComposerPaint(agent, lines...))
				// Keep an explicit empty continuation at the cursor as user input.
				if lines[len(lines)-1] == "" {
					s.Cursor.Y++
				}
				if got := decidePlainReturn(profile, false, &s); !bytes.Equal(got.bytes, profile.keymap.plainCR) || got.submits {
					t.Errorf("multiline %q submitted: %+v", strings.Join(lines, "\n"), got)
				}
			}
		})
	}
}

func TestSlashReturnRequiresCompleteTextProof(t *testing.T) {
	for _, tc := range []struct{ name, agent, paint string }{
		{"unknown Claude mode", "claude", claudeBox(5, "?", "136;136;136", "/model") + "\x1b[?25h\x1b[7;9H"},
		{"Claude shell mode", "claude", claudeBox(5, "!", "253;93;177", "/model") + "\x1b[?25h\x1b[7;9H"},
		{"Claude faint suggestion", "claude", claudeBox(5, "❯", "136;136;136", "\x1b[2m/model\x1b[22m") + "\x1b[?25h\x1b[7;3H"},
		{"Codex faint suggestion", "codex", "\x1b[20;1H\x1b[1m›\x1b[22m \x1b[2m/model\x1b[22m\x1b[?25h\x1b[20;3H"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := peerSnapshot(t, tc.paint)
			p, _ := profileForHarness(tc.agent, true)
			if !p.recognize(s).Active {
				t.Fatal("text refusal must not remove activity recognition")
			}
			got := decidePlainReturn(p, false, &s)
			if got.submits || !bytes.Equal(got.bytes, p.keymap.plainCR) {
				t.Fatalf("unproved text submitted: %+v", got)
			}
		})
	}
	for _, agent := range []string{"claude", "codex", "qoder", "muse", "agy", "grok"} {
		s := peerSnapshot(t, slashComposerPaint(agent, "/model"))
		s.Cursor.X = peerInputColumn(agent)
		p, _ := profileForHarness(agent, true)
		if got := decidePlainReturn(p, false, &s); !got.submits {
			t.Errorf("%s cursor before real text lost complete command", agent)
		}
	}
}

func TestSlashReturnTranslatesNativeKeys(t *testing.T) {
	for agent := range harnessTTYProfiles {
		t.Run(agent, func(t *testing.T) {
			p := &proxy{agentBasename: agent}
			if err := p.configureHarnessTTY(true, 120, 38); err != nil {
				t.Fatal(err)
			}
			defer p.closeTerminal()
			var rolling []byte
			p.handleChunk([]byte("\x1b[?2004h"+slashComposerPaint(agent, "/model")), &rolling)
			for _, key := range []string{"\r", "\x1b[13u", "\x1b[13;1u"} {
				out, left, paste := p.translateChunk([]byte(key), false)
				if !bytes.Equal(out, p.ttyProfile.keymap.altCR) || len(left) != 0 || paste {
					t.Errorf("%q => %q carry=%q paste=%t", key, out, left, paste)
				}
			}
			out, _, _ := p.translateChunk([]byte("\x1b[200~\r\x1b[201~"), false)
			if string(out) != "\x1b[200~\r\x1b[201~" {
				t.Errorf("paste remapped: %q", out)
			}
			out, _, _ = p.translateChunk([]byte("\x1b\r"), false)
			if !bytes.Equal(out, p.ttyProfile.keymap.altCR) {
				t.Errorf("Alt+Return changed: %q", out)
			}
		})
	}
}

func TestComposerProjectionWorkIsOnePass(t *testing.T) {
	for _, size := range [][2]int{{64, 32}, {128, 64}} {
		width, height := size[0], size[1]
		s := peerSnapshot(t, slashComposerPaint("claude", "/model"))
		s.Width, s.Height = width, height
		s.Cells = make([]uv.Cell, width*height)
		for i := range s.Cells {
			s.Cells[i] = uv.Cell{Content: "a", Width: 1}
		}
		s.Cursor.X, s.Cursor.Y = 10, 0
		r := composerRegion{top: 0, bottom: height, left: 2, agent: "claude", valid: true}
		calls := 0
		_, ok := projectComposerCells(s, r, func(x, y int) *uv.Cell { calls++; return s.CellAt(x, y) })
		if !ok || calls != (width-2)*height {
			t.Fatalf("%dx%d read %d cells; want %d", width, height, calls, (width-2)*height)
		}
	}
}

func TestReturnLocatesComposerOncePerDecision(t *testing.T) {
	s := peerSnapshot(t, slashComposerPaint("claude", "/model"))
	for _, n := range []int{32, 64} {
		p, _ := profileForHarness("claude", true)
		locate := p.recognize
		calls := 0
		p.recognize = func(s terminalSnapshot) composerRecognition { calls++; return locate(s) }
		for range n {
			if !decidePlainReturn(p, false, &s).submits {
				t.Fatal("command did not submit")
			}
		}
		if calls != n {
			t.Fatalf("%d decisions located %d times", n, calls)
		}
	}
}

// Styled suggestions remain visible without becoming authored command proof.
func TestSlashReturnDeclinesMixedFaintSuggestion(t *testing.T) {
	for agent := range harnessTTYProfiles {
		t.Run(agent, func(t *testing.T) {
			profile, _ := profileForHarness(agent, true)
			for _, tc := range []struct {
				name, text string
				authored   int
			}{
				{"command completion", "/model", 1},
				{"argument completion", "/model opus", 7},
				{"argument suffix", "/model opus", 9},
				{"separator only", "/model opus", 6},
			} {
				t.Run(tc.name, func(t *testing.T) {
					s := peerSnapshot(t, slashComposerPaint(agent, tc.text))
					r := profile.recognize(s)
					if !r.Active || !r.region.valid {
						t.Fatal("fixture lacks active located composer")
					}
					for x := r.region.left + tc.authored; x < r.region.left+len(tc.text); x++ {
						s.CellAt(x, r.region.top).Style.Attrs |= uv.AttrFaint
						if tc.name == "separator only" {
							break
						}
					}
					s.Cursor.X = r.region.left + tc.authored
					s.Cursor.Y = r.region.top
					if !profile.recognize(s).Active {
						t.Fatal("suggestion removed activity")
					}
					if text, known := projectComposerRegion(s, r.region); known {
						t.Errorf("suggested text supplied authored proof: %q", text)
					}
					got := decidePlainReturn(profile, false, &s)
					if got.submits || !bytes.Equal(got.bytes, profile.keymap.plainCR) {
						t.Fatalf("unaccepted completion submitted: %+v", got)
					}
					// Accepting completion clears its faint provenance, not merely
					// moving the cursor; normal full text remains command authority.
					for x := r.region.left; x < r.region.left+len(tc.text); x++ {
						s.CellAt(x, r.region.top).Style.Attrs &^= uv.AttrFaint
					}
					if !decidePlainReturn(profile, false, &s).submits {
						t.Fatal("authored command refused")
					}
				})
			}
		})
	}
}

func TestSlashReturnPreservesBlankContinuationAfterCursorMove(t *testing.T) {
	for agent := range harnessTTYProfiles {
		for _, blank := range []string{"", "   "} {
			for _, cursorLast := range []bool{false, true} {
				t.Run(fmt.Sprintf("%s/spaces=%d/cursorLast=%t", agent, len(blank), cursorLast), func(t *testing.T) {
					paint := slashComposerPaint(agent, "/model", blank)
					if agent == "codex" {
						// Two input rows, one separator, then the native footer.
						paint += "\x1b[23;3HGPT-6.1-Sol default · /fixture\x1b[20;9H"
					}
					s := peerSnapshot(t, paint)
					profile, _ := profileForHarness(agent, true)
					r := profile.recognize(s)
					if !r.Active || !r.region.valid {
						t.Fatal("fixture lacks composer")
					}
					s.Cursor.Y, s.Cursor.X = r.region.top, r.region.left+6
					if cursorLast {
						s.Cursor.Y++
						s.Cursor.X = r.region.left + len(blank)
					}
					if got := decidePlainReturn(profile, false, &s); got.submits || !bytes.Equal(got.bytes, profile.keymap.plainCR) {
						t.Fatalf("multiline continuation erased after cursor move: %+v", got)
					}
				})
			}
		}
	}
}

func TestSlashReturnDeclinesFooterlessCodexAmbiguity(t *testing.T) {
	profile, _ := profileForHarness("codex", true)
	for _, text := range []string{"/model", "/model opus  "} {
		paint := "\x1b[20;1H\x1b[1m›\x1b[22m " + text + "\x1b[?25h\x1b[20;3H"
		s := peerSnapshot(t, paint)
		if !profile.recognize(s).Active {
			t.Fatal("activity lost")
		}
		if got := decidePlainReturn(profile, false, &s); got.submits || !bytes.Equal(got.bytes, profile.keymap.plainCR) {
			t.Fatalf("unbounded Codex slash granted authority: %+v", got)
		}
		if text, known := qualifiedComposerText("codex", s); known {
			t.Fatalf("unbounded slash proof %q", text)
		}
	}
}
