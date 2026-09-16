package wrapcmd

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
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
		metadata, rawFiles := readHarnessTTYFixture(t, metadataPath)
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
		if _, listed := ttyFixtureDiscriminationGaps[harness]; !listed && !found {
			t.Errorf("%s has neither a discriminating negative nor a ttyFixtureDiscriminationGaps entry", harness)
		}
	}

	// A `menu.raw: true` expectation says the gate deliberately stays OPEN on a
	// menu screen, which makes what the harness does with the remapped Return
	// the whole question — and that is the one thing no fixture replay can
	// answer. So each such harness must either press Return in a driven
	// scenario or carry a reaction-gap entry; a comment asserting the outcome
	// is not evidence (#266 close BR-14).
	for harness, files := range ttyFixtureExpectation {
		if harness == "" || !anyOpenGateScreen(files) {
			continue
		}
		reason, acknowledged := ttyFixtureReactionGaps[harness]
		driven := drivenReturnOnOpenGateScreen(harness, files)
		switch {
		case driven && acknowledged:
			t.Errorf("%s now drives Return on its menu capture; drop its ttyFixtureReactionGaps entry (%q)", harness, reason)
		case !driven && !acknowledged:
			t.Errorf("%s keeps its gate open on a non-composer screen with no driven Return and no ttyFixtureReactionGaps entry: drive it, or record what is unproven", harness)
		}
	}
	for harness := range ttyFixtureReactionGaps {
		if _, ok := harnessTTYProfiles[harness]; !ok {
			t.Errorf("ttyFixtureReactionGaps names %q, which has no profile", harness)
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
	"muse":   "muse/1.3.0-R3233.1/menu.raw pins that plain Return on the slash menu emits Shift+Return, but nobody has pressed it there: whether Muse leaves the highlighted command unpicked (the Agy tradeoff) or acts on it is undriven.",
	"agy":    "the agy menu.raw comment states that Agy inserts a newline on LF there rather than selecting. That is an observation from reading the screen, not a driven result; the keystroke was never sent.",
	"claude": "claude/2.1.237/menu.raw pins that the gate stays open on the slash menu, so plain Return remaps to Claude's newline there. What Claude does with it — and with a Return in bash mode — is undriven.",
}

// drivenReturnScenario reports whether a harness has a driven scenario that
// actually presses Return on the screen a fixture file captures. That is the
// only thing that retires a reaction gap: a fixture replay can prove what the
// wrapper emits, never what the harness makes of it.
func drivenReturnOnOpenGateScreen(harness string, files map[string]bool) bool {
	for _, scenario := range harnessTTYDrivenScenarios[harness] {
		if files[scenario.file] && scenario.file != "composer.raw" && strings.Contains(scenario.send, "\r") {
			return true
		}
	}
	return false
}

// anyOpenGateScreen reports whether a harness pins any NON-composer capture the
// gate must stay open on — the screens where the remapped Return is a tradeoff
// rather than the intended edit.
func anyOpenGateScreen(files map[string]bool) bool {
	for file, open := range files {
		if open && file != "composer.raw" {
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
}

func readHarnessTTYFixture(t *testing.T, metadataPath string) (ttyFixtureMetadata, map[string][]byte) {
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
	return metadata, rawFiles
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
	// Agy's slash menu keeps the composer live below its own box. The gate
	// stays open, which is safe only because Agy inserts a newline on LF there
	// rather than selecting; pinned so that stays a checked property. Keyed per
	// harness because another harness's menu may have to decline.
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
