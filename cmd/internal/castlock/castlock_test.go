package castlock

import (
	"crypto/rand"
	"errors"
	mrand "math/rand/v2"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func mustNew(t *testing.T, cfg Config) *Lock {
	t.Helper()
	l, err := New(cfg)
	if err != nil {
		t.Fatal(err)
	}
	return l
}

func wrongFor(code string) string {
	b := []byte(code)
	b[0] = '0' + (b[0]-'0'+1)%10
	return string(b)
}

func TestRightCodeUnlocksWrongCodeDoesNot(t *testing.T) {
	l := mustNew(t, Config{})
	if s, out := l.Unlock(time.Now(), wrongFor(l.Code())); out != Wrong || s != "" {
		t.Fatalf("wrong code: %q %v", s, out)
	}
	s, out := l.Unlock(time.Now(), l.Code())
	if out != Unlocked || s == "" {
		t.Fatalf("right code: %q %v", s, out)
	}
	if _, ok := l.Valid(s); !ok {
		t.Fatal("issued session is not valid")
	}
	if _, ok := l.Valid(s + "x"); ok {
		t.Fatal("an altered session is valid")
	}
}

func TestSpacesAndDashesAreIgnoredOtherInputIsMalformed(t *testing.T) {
	l := mustNew(t, Config{Budget: 2})
	c := l.Code()
	spaced := c[:2] + " - " + c[2:]
	if _, out := l.Unlock(time.Now(), spaced); out != Unlocked {
		t.Fatalf("spaced code: %v", out)
	}
	for _, in := range []string{"", "12", "12345", "12a4", "١٢٣٤", strings.Repeat("1", 40)} {
		if _, out := l.Unlock(time.Now(), in); out != Malformed {
			t.Fatalf("%q: %v", in, out)
		}
	}
	// Malformed input costs no budget: the code is unchanged.
	if l.Code() != c {
		t.Fatal("malformed input rotated the code")
	}
}

func TestSpentBudgetRotatesCodeButKeepsSessions(t *testing.T) {
	var rotations []Rotation
	l := mustNew(t, Config{Budget: 3, OnRotated: func(r Rotation) { rotations = append(rotations, r) }})
	viewer, _ := l.Unlock(time.Now(), l.Code())
	first := l.Code()
	for range 3 {
		l.Unlock(time.Now(), wrongFor(l.Code()))
	}
	if l.Code() == first {
		t.Fatal("budget spent but code unchanged")
	}
	if len(rotations) != 1 || rotations[0].Reason != ReasonGuessing || rotations[0].Code != l.Code() {
		t.Fatalf("rotations %+v", rotations)
	}
	if _, ok := l.Valid(viewer); !ok {
		t.Fatal("a guess-rotation evicted a viewer")
	}
	if _, out := l.Unlock(time.Now(), first); out != Wrong {
		t.Fatalf("the spent code still unlocks: %v", out)
	}
}

func TestOperatorRotationRevokesEverySession(t *testing.T) {
	l := mustNew(t, Config{})
	s, _ := l.Unlock(time.Now(), l.Code())
	revoked, _ := l.Valid(s)
	old := l.Code()
	next, err := l.Rotate()
	if err != nil || next == old {
		t.Fatalf("rotate: %q %v", next, err)
	}
	select {
	case <-revoked:
	default:
		t.Fatal("revoked channel open after rotation")
	}
	if _, ok := l.Valid(s); ok {
		t.Fatal("session valid after rotation")
	}
	if l.Sessions() != 0 {
		t.Fatal("sessions survived rotation")
	}
	s2, out := l.Unlock(time.Now(), next)
	if out != Unlocked {
		t.Fatal("new code does not unlock")
	}
	if ch, _ := l.Valid(s2); ch == revoked {
		t.Fatal("a new session shares the revoked generation")
	}
}

func TestEndRefusesEverything(t *testing.T) {
	l := mustNew(t, Config{})
	s, _ := l.Unlock(time.Now(), l.Code())
	l.End()
	l.End()
	if _, ok := l.Valid(s); ok {
		t.Fatal("session valid after End")
	}
	if _, out := l.Unlock(time.Now(), l.Code()); out != Ended {
		t.Fatalf("unlock after End: %v", out)
	}
	if _, err := l.Rotate(); err == nil {
		t.Fatal("rotate after End succeeded")
	}
}

type countLimiter struct{ left int }

func (c *countLimiter) Allow(time.Time) bool { c.left--; return c.left >= 0 }

func TestLimitedAttemptsAreNotJudged(t *testing.T) {
	l := mustNew(t, Config{Budget: 1, Limiter: &countLimiter{left: 0}})
	c := l.Code()
	if _, out := l.Unlock(time.Now(), wrongFor(c)); out != Limited {
		t.Fatalf("over the rate: %v", out)
	}
	if l.Code() != c {
		t.Fatal("a limited attempt spent budget")
	}
}

func TestSessionsAreCappedOldestFirst(t *testing.T) {
	l := mustNew(t, Config{MaxSessions: 2})
	a, _ := l.Unlock(time.Now(), l.Code())
	b, _ := l.Unlock(time.Now(), l.Code())
	c, _ := l.Unlock(time.Now(), l.Code())
	if _, ok := l.Valid(a); ok {
		t.Fatal("oldest session survived the cap")
	}
	for _, s := range []string{b, c} {
		if _, ok := l.Valid(s); !ok {
			t.Fatal("a newer session was evicted")
		}
	}
}

func TestDriveLengthCode(t *testing.T) {
	l := mustNew(t, Config{Digits: 6})
	if len(l.Code()) != 6 {
		t.Fatalf("code %q", l.Code())
	}
	if _, out := l.Unlock(time.Now(), l.Code()[:4]); out != Malformed {
		t.Fatalf("4 digits on a 6-digit lock: %v", out)
	}
}

// TestRandomSequencesKeepInvariants drives two locks (a view and a drive link)
// through seeded random operations and checks, after every step, invariants
// stated independently of the implementation:
//   - a session revoked by rotation or End is never valid again;
//   - a session from one lock never validates on the other;
//   - no code generation absorbs more than budget wrong tries;
//   - live sessions never exceed the cap.
func TestRandomSequencesKeepInvariants(t *testing.T) {
	for seed := range uint64(50) {
		rng := mrand.New(mrand.NewPCG(seed, seed^0x9e37))
		const budget, maxSessions = 4, 5
		locks := [2]*Lock{
			mustNew(t, Config{Budget: budget, MaxSessions: maxSessions}),
			mustNew(t, Config{Digits: 6, Budget: budget, MaxSessions: maxSessions}),
		}
		var issued [2][]string
		var dead [2]map[string]bool
		dead[0], dead[1] = map[string]bool{}, map[string]bool{}
		var wrongThisCode, forced [2]int
		var codeSeen [2]string
		ended := [2]bool{}
		for i := range 2 {
			codeSeen[i] = locks[i].Code()
		}
		for step := range 400 {
			i := rng.IntN(2)
			l := locks[i]
			switch op := rng.IntN(10); {
			case op < 4:
				if s, out := l.Unlock(time.Now(), l.Code()); out == Unlocked {
					issued[i] = append(issued[i], s)
				}
			case op < 8:
				l.Unlock(time.Now(), wrongFor(l.Code()))
				if !ended[i] && forced[i] < MaxForcedRotations {
					wrongThisCode[i]++
				}
			case op < 9:
				if _, err := l.Rotate(); err == nil {
					forced[i] = -1 // the code-change observation below starts a new lifetime
					for _, s := range issued[i] {
						dead[i][s] = true
					}
				}
			default:
				if rng.IntN(20) == 0 {
					l.End()
					ended[i] = true
					for _, s := range issued[i] {
						dead[i][s] = true
					}
				}
			}
			for j := range 2 {
				if c := locks[j].Code(); c != codeSeen[j] {
					codeSeen[j], wrongThisCode[j] = c, 0
					forced[j]++
				}
				if wrongThisCode[j] >= budget && !ended[j] {
					t.Fatalf("seed %d step %d: lock %d absorbed %d wrong tries on one code", seed, step, j, wrongThisCode[j])
				}
				if n := locks[j].Sessions(); n > maxSessions {
					t.Fatalf("seed %d step %d: %d sessions over cap", seed, step, n)
				}
				for _, s := range issued[j] {
					if _, ok := locks[j].Valid(s); ok && dead[j][s] {
						t.Fatalf("seed %d step %d: revoked session valid", seed, step)
					}
					if _, ok := locks[1-j].Valid(s); ok {
						t.Fatalf("seed %d step %d: a session crossed locks", seed, step)
					}
				}
			}
		}
	}
}

func FuzzParseCode(f *testing.F) {
	for _, s := range []string{"1234", "12 34", "12-34", "", "abcd", "١٢٣٤", "123456"} {
		f.Add(s, 4)
	}
	f.Fuzz(func(t *testing.T, in string, digits int) {
		if digits < 1 || digits > 9 {
			return
		}
		code, ok := ParseCode(in, digits)
		if !ok {
			return
		}
		if len(code) != digits || strings.Trim(code, "0123456789") != "" {
			t.Fatalf("%q → %q", in, code)
		}
	})
}

func FuzzUnlock(f *testing.F) {
	for _, s := range []string{"1234", "", "0000", "x", strings.Repeat("9", 64)} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, in string) {
		l, err := New(Config{Budget: 3})
		if err != nil {
			t.Fatal(err)
		}
		code := l.Code()
		s, out := l.Unlock(time.Now(), in)
		parsed, ok := ParseCode(in, 4)
		switch {
		case out == Unlocked && (!ok || parsed != code):
			t.Fatalf("%q unlocked code %q", in, code)
		case out == Unlocked && s == "":
			t.Fatal("unlocked without a session")
		case out != Unlocked && s != "":
			t.Fatal("session without unlock")
		}
	})
}

// BR-4: callbacks run outside the mutex, so concurrent rotations may arrive in
// any order; the one with the highest Gen always carries the current code.
func TestConcurrentRotationsNewestGenIsCurrentCode(t *testing.T) {
	for range 20 {
		var mu sync.Mutex
		var newest Rotation
		l := mustNew(t, Config{OnRotated: func(r Rotation) {
			mu.Lock()
			defer mu.Unlock()
			if r.Gen > newest.Gen {
				newest = r
			}
		}})
		var wg sync.WaitGroup
		for range 8 {
			wg.Add(1)
			go func() { defer wg.Done(); _, _ = l.Rotate() }()
		}
		wg.Wait()
		if newest.Gen != 8 || newest.Code != l.Code() {
			t.Fatalf("newest %+v, current %q", newest, l.Code())
		}
	}
}

// failingRand serves crypto randomness until fail is set.
type failingRand struct {
	fail  atomic.Bool
	reads atomic.Int64
}

func (f *failingRand) Read(p []byte) (int, error) {
	f.reads.Add(1)
	if f.fail.Load() {
		return 0, errors.New("no entropy")
	}
	return rand.Read(p)
}

// A spent code that cannot rotate refuses every attempt -- the right code
// too -- until the operator rotates it: it never keeps absorbing guesses.
func TestSpentCodeThatCannotRotateFailsClosed(t *testing.T) {
	src := &failingRand{}
	l := mustNew(t, Config{Budget: 2, Rand: src})
	code := l.Code()
	src.fail.Store(true)
	for range 2 {
		l.Unlock(time.Now(), wrongFor(code))
	}
	if l.Code() != code {
		t.Fatal("rotated without randomness")
	}
	if _, out := l.Unlock(time.Now(), code); out != Wrong {
		t.Fatalf("the spent code still unlocks: %v", out)
	}
	src.fail.Store(false)
	next, err := l.Rotate()
	if err != nil {
		t.Fatal(err)
	}
	if _, out := l.Unlock(time.Now(), next); out != Unlocked {
		t.Fatalf("after the operator's rotation: %v", out)
	}
}

func TestSessionMintFailureIsNotEnded(t *testing.T) {
	src := &failingRand{}
	l := mustNew(t, Config{Rand: src})
	src.fail.Store(true)
	if _, out := l.Unlock(time.Now(), l.Code()); out != Failed {
		t.Fatalf("mint failure: %v", out)
	}
}

func TestLifetimeGuessCapSurvivesUnlocksAndNeedsOperatorRotation(t *testing.T) {
	for _, digits := range []int{4, 6} {
		t.Run(string(rune('0'+digits)), func(t *testing.T) {
			src := &failingRand{}
			var rotations []Rotation
			l := mustNew(t, Config{Digits: digits, Rand: src, OnRotated: func(r Rotation) { rotations = append(rotations, r) }})
			viewer, out := l.Unlock(time.Now(), l.Code())
			if out != Unlocked {
				t.Fatal(out)
			}
			revoked, _ := l.Valid(viewer)
			for rotation := 1; rotation <= 3; rotation++ {
				before := l.Code()
				for wrong := 1; wrong <= DefaultBudget; wrong++ {
					if _, out := l.Unlock(time.Now(), wrongFor(l.Code())); out != Wrong {
						t.Fatal(out)
					}
					if wrong < DefaultBudget && l.Code() != before {
						t.Fatal("rotated early")
					}
				}
				if l.Code() == before || len(rotations) != rotation || rotations[rotation-1].Capped != (rotation == 3) {
					t.Fatalf("rotation %d: %+v", rotation, rotations)
				}
				_, out := l.Unlock(time.Now(), l.Code())
				if rotation < 3 && out != Unlocked {
					t.Fatalf("capped early at %d: %v", rotation, out)
				}
				if rotation == 3 && out != Wrong {
					t.Fatalf("correct code unlocked at lifetime cap: %v", out)
				}
			}
			for range 20 {
				if s, out := l.Unlock(time.Now(), l.Code()); out != Wrong || s != "" {
					t.Fatalf("cap admitted %q: %v", s, out)
				}
				l.Unlock(time.Now(), wrongFor(l.Code()))
			}
			if len(rotations) != 3 {
				t.Fatal("kept rotating after cap")
			}
			if _, ok := l.Valid(viewer); !ok {
				t.Fatal("cap revoked unlocked viewer")
			}
			select {
			case <-revoked:
				t.Fatal("cap closed session generation")
			default:
			}
			other := mustNew(t, Config{Digits: digits})
			if _, out := other.Unlock(time.Now(), other.Code()); out != Unlocked {
				t.Fatal("cap crossed links")
			}
			src.fail.Store(true)
			if _, err := l.Rotate(); err == nil {
				t.Fatal("operator rotation ignored RNG failure")
			}
			src.fail.Store(false)
			if _, out := l.Unlock(time.Now(), l.Code()); out != Wrong {
				t.Fatal("failed rotation cleared cap")
			}
			if _, ok := l.Valid(viewer); !ok {
				t.Fatal("failed rotation revoked viewer")
			}
			next, err := l.Rotate()
			if err != nil {
				t.Fatal(err)
			}
			if _, out := l.Unlock(time.Now(), next); out != Unlocked {
				t.Fatal("operator rotation did not clear cap")
			}
			if _, ok := l.Valid(viewer); ok {
				t.Fatal("operator rotation kept old viewer")
			}
			for range 2 * DefaultBudget {
				l.Unlock(time.Now(), wrongFor(l.Code()))
			}
			if _, out := l.Unlock(time.Now(), l.Code()); out != Unlocked {
				t.Fatal("recovery did not restore full lifetime budget")
			}
		})
	}
}

func TestThirdForcedRotationEntropyFailureKeepsRefusalAndSessions(t *testing.T) {
	src := &failingRand{}
	l := mustNew(t, Config{Rand: src})
	viewer, _ := l.Unlock(time.Now(), l.Code())
	for range 2 * DefaultBudget {
		l.Unlock(time.Now(), wrongFor(l.Code()))
	}
	last := l.Code()
	src.fail.Store(true)
	for range DefaultBudget {
		l.Unlock(time.Now(), wrongFor(last))
	}
	if l.Code() != last {
		t.Fatal("failed entropy changed code")
	}
	src.fail.Store(false)
	for range 32 {
		if _, out := l.Unlock(time.Now(), last); out != Wrong {
			t.Fatal("spent code unlocked after entropy recovered")
		}
	}
	if _, ok := l.Valid(viewer); !ok {
		t.Fatal("failed forced rotation revoked viewer")
	}
	if _, err := l.Rotate(); err != nil {
		t.Fatal(err)
	}
	if _, out := l.Unlock(time.Now(), l.Code()); out != Unlocked {
		t.Fatal("operator could not recover")
	}
}

func TestCappedAttemptsDoNotDrawMoreRandomness(t *testing.T) {
	for _, attempts := range []int{32, 64} {
		src := &failingRand{}
		l := mustNew(t, Config{Rand: src})
		for range 3 * DefaultBudget {
			l.Unlock(time.Now(), wrongFor(l.Code()))
		}
		reads := src.reads.Load()
		for range attempts {
			if _, out := l.Unlock(time.Now(), l.Code()); out != Wrong {
				t.Fatal(out)
			}
		}
		if got := src.reads.Load(); got != reads {
			t.Fatalf("%d capped attempts added %d entropy reads", attempts, got-reads)
		}
	}
}
