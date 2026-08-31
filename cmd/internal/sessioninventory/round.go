package sessioninventory

import (
	"cmp"
	"crypto/sha256"
	"fmt"
	"slices"
	"unicode"
	"unicode/utf8"
)

// NativeEventFact locates one accepted event within a scanner-authorized root.
type NativeEventFact struct {
	Agent      Agent
	RootNodeID string
	Position   uint64
	Event      NativeEvent
}

// RoundObservation is an exact Pair/native turn correspondence whose native
// operator event has subsequent accepted progress before the next operator.
// Texts are transient matcher input; durable/public evidence uses fingerprints
// and positions rather than copying authored content.
// pair:155-concept pure new M2
type RoundObservation struct {
	ScopeKey          string   `json:"scope_key"`
	Tag               string   `json:"tag"`
	Agent             Agent    `json:"agent"`
	RootNodeID        string   `json:"root_node_id"`
	PairPositions     []uint64 `json:"pair_positions"`
	NativePositions   []uint64 `json:"native_positions"`
	ProgressPositions []uint64 `json:"progress_positions"`
	Fingerprints      []string `json:"fingerprints"`
	Texts             []string `json:"-"`
}

type normalizedTurn struct {
	owner            string
	scopeKey         string
	tag              string
	agent            Agent
	position         uint64
	text             string
	fingerprint      string
	words            int
	progressPosition uint64
	completed        bool
}

// QualifyTurnSequence applies the exact one/two-turn thresholds and global
// uniqueness rules to the post-launch Pair/native suffixes.
func QualifyTurnSequence(pairFacts []PairLogFact, nativeFacts []NativeEventFact) []RoundObservation {
	pairTurns := make([]normalizedTurn, 0, len(pairFacts))
	for _, fact := range pairFacts {
		text := NormalizePairText(fact.Text)
		if text == "" || !utf8.ValidString(text) {
			continue
		}
		turn := makeNormalizedTurn(fact.Position, text)
		turn.scopeKey, turn.tag, turn.agent = fact.ScopeKey, fact.Tag, fact.Agent
		turn.owner = pairOwnerKey(fact.ScopeKey, fact.Tag, fact.Agent)
		pairTurns = append(pairTurns, turn)
	}
	slices.SortFunc(pairTurns, func(a, b normalizedTurn) int {
		if result := cmp.Compare(a.owner, b.owner); result != 0 {
			return result
		}
		return cmp.Compare(a.position, b.position)
	})
	if !uniqueTurnPositions(pairTurns) {
		return nil
	}

	rootEvents := map[string][]NativeEventFact{}
	for _, fact := range nativeFacts {
		if fact.RootNodeID == "" {
			continue
		}
		fact.Event.Text = NormalizePairText(fact.Event.Text)
		rootEvents[fact.RootNodeID] = append(rootEvents[fact.RootNodeID], fact)
	}
	roots := make([]string, 0, len(rootEvents))
	for root := range rootEvents {
		roots = append(roots, root)
	}
	slices.Sort(roots)
	nativeTurns := map[string][]normalizedTurn{}
	for _, root := range roots {
		events := rootEvents[root]
		slices.SortFunc(events, func(a, b NativeEventFact) int {
			if result := cmp.Compare(a.Position, b.Position); result != 0 {
				return result
			}
			if result := cmp.Compare(a.Event.Kind, b.Event.Kind); result != 0 {
				return result
			}
			return cmp.Compare(a.Event.SourceKind, b.Event.SourceKind)
		})
		duplicatePosition := false
		for i := 1; i < len(events); i++ {
			if events[i-1].Position == events[i].Position {
				duplicatePosition = true
				break
			}
		}
		if duplicatePosition {
			continue
		}
		for i, fact := range events {
			if fact.Event.Kind != EventOperator || fact.Event.Text == "" || !utf8.ValidString(fact.Event.Text) {
				continue
			}
			turn := makeNormalizedTurn(fact.Position, fact.Event.Text)
			turn.agent = fact.Agent
			for j := i + 1; j < len(events); j++ {
				if events[j].Event.Kind == EventOperator {
					break
				}
				if events[j].Event.Progress() {
					turn.completed = true
					turn.progressPosition = events[j].Position
					break
				}
			}
			nativeTurns[root] = append(nativeTurns[root], turn)
		}
	}

	pairSingles := countTurnFingerprints(pairTurns)
	nativeSingles := map[string]int{}
	for _, root := range roots {
		for fingerprint, count := range countTurnFingerprints(nativeTurns[root]) {
			nativeSingles[fingerprint] += count
		}
	}
	var observations []RoundObservation
	singleUsed := make([]bool, len(pairTurns))
	for pairIndex, pairTurn := range pairTurns {
		fingerprintKey := agentFingerprintKey(pairTurn.agent, pairTurn.fingerprint)
		// One completed round is enough to preserve a new thread, even when the
		// operator's only message is short. Once multiple turns exist, retain the
		// stronger single-turn threshold and let the paired-turn matcher below
		// establish short exchanges without weakening ambiguity rejection.
		if (len(pairTurns) != 1 && (len([]byte(pairTurn.text)) < 32 || pairTurn.words < 5)) || pairSingles[fingerprintKey] != 1 || nativeSingles[fingerprintKey] != 1 {
			continue
		}
		for _, root := range roots {
			for _, nativeTurn := range nativeTurns[root] {
				if nativeTurn.fingerprint == pairTurn.fingerprint && agentsCompatible(pairTurn.agent, nativeTurn.agent) && nativeTurn.completed {
					observations = append(observations, observation(root, []normalizedTurn{pairTurn}, []normalizedTurn{nativeTurn}))
					singleUsed[pairIndex] = true
				}
			}
		}
	}

	pairPairs := countTurnPairs(pairTurns)
	nativePairs := map[string]int{}
	for _, root := range roots {
		for key, count := range countTurnPairs(nativeTurns[root]) {
			nativePairs[key] += count
		}
	}
	for pairIndex := 0; pairIndex+1 < len(pairTurns); pairIndex++ {
		if singleUsed[pairIndex] || singleUsed[pairIndex+1] {
			continue
		}
		first, second := pairTurns[pairIndex], pairTurns[pairIndex+1]
		if first.owner != second.owner {
			continue
		}
		if len([]byte(first.text)) < 8 || len([]byte(second.text)) < 8 || (first.words < 3 && second.words < 3) {
			continue
		}
		key := turnPairKey(first, second)
		if pairPairs[key] != 1 || nativePairs[key] != 1 {
			continue
		}
		for _, root := range roots {
			turns := nativeTurns[root]
			for nativeIndex := 0; nativeIndex+1 < len(turns); nativeIndex++ {
				left, right := turns[nativeIndex], turns[nativeIndex+1]
				if turnPairKey(left, right) == key && agentsCompatible(first.agent, left.agent) && left.completed && right.completed {
					observations = append(observations, observation(root, []normalizedTurn{first, second}, []normalizedTurn{left, right}))
				}
			}
		}
	}
	slices.SortFunc(observations, compareRoundObservation)
	return observations
}

func makeNormalizedTurn(position uint64, text string) normalizedTurn {
	sum := sha256.Sum256([]byte(text))
	return normalizedTurn{position: position, text: text, fingerprint: fmt.Sprintf("%x", sum), words: unicodeWordCount(text)}
}

func unicodeWordCount(text string) int {
	words, inWord := 0, false
	for _, r := range text {
		wordRune := unicode.IsLetter(r) || unicode.IsNumber(r)
		if wordRune && !inWord {
			words++
		}
		inWord = wordRune
	}
	return words
}

func countTurnFingerprints(turns []normalizedTurn) map[string]int {
	counts := map[string]int{}
	for _, turn := range turns {
		counts[agentFingerprintKey(turn.agent, turn.fingerprint)]++
	}
	return counts
}

func uniqueTurnPositions(turns []normalizedTurn) bool {
	for i := 1; i < len(turns); i++ {
		if turns[i-1].owner == turns[i].owner && turns[i-1].position == turns[i].position {
			return false
		}
	}
	return true
}

func countTurnPairs(turns []normalizedTurn) map[string]int {
	counts := map[string]int{}
	for i := 0; i+1 < len(turns); i++ {
		if turns[i].owner != turns[i+1].owner && turns[i].owner != "" {
			continue
		}
		counts[turnPairKey(turns[i], turns[i+1])]++
	}
	return counts
}

func turnPairKey(first, second normalizedTurn) string {
	return string(first.agent) + "\x00" + first.fingerprint + "\x00" + second.fingerprint
}

func observation(root string, pair, native []normalizedTurn) RoundObservation {
	result := RoundObservation{RootNodeID: root}
	if len(pair) != 0 {
		result.ScopeKey, result.Tag, result.Agent = pair[0].scopeKey, pair[0].tag, pair[0].agent
	}
	for _, turn := range pair {
		result.PairPositions = append(result.PairPositions, turn.position)
		result.Fingerprints = append(result.Fingerprints, turn.fingerprint)
		result.Texts = append(result.Texts, turn.text)
	}
	for _, turn := range native {
		result.NativePositions = append(result.NativePositions, turn.position)
		result.ProgressPositions = append(result.ProgressPositions, turn.progressPosition)
	}
	return result
}

func pairOwnerKey(scopeKey, tag string, agent Agent) string {
	return scopeKey + "\x00" + tag + "\x00" + string(agent)
}

func agentFingerprintKey(agent Agent, fingerprint string) string {
	return string(agent) + "\x00" + fingerprint
}

func agentsCompatible(left, right Agent) bool { return left == "" || right == "" || left == right }

func compareRoundObservation(a, b RoundObservation) int {
	for _, values := range [][2]string{{a.ScopeKey, b.ScopeKey}, {a.Tag, b.Tag}, {string(a.Agent), string(b.Agent)}} {
		if result := cmp.Compare(values[0], values[1]); result != 0 {
			return result
		}
	}
	if result := compareUint64Slices(a.PairPositions, b.PairPositions); result != 0 {
		return result
	}
	if result := cmp.Compare(a.RootNodeID, b.RootNodeID); result != 0 {
		return result
	}
	return compareUint64Slices(a.NativePositions, b.NativePositions)
}

func compareUint64Slices(a, b []uint64) int {
	for i := 0; i < min(len(a), len(b)); i++ {
		if result := cmp.Compare(a[i], b[i]); result != 0 {
			return result
		}
	}
	return cmp.Compare(len(a), len(b))
}
