package wrapcmd

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"
)

type ttyFixtureMetadata struct {
	Agent      string            `json:"agent"`
	Version    string            `json:"version"`
	CapturedAt string            `json:"captured_at"`
	Command    []string          `json:"command"`
	Files      map[string]string `json:"files"`
}

var ttyFixtureVersionToken = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]*$`)

func ttyFixtureVersionDir(version string) string {
	fields := strings.Fields(strings.TrimSpace(version))
	for i := len(fields) - 1; i >= 0; i-- {
		candidate := strings.Trim(fields[i], "()[]{}.,;")
		if strings.IndexFunc(candidate, func(r rune) bool { return r >= '0' && r <= '9' }) < 0 {
			continue
		}
		if ttyFixtureVersionToken.MatchString(candidate) {
			return candidate
		}
	}
	return ""
}

func TestHarnessTTYFixtureVersionDir(t *testing.T) {
	tests := map[string]string{
		"Muse Code 0.1.0 (0.1.0-R708.1)": "0.1.0-R708.1",
		"codex-cli 0.42.0":               "0.42.0",
		"agy version 1.2.3-beta+unsafe":  "",
		"no-version":                     "",
	}
	for version, want := range tests {
		if got := ttyFixtureVersionDir(version); got != want {
			t.Errorf("ttyFixtureVersionDir(%q) = %q, want %q", version, got, want)
		}
	}
}

func TestHarnessTTYFixtureConformance(t *testing.T) {
	root := filepath.Join("testdata", "tty")
	required := map[string]bool{}
	// negatives tracks which positively gated harnesses have a captured state
	// where the gate must decline. A harness without one ships a positive gate
	// backed by no evidence that it refuses anything.
	negatives := map[string]bool{}
	// embedsHomePath tracks which harnesses have ANY capture carrying an
	// absolute home path, so a per-harness acknowledgment expires on the whole
	// set rather than on whichever directory the walk reached last.
	embedsHomePath := map[string]bool{}
	for harness, profile := range harnessTTYProfiles {
		if profile.composerGate == composerGatePositive {
			required[harness] = false
			negatives[harness] = false
		}
	}
	if len(required) == 0 {
		t.Fatal("positive-gated harness fixture inventory is empty")
	}

	metadataPaths, err := filepath.Glob(filepath.Join(root, "*", "*", "metadata.json"))
	if err != nil {
		t.Fatalf("glob fixture metadata: %v", err)
	}
	if len(metadataPaths) == 0 {
		t.Fatal("fixture metadata inventory is empty")
	}
	for _, metadataPath := range metadataPaths {
		metadata, rawFiles, machineBound := readHarnessTTYFixture(t, metadataPath)
		if machineBound {
			embedsHomePath[metadata.Agent] = true
		}
		if _, ok := required[metadata.Agent]; !ok {
			t.Errorf("%s: agent %q has no positive-gated profile", metadataPath, metadata.Agent)
			continue
		}
		required[metadata.Agent] = true
		if filepath.Base(filepath.Dir(metadataPath)) != ttyFixtureVersionDir(metadata.Version) {
			t.Errorf("%s: version directory %q does not match normalized version %q", metadataPath, filepath.Base(filepath.Dir(metadataPath)), ttyFixtureVersionDir(metadata.Version))
		}
		if composer, ok := rawFiles["composer.raw"]; !ok {
			t.Errorf("%s: missing composer.raw", metadataPath)
		} else {
			assertKittyKeyboardPrecondition(t, metadata.Agent, harnessTTYProfiles[metadata.Agent].keymap.plainCR, composer)
		}
		if _, ok := rawFiles["overlay.raw"]; ok {
			negatives[metadata.Agent] = true
		}
		for _, name := range sortedKeys(rawFiles) {
			if name == "working.raw" {
				// Lifecycle replay owns this full Working-to-final transition in
				// codex_working_test.go. The generic Return harness tries every
				// byte split and is both unrelated and quadratic for this capture.
				continue
			}
			wantComposer, ok := ttyFixtureReturnExpectation(metadata.Agent, name)
			if !ok {
				t.Errorf("%s: raw file %q has no recorded Return expectation", metadataPath, name)
				continue
			}
			t.Run(fixtureLabel(metadata)+"/"+name, func(t *testing.T) {
				replayHarnessTTYFixture(t, metadata.Agent, rawFiles[name], wantComposer)
			})
		}
	}

	var missing []string
	for harness, found := range required {
		if !found {
			missing = append(missing, harness)
		}
	}
	sort.Strings(missing)
	if len(missing) != 0 {
		t.Fatalf("required positive-gated fixtures missing: %s", strings.Join(missing, ", "))
	}

	// A positive gate with no captured declining state has no evidence it
	// refuses anything. `go test` suppresses passing-package output entirely,
	// so a log line here would be invisible in `make test` — the gap is
	// therefore acknowledged in code, where review can see it, and any harness
	// that is neither covered nor acknowledged fails.
	for harness, found := range negatives {
		reason, acknowledged := ttyFixtureNegativeGaps[harness]
		switch {
		case found && acknowledged:
			t.Errorf("%s now has a captured declining state; drop its ttyFixtureNegativeGaps entry (%q)", harness, reason)
		case !found && !acknowledged:
			t.Errorf("%s is positively gated with no captured declining state and no recorded reason: capture one, or record why it cannot be captured in ttyFixtureNegativeGaps", harness)
		}
		// Having *a* declining state is not the same as proving the gate
		// separates a live composer from a picker painted in the same shape.
		// A harness that declines only on cursor state has not shown that, so
		// it must say so rather than reading as covered.
		if _, listed := ttyFixtureDiscriminationGaps[harness]; !listed && !harnessHasDiscriminatingNegative(harness) {
			t.Errorf("%s has neither a discriminating negative nor a ttyFixtureDiscriminationGaps entry", harness)
		}
	}

	// Every positively gated harness remaps Return on SOME screen, and what the
	// harness then does with the remapped key is the one thing no fixture
	// replay can answer — a replay proves what the wrapper emitted, full stop.
	// That includes the composer itself: Muse reading `ESC[13;2u` as a newline
	// is inferred from its own shortcut sheet plus a KKP push, never driven. So
	// each harness must either press Return in a driven scenario or carry a
	// reaction-gap entry; a comment asserting the outcome is not evidence
	// (#266 close BR-14, BR-15).
	for harness := range required {
		reason, acknowledged := ttyFixtureReactionGaps[harness]
		driven := harnessPressesReturn(harness)
		switch {
		case driven && acknowledged:
			t.Errorf("%s now presses Return on a captured screen; drop its ttyFixtureReactionGaps entry (%q)", harness, reason)
		case !driven && !acknowledged:
			t.Errorf("%s remaps Return with no driven Return and no ttyFixtureReactionGaps entry: drive it, or record what is unproven", harness)
		}
	}

	// The discrimination ledger gets the same exact expiry the negative ledger
	// has. It had none at all, so an entry outlived its gap silently; and
	// "has an overlay.raw" was never the property it acknowledges — a negative
	// that declines on a hidden cursor proves nothing about a picker painted in
	// the composer's shape. That property is now declared by the scenario that
	// reaches such a screen (#266 close BR-15).
	for harness := range required {
		reason, acknowledged := ttyFixtureEnvironmentGaps[harness]
		if acknowledged && !embedsHomePath[harness] {
			t.Errorf("%s captures are all machine-neutral now; drop its ttyFixtureEnvironmentGaps entry (%q)", harness, reason)
		}
	}

	for harness := range required {
		reason, listed := ttyFixtureDiscriminationGaps[harness]
		if harnessHasDiscriminatingNegative(harness) && listed {
			t.Errorf("%s now has a discriminating negative; drop its ttyFixtureDiscriminationGaps entry (%q)", harness, reason)
		}
	}

	for _, ledger := range []struct {
		name    string
		entries map[string]string
	}{
		{"ttyFixtureNegativeGaps", ttyFixtureNegativeGaps},
		{"ttyFixtureDiscriminationGaps", ttyFixtureDiscriminationGaps},
		{"ttyFixtureReactionGaps", ttyFixtureReactionGaps},
	} {
		for harness := range ledger.entries {
			if _, ok := harnessTTYProfiles[harness]; !ok {
				t.Errorf("%s names %q, which has no profile", ledger.name, harness)
			}
		}
	}
}

// ttyFixtureNegativeGaps records positively gated harnesses with no captured
// state where the gate declines, and why. An entry is a known hole in the
// evidence, not a waiver of the requirement: it must name what is missing so
// the gap is reviewable, and it must be removed once the capture exists.
var ttyFixtureNegativeGaps = map[string]string{
	"claude": "Claude's declining state is its permission prompt. A child spawned from an agent session inherits auto-approve mode — verified 2026-08-20: the child ran Bash(uptime) and returned output with no prompt, despite `uptime` not being allowlisted — so the prompt is unreachable from here. The route is registered as the `permission prompt` scenario in harnessTTYDrivenScenarios; run it from a plain terminal with default (ask) permissions to capture overlay.raw.",
	"muse":   "Muse's slash menu and `?` shortcut sheet were both driven live on 1.3.0-R3233.1 (see harnessTTYDrivenScenarios) and neither declines: each paints below the composer box and leaves it intact, so menu.raw is discrimination evidence rather than a negative. The remaining declining state is a tool-approval dialog, which needs a real tool call; capture overlay.raw when one is available.",
}

// ttyFixtureReactionGaps records claims about how a harness REACTS to bytes we
// emit that are not backed by a capture or a live drive. A gate's decision can
// be replayed from a fixture; what the harness then does with the remapped key
// cannot, so such a claim either gets driven or gets recorded here. An entry
// names the screen, the unproven reaction, and how to drive it.
var ttyFixtureReactionGaps = map[string]string{
	"muse":   "no Return has been pressed on any captured Muse screen. That `ESC[13;2u` inserts a newline in the composer is inferred from Muse's own shortcut sheet (`shortcuts.raw`: \"shift + enter for newline\") plus the KKP push in every capture, and from live operator smoke — not from a driven assertion. On the slash menu, whether Muse leaves the highlighted command unpicked (the Agy tradeoff) is likewise undriven.",
	"agy":    "no Return has been pressed on any captured Agy screen. That Agy inserts a newline on LF in its slash menu rather than selecting is an observation from reading the screen, not a driven result.",
	"claude": "no Return has been pressed on any captured Claude screen. The gate stays open on the slash menu and in bash mode, so Return remaps to Claude's newline on both; what Claude does with it is undriven.",
	"codex":  "no Return has been pressed on any captured Codex screen. The composer remap to LF is undriven; the overlay path is the one with independent evidence, since the interstitial's own footer says Enter continues.",
	"qoder":  "no Return has been pressed on any captured Qoder screen. That `\\\\<CR>` inserts a newline in the composer is inferred from Qoder sharing Claude's keymap convention; what Qoder does with it is undriven. The overlay path has better evidence than most: the question picker's own footer (selection.raw) says `↑↓navigate·Enterselect·Esccancel`, and the permission picker holds focus the same way — but nobody has pressed Return on either screen, so that a bare CR confirms the highlighted choice remains inferred.",
}

// harnessPressesReturn reports whether a harness has a driven scenario that
// actually presses Return on the screen a fixture file captures. That is the
// only thing that retires a reaction gap: a fixture replay can prove what the
// wrapper emits, never what the harness makes of it.
func harnessPressesReturn(harness string) bool {
	for _, scenario := range harnessTTYDrivenScenarios[harness] {
		if scenario.pressesReturn {
			return true
		}
	}
	return false
}

// harnessHasDiscriminatingNegative reports whether a harness has a declining
// screen painted in its composer's own shape — the exact property
// ttyFixtureDiscriminationGaps acknowledges the absence of. Two of its three
// components are expressible, so the oracle conjoins them rather than trusting
// one boolean: the screen must be one the gate DECLINES, and it must actually
// be captured. Only "painted in the composer's own shape" is honor-system,
// which is the residue the field's doc owns (#266 close BR-19).
func harnessHasDiscriminatingNegative(harness string) bool {
	for _, scenario := range harnessTTYDrivenScenarios[harness] {
		if !scenario.discriminating || scenario.wantComposer || scenario.file == "" {
			continue
		}
		captures, err := filepath.Glob(filepath.Join("testdata", "tty", harness, "*", scenario.file))
		if err == nil && len(captures) > 0 {
			return true
		}
	}
	return false
}

// ttyFixtureDiscriminationGaps records positively gated harnesses whose
// captured declining states do NOT prove the gate separates a live composer
// from a picker or menu painted in the same shape. Codex is absent because its
// overlay.raw is exactly that proof: the update interstitial paints the same
// U+203A at column 0 and is rejected on emphasis alone.
var ttyFixtureDiscriminationGaps = map[string]string{
	"claude": "no captured declining state at all; see ttyFixtureNegativeGaps. Claude does not reuse its prompt glyph as a menu marker — menu.raw pins its slash menu rendering below the box with column 0 blank — so the Agy failure mode does not apply; what is still unproven is a blocking dialog the gate must refuse.",
	"agy":    "agy/1.1.15/overlay.raw declines on hidden cursor and cursor position, not on any composer-vs-picker rule, and menu.raw shows Agy painting a menu marker in the SAME bright blue as the composer prompt. The permission-picker capture is reachable by dropping --dangerously-skip-permissions from the agy driven scenario and driving one tool call; attempted 2026-08-19 and blocked, the account was in \"Verifying your account...\" and would not execute tool calls.",
	"muse":   "no captured declining state at all; see ttyFixtureNegativeGaps. muse/1.3.0-R3233.1/menu.raw does rule out the Agy failure mode: Muse's slash menu paints its rows below the box and leaves column 0 blank, so it never reuses the prompt glyph as a selection marker. What is still unproven is a blocking dialog the gate must refuse.",
	"qoder":  "qoder/1.1.60 declines on two captured screens, neither painted in the composer's own shape: overlay.raw's permission picker replaces the box entirely, and selection.raw's question picker paints a ruled card whose option rows put `❯` at column 0 where the composer keeps `>` at column 1 — the recognizer refuses on shape in both cases rather than on any composer-vs-picker rule. menu.raw shows the slash menu painting below a live box, so the Agy failure mode does not apply. What is still unproven is a blocking dialog painted inside a live composer box.",
}

// ttyFixtureEnvironmentGaps records harnesses whose captures cannot be taken
// without embedding the capture machine's filesystem, and why. An entry is a
// measured impossibility, not a waiver: it must name what blocks a neutral
// capture, and it expires the moment one is clean.
var ttyFixtureEnvironmentGaps = map[string]string{
	"muse": "Muse prints `warning: rules file at <abs>/CLAUDE.md is ignored ... because AGENTS.md takes precedence` whenever it starts anywhere inside this repo — it reads the rules files from the git root, so capturing from a subdirectory does not help (measured 2026-09-16 from testdata/tty: the warning is still painted). Capturing from outside the checkout is not available either: a fresh directory puts Muse in workspace-trust, which the conformance classifier reports as `workspace-trust` rather than a composer. So every Muse capture carries one absolute path until Muse stops printing it or offers a way to silence it.",
}

// assertFixtureIsMachineNeutral requires a frozen capture to contain no
// absolute home path. A fixture that embeds the capture machine's filesystem
// differs on recapture elsewhere for reasons unrelated to harness drift, which
// is the only thing it exists to detect. The check is mechanical rather than a
// convention because these were the first machine-bound captures in the tree
// and nothing noticed (#266 close BR-21).
func assertFixtureIsMachineNeutral(t *testing.T, metadataPath, harness string, rawFiles map[string][]byte) bool {
	t.Helper()
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("resolve home directory: %v", err)
	}
	homeRoot := filepath.Dir(home) + string(filepath.Separator)
	_, acknowledged := ttyFixtureEnvironmentGaps[harness]
	embedded := false
	for _, name := range sortedKeys(rawFiles) {
		body := string(rawFiles[name])
		if !strings.Contains(body, home) && !strings.Contains(body, homeRoot) {
			continue
		}
		embedded = true
		if !acknowledged {
			t.Errorf("%s: %s embeds an absolute home path; recapture somewhere neutral, or record in ttyFixtureEnvironmentGaps why that is not possible", metadataPath, name)
		}
	}
	// The acknowledgment is per harness, so its expiry is judged across ALL of
	// that harness's captures at once, by the caller — an older capture taken
	// before the harness started printing the path is clean without making the
	// gap stale.
	return embedded
}

func readHarnessTTYFixture(t *testing.T, metadataPath string) (ttyFixtureMetadata, map[string][]byte, bool) {
	t.Helper()
	encoded, err := os.ReadFile(metadataPath)
	if err != nil {
		t.Fatalf("read %s: %v", metadataPath, err)
	}
	var metadata ttyFixtureMetadata
	if err := json.Unmarshal(encoded, &metadata); err != nil {
		t.Fatalf("decode %s: %v", metadataPath, err)
	}
	if metadata.Agent == "" || strings.TrimSpace(metadata.Version) != metadata.Version || metadata.Version == "" {
		t.Errorf("%s: agent/version must be non-empty and version exactly trimmed", metadataPath)
	}
	if _, err := time.Parse(time.RFC3339, metadata.CapturedAt); err != nil {
		t.Errorf("%s: captured_at %q is not RFC3339: %v", metadataPath, metadata.CapturedAt, err)
	}
	if len(metadata.Command) == 0 {
		t.Errorf("%s: command argv is empty", metadataPath)
	}
	for i, arg := range metadata.Command {
		if arg == "" {
			t.Errorf("%s: command argv[%d] is empty", metadataPath, i)
		}
	}

	dir := filepath.Dir(metadataPath)
	rawFiles := map[string][]byte{}
	err = filepath.WalkDir(dir, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".raw" {
			return nil
		}
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		rawFiles[entry.Name()] = data
		return nil
	})
	if err != nil {
		t.Fatalf("inventory raw files beside %s: %v", metadataPath, err)
	}
	if len(metadata.Files) != len(rawFiles) {
		t.Errorf("%s: metadata has %d digests for %d raw files", metadataPath, len(metadata.Files), len(rawFiles))
	}
	machineBound := assertFixtureIsMachineNeutral(t, metadataPath, metadata.Agent, rawFiles)
	for name, data := range rawFiles {
		digest := sha256.Sum256(data)
		want := hex.EncodeToString(digest[:])
		got, ok := metadata.Files[name]
		if !ok {
			t.Errorf("%s: raw file %q has no digest", metadataPath, name)
		} else if got != want {
			t.Errorf("%s: digest for %q = %q, want %q", metadataPath, name, got, want)
		}
	}
	for name, digest := range metadata.Files {
		if _, ok := rawFiles[name]; !ok {
			t.Errorf("%s: dangling digest for %q", metadataPath, name)
		}
		if len(digest) != sha256.Size*2 || strings.ToLower(digest) != digest {
			t.Errorf("%s: digest for %q is not lowercase SHA-256: %q", metadataPath, name, digest)
		}
		if _, err := hex.DecodeString(digest); err != nil {
			t.Errorf("%s: digest for %q is not hex: %v", metadataPath, name, err)
		}
	}
	if len(rawFiles) == 0 {
		t.Errorf("%s: fixture contains no raw files", metadataPath)
	}
	return metadata, rawFiles, machineBound
}

func fixtureLabel(metadata ttyFixtureMetadata) string {
	return fmt.Sprintf("%s/%s", metadata.Agent, ttyFixtureVersionDir(metadata.Version))
}

// ttyFixtureExpectation records what each captured state must decide for a
// plain Return. A composer capture must remap to a newline; an overlay capture
// must pass a bare CR through so the overlay confirms.
var ttyFixtureExpectation = map[string]map[string]bool{
	// Defaults every harness inherits.
	"": {
		"composer.raw": true,
		"overlay.raw":  false,
	},
	// Agy's slash menu keeps the composer live below its own box, so the gate
	// stays open and Return remaps there. What this entry checks is the
	// wrapper's half; the tolerability argument rests on Agy inserting a
	// newline rather than selecting, which ttyFixtureReactionGaps records as
	// undriven. Keyed per harness because another harness's menu may have to
	// decline.
	"agy": {"menu.raw": true},
	// Claude's bash mode is still a composer — it repaints the glyph and rule
	// colour, not the shape — so the gate must stay open.
	"claude": {"bash-mode.raw": true, "menu.raw": true},
	// Muse's slash menu paints below the composer box with column 0 blank, so
	// the box shape still selects the composer and the gate stays open. What
	// this entry checks is the wrapper's half: a plain Return on that screen
	// emits the profile's Shift+Return rather than a bare CR. What Muse then
	// does with it — whether the highlighted command stays unpicked — was not
	// driven, and is recorded in ttyFixtureReactionGaps rather than asserted
	// here. The escape hatch needs no capture either way: Alt+Return reaches
	// Muse as the bare CR it submits on natively.
	// shortcuts.raw is the `?` sheet — Muse printing its own key contract,
	// which is the evidence this profile's inverted keymap rests on. It paints
	// below the box like the menu, so the gate stays open there too.
	"muse": {"menu.raw": true, "shortcuts.raw": true},
	// Qoder's slash menu paints below the composer box and leaves the box —
	// prompt glyph included — intact, so the gate stays open and Return remaps
	// there. The menu's own `❯` selection marker lands in the prompt glyph's
	// column but below the closing rule, which the box-bounded recognizer
	// ignores. The permission picker (overlay.raw) and the question picker
	// (selection.raw) both take the shared declining default; both are caught
	// by qoderPickerMarkers, so their decline survives a recognizer that
	// would otherwise accept a ruled card.
	"qoder": {"menu.raw": true, "selection.raw": false},
}

// ttyFixtureReturnExpectation reports whether a fixture file must remap Return,
// preferring a harness-specific entry over the shared default.
func ttyFixtureReturnExpectation(harness, file string) (bool, bool) {
	if want, ok := ttyFixtureExpectation[harness][file]; ok {
		return want, true
	}
	want, ok := ttyFixtureExpectation[""][file]
	return want, ok
}

// composerReturnBytes reports what a harness emits for a plain Return inside a
// recognized composer, read from the profile registry rather than restated.
func composerReturnBytes(harness string) (string, bool) {
	profile, ok := profileForHarness(harness, true)
	if !ok {
		return "", false
	}
	return string(profile.keymap.plainCR), true
}

// harnessTTYReplayResult is the observable result of replaying a fixture
// through the production seam: what the recognizer saw and what a plain Return
// would actually emit.
type harnessTTYReplayResult struct {
	recognized   bool
	overlayArmed bool
	returnBytes  string
	reason       string
}

// replayHarnessTTYFixture feeds raw through the production proxy unsplit to
// establish a baseline, then repeats the feed at every possible split point.
// Chunk boundaries are an artifact of PTY scheduling, so no split may change
// what the user's next Return does.
func replayHarnessTTYFixture(t *testing.T, harness string, raw []byte, wantComposer bool) {
	t.Helper()
	baseline := replayHarnessTTYSplit(t, harness, raw, len(raw))
	if baseline.recognized != wantComposer {
		t.Fatalf("unsplit recognizer = %t, want %t (%s)", baseline.recognized, wantComposer, baseline.reason)
	}
	wantReturn := "\r"
	if wantComposer {
		remap, ok := composerReturnBytes(harness)
		if !ok {
			t.Fatalf("no profile for %s", harness)
		}
		wantReturn = remap
	}
	if baseline.returnBytes != wantReturn {
		t.Fatalf("unsplit plain Return = %q, want %q (%s)", baseline.returnBytes, wantReturn, baseline.reason)
	}
	// Every split is O(n) feeds of an O(n) stream plus two full-grid snapshot
	// clones, so exhaustive coverage grows superlinearly with fixture size.
	// Cover the prefix exhaustively — that is where escape sequences that
	// establish the composer live — then stride the tail, and say what was
	// skipped rather than capping silently.
	skipped := 0
	for split := 0; split <= len(raw); split++ {
		if split > harnessTTYExhaustiveSplitBytes && split%harnessTTYSplitStride != 0 && split != len(raw) {
			skipped++
			continue
		}
		if got := replayHarnessTTYSplit(t, harness, raw, split); got != baseline {
			t.Fatalf("split at %d/%d changed the Return decision: %+v, want %+v", split, len(raw), got, baseline)
		}
	}
	if skipped != 0 {
		t.Logf("replayed all splits through %d bytes, then every %dth of %d; %d splits sampled out",
			harnessTTYExhaustiveSplitBytes, harnessTTYSplitStride, len(raw), skipped)
	}
}

const (
	// harnessTTYExhaustiveSplitBytes is the prefix length covered at every
	// split; it comfortably spans each captured composer's establishing paint.
	harnessTTYExhaustiveSplitBytes = 1024
	// harnessTTYSplitStride samples the remainder.
	harnessTTYSplitStride = 7
)

// replayHarnessTTYSplit feeds raw as two chunks divided at split and reports
// what a plain Return would emit afterwards.
func replayHarnessTTYSplit(t *testing.T, harness string, raw []byte, split int) harnessTTYReplayResult {
	t.Helper()
	p := &proxy{agentBasename: harness}
	if err := p.configureHarnessTTY(true, 120, 38); err != nil {
		t.Fatalf("configure %s replay proxy: %v", harness, err)
	}
	defer func() {
		if err := p.closeTerminal(); err != nil {
			t.Errorf("close %s replay proxy: %v", harness, err)
		}
	}()
	var rolling []byte
	p.handleChunk(raw[:split], &rolling)
	p.handleChunk(raw[split:], &rolling)

	snapshot := p.terminal.Snapshot()
	result := harnessTTYReplayResult{
		recognized:   p.ttyProfile.recognize(snapshot),
		overlayArmed: p.pickerActive.Load(),
	}
	// emitPlainCR is the production Return path: it consumes overlay state
	// under the same lock the detector arms it with, then decides.
	result.returnBytes = string(p.emitPlainCR(nil))
	result.reason = decidePlainReturn(*p.ttyProfile, result.overlayArmed, &snapshot).reason
	return result
}

// TestDocCommentsNameTheirSubject checks the one claim a doc comment makes that
// the tree can verify: when it opens with a camelCase SYMBOL name, that symbol
// must exist and must be the thing being documented (or, for a test, the symbol
// under test). Most comments here open with prose, which says nothing checkable
// and is left alone. The two failure modes this catches both happened in this
// package: a comment opening on a function that was later renamed (#266), and a
// comment stranded above a declaration inserted underneath it (#184). Sibling of
// TestTTYFixtureReferencesResolve, over doc groups instead of paths.
func TestDocCommentsNameTheirSubject(t *testing.T) {
	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, ".", nil, parser.ParseComments)
	if err != nil {
		t.Fatalf("parse package: %v", err)
	}
	declared := map[string]bool{}
	var subjects []docCommentSubject
	for _, pkg := range pkgs {
		for name, file := range pkg.Files {
			for _, decl := range file.Decls {
				for _, named := range declaredNames(decl) {
					declared[named] = true
				}
				for _, subject := range docCommentSubjects(decl) {
					subject.file = filepath.Base(name)
					subject.line = fset.Position(subject.pos).Line
					subjects = append(subjects, subject)
				}
			}
		}
	}
	if len(declared) == 0 || len(subjects) == 0 {
		t.Fatal("no declarations with doc comments found; this test has stopped checking anything")
	}
	for _, subject := range subjects {
		lead, ok := leadingSymbol(subject.doc)
		if !ok {
			continue
		}
		switch {
		case !declared[lead]:
			t.Errorf("%s:%d: doc comment opens with %q, which this package does not declare",
				subject.file, subject.line, lead)
		case lead != subject.name && !strings.Contains(strings.ToLower(subject.name), strings.ToLower(lead)):
			t.Errorf("%s:%d: doc comment opens with %q but documents %q",
				subject.file, subject.line, lead, subject.name)
		}
	}
}

type docCommentSubject struct {
	name string
	doc  *ast.CommentGroup
	pos  token.Pos
	file string
	line int
}

// declaredNames lists the names a declaration introduces.
func declaredNames(decl ast.Decl) []string {
	switch d := decl.(type) {
	case *ast.FuncDecl:
		return []string{d.Name.Name}
	case *ast.GenDecl:
		var names []string
		for _, spec := range d.Specs {
			switch sp := spec.(type) {
			case *ast.TypeSpec:
				names = append(names, sp.Name.Name)
			case *ast.ValueSpec:
				for _, name := range sp.Names {
					names = append(names, name.Name)
				}
			}
		}
		return names
	}
	return nil
}

// docCommentSubjects pairs a declaration's doc comment with the name it should
// name. A grouped `var (...)`/`const (...)` block documents the group rather
// than one name, so only its individually documented specs are paired.
func docCommentSubjects(decl ast.Decl) []docCommentSubject {
	switch d := decl.(type) {
	case *ast.FuncDecl:
		if d.Doc == nil {
			return nil
		}
		return []docCommentSubject{{name: d.Name.Name, doc: d.Doc, pos: d.Pos()}}
	case *ast.GenDecl:
		var subjects []docCommentSubject
		for _, spec := range d.Specs {
			var (
				name string
				doc  *ast.CommentGroup
			)
			switch sp := spec.(type) {
			case *ast.TypeSpec:
				name, doc = sp.Name.Name, sp.Doc
			case *ast.ValueSpec:
				if len(sp.Names) == 0 {
					continue
				}
				name, doc = sp.Names[0].Name, sp.Doc
			default:
				continue
			}
			if doc == nil && len(d.Specs) == 1 {
				doc = d.Doc
			}
			if doc != nil {
				subjects = append(subjects, docCommentSubject{name: name, doc: doc, pos: spec.Pos()})
			}
		}
		return subjects
	}
	return nil
}

// leadingSymbol reports a doc comment's first word when it is shaped like a Go
// symbol rather than prose: mixed case with an internal capital and nothing but
// identifier characters. "drivenReturnScenario" qualifies; "The", "Codex" and
// "NO_COLOR" do not, and neither does a word carrying punctuation.
func leadingSymbol(doc *ast.CommentGroup) (string, bool) {
	fields := strings.Fields(strings.TrimSpace(doc.Text()))
	if len(fields) < 2 {
		return "", false
	}
	word := fields[0]
	internalCapital := false
	for i, r := range word {
		switch {
		case r == '_' || r >= '0' && r <= '9':
			return "", false
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z':
		default:
			return "", false
		}
		if i > 0 && r >= 'A' && r <= 'Z' && word[i-1] >= 'a' && word[i-1] <= 'z' {
			internalCapital = true
		}
	}
	return word, internalCapital
}

// ttyFixtureReference matches a fixture path — in code or in a comment — as
// the agent/version directory under the fixture root, optionally with a `*`
// glob or a file name.
var ttyFixtureReference = regexp.MustCompile(`testdata/tty/[A-Za-z0-9._*-]+(?:/[A-Za-z0-9._*-]+)*`)

// TestTTYFixtureReferencesResolve requires every fixture path this package
// mentions to exist — including the ones named only in comments. A comment that
// cites a fixture directory is a claim the tree can check, so nothing should
// have to notice by reading: #266 shipped a profile comment pointing at a
// fixture the same change deleted, and a version directory is exactly the kind
// of referent that dies quietly when a harness self-updates.
func TestTTYFixtureReferencesResolve(t *testing.T) {
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("read package directory: %v", err)
	}
	checked := 0
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") {
			continue
		}
		source, err := os.ReadFile(entry.Name())
		if err != nil {
			t.Fatalf("read %s: %v", entry.Name(), err)
		}
		for _, match := range ttyFixtureReference.FindAllString(string(source), -1) {
			// A path at the end of a sentence keeps the period; a bare root
			// with no agent is the regexp in this file, not a reference.
			reference := strings.TrimRight(match, ".")
			if strings.Count(reference, "/") < 3 {
				continue
			}
			checked++
			if strings.Contains(reference, "*") {
				hits, err := filepath.Glob(reference)
				if err != nil {
					t.Errorf("%s: bad fixture glob %q: %v", entry.Name(), reference, err)
				} else if len(hits) == 0 {
					t.Errorf("%s: fixture glob %q matches nothing", entry.Name(), reference)
				}
				continue
			}
			if _, err := os.Stat(reference); err != nil {
				t.Errorf("%s: fixture reference %q does not resolve: %v", entry.Name(), reference, err)
			}
		}
	}
	if checked == 0 {
		t.Fatal("no fixture references found; this test has stopped checking anything")
	}
}

// plainCRNeedsKittyKeyboard reports whether a profile's composer Return bytes
// are a Kitty keyboard protocol key — CSI <params> u — rather than a literal
// control byte. Muse's newline is one (`\x1b[13;2u`, Shift+Return).
func plainCRNeedsKittyKeyboard(plainCR []byte) bool {
	return len(plainCR) > len("\x1b[u") && strings.HasPrefix(string(plainCR), "\x1b[") && plainCR[len(plainCR)-1] == 'u'
}

// kittyKeyboardPush matches a progressive-enhancement push, CSI > <flags> u,
// capturing the flag word.
var kittyKeyboardPush = regexp.MustCompile("\x1b\\[>([0-9]*)(?:;[0-9]*)?u")

// kittyDisambiguateFlag is bit 0 of the KKP flag word, "disambiguate escape
// codes" — the level at which a modified key is reported as CSI code;mods u.
// A push is not enough on its own: `CSI > 0 u` is a push that DISABLES every
// enhancement, and any flag word without this bit leaves a modified Return
// unencodable, so it would not parse the sequence we emit.
const kittyDisambiguateFlag = 1

// kittyKeyboardDisambiguates reports whether a capture pushes KKP with the
// disambiguate bit set.
func kittyKeyboardDisambiguates(raw []byte) bool {
	for _, match := range kittyKeyboardPush.FindAllSubmatch(raw, -1) {
		flags, err := strconv.Atoi(string(match[1]))
		if err != nil {
			// An omitted flag word defaults to 1 in the protocol.
			flags = kittyDisambiguateFlag
		}
		if flags&kittyDisambiguateFlag != 0 {
			return true
		}
	}
	return false
}

// assertKittyKeyboardPrecondition fails a harness whose composer Return is
// KKP-encoded but whose own capture pushes no Kitty keyboard flags. That
// precondition is invisible to every other assertion here: both the frozen
// replay and the live check read their expected bytes FROM the profile, so they
// agree with it by construction and cannot notice the day a harness stops
// parsing the encoding. The failure is otherwise silent in the worst way —
// Return does nothing at all while return-remap telemetry still reports
// `fired`, because the wrapper did emit its remap (#266 close BR-2).
func assertKittyKeyboardPrecondition(t *testing.T, harness string, plainCR, raw []byte) {
	t.Helper()
	if !plainCRNeedsKittyKeyboard(plainCR) || kittyKeyboardDisambiguates(raw) {
		return
	}
	t.Errorf("%s composer Return is KKP-encoded (%q) but its capture pushes no Kitty keyboard flags with the disambiguate bit set: the harness would read those bytes as literal text", harness, plainCR)
}

func sortedKeys(files map[string][]byte) []string {
	names := make([]string, 0, len(files))
	for name := range files {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// TestComposerReturnExpectationMatchesProfile pins that a recognized composer's
// expected Return bytes come from the harness's own keymap, not from a constant
// restated here. Codex and Agy remap to LF, which made a hardcoded "\n" look
// correct until Claude arrived with `\\<CR>` and Muse with a KKP Shift+Return.
// Each value below is the harness's documented composer newline; the profile is
// the authority the production path reads.
func TestComposerReturnExpectationMatchesProfile(t *testing.T) {
	want := map[string]string{
		"claude": "\\\r",
		"codex":  "\n",
		"muse":   "\x1b[13;2u",
		"agy":    "\n",
		"qoder":  "\\\r",
	}
	for harness, wantBytes := range want {
		got, ok := composerReturnBytes(harness)
		if !ok {
			t.Errorf("%s: no profile", harness)
			continue
		}
		if got != wantBytes {
			t.Errorf("%s composer Return = %q, want %q", harness, got, wantBytes)
		}
	}
}
