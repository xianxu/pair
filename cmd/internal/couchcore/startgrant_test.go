package couchcore

import (
	"bytes"
	"encoding/base64"
	"errors"
	"io"
	"sync"
	"testing"
	"time"
)

type testStartResolution struct {
	Path string
	Argv []string
}

func cloneTestStartResolution(value testStartResolution) testStartResolution {
	value.Argv = append([]string(nil), value.Argv...)
	return value
}

func TestStartGrantStoreMatchesReferenceLifecycle(t *testing.T) {
	clock := &manualGrantClock{now: time.Unix(100, 0).UTC()}
	store := NewStartGrantStore[testStartResolution](clock, &grantSequenceEntropy{}, cloneTestStartResolution)
	original := testStartResolution{Path: "/repo", Argv: []string{"--model", "opus"}}

	token, err := store.Issue(original)
	if err != nil {
		t.Fatal(err)
	}
	original.Argv[1] = "mutated-after-issue"
	claimed, err := store.Claim(token)
	if err != nil || claimed.Argv[1] != "opus" {
		t.Fatalf("Claim = %+v, %v", claimed, err)
	}
	claimed.Argv[1] = "mutated-after-claim"
	if _, err := store.Claim(token); !errors.Is(err, ErrStartGrantUnavailable) {
		t.Fatalf("replay err = %v, want unavailable", err)
	}
	if err := store.Finish(token); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Claim(token); !errors.Is(err, ErrStartGrantUnavailable) {
		t.Fatalf("finished token err = %v, want unavailable", err)
	}

	restarted := NewStartGrantStore[testStartResolution](clock, &grantSequenceEntropy{}, cloneTestStartResolution)
	if _, err := restarted.Claim(token); !errors.Is(err, ErrStartGrantUnavailable) {
		t.Fatalf("owner restart retained token: %v", err)
	}
}

func TestStartGrantStoreBoundsAndExpiresOnlyUnclaimed(t *testing.T) {
	clock := &manualGrantClock{now: time.Unix(100, 0).UTC()}
	store := NewStartGrantStore[int](clock, &grantSequenceEntropy{}, func(value int) int { return value })
	tokens := make([]StartGrantToken, startGrantCapacity)
	for i := range tokens {
		var err error
		tokens[i], err = store.Issue(i)
		if err != nil {
			t.Fatalf("Issue(%d): %v", i, err)
		}
	}
	if _, err := store.Issue(99); !errors.Is(err, ErrStartGrantCapacity) {
		t.Fatalf("capacity err = %v", err)
	}
	if _, err := store.Claim(tokens[0]); err != nil {
		t.Fatal(err)
	}
	clock.Advance(startGrantTTL)
	replacement, err := store.Issue(100)
	if err != nil {
		t.Fatalf("expired unclaimed grants did not free capacity: %v", err)
	}
	if _, err := store.Claim(tokens[1]); !errors.Is(err, ErrStartGrantUnavailable) {
		t.Fatalf("expired token err = %v", err)
	}
	if err := store.Finish(tokens[0]); err != nil {
		t.Fatalf("consuming token was evicted: %v", err)
	}
	if _, err := store.Claim(replacement); err != nil {
		t.Fatalf("replacement token unavailable: %v", err)
	}

	consuming := NewStartGrantStore[int](clock, &grantSequenceEntropy{}, func(value int) int { return value })
	for i := 0; i < startGrantCapacity; i++ {
		token, issueErr := consuming.Issue(i)
		if issueErr != nil {
			t.Fatal(issueErr)
		}
		if _, claimErr := consuming.Claim(token); claimErr != nil {
			t.Fatal(claimErr)
		}
	}
	clock.Advance(startGrantTTL)
	if _, err := consuming.Issue(101); !errors.Is(err, ErrStartGrantCapacity) {
		t.Fatalf("consuming grants were evicted: %v", err)
	}
}

func TestStartGrantStoreUsesRawURLTokensAndThreeCollisionDraws(t *testing.T) {
	clock := &manualGrantClock{now: time.Unix(100, 0).UTC()}
	raw := bytes.Repeat([]byte{0x5a}, startGrantTokenBytes)
	store := NewStartGrantStore[int](clock, bytes.NewReader(bytes.Repeat(raw, startGrantCollisionAttempts+1)), func(value int) int { return value })
	token, err := store.Issue(1)
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := base64.RawURLEncoding.DecodeString(string(token))
	if err != nil || len(decoded) != startGrantTokenBytes {
		t.Fatalf("token %q decodes to %d bytes, %v", token, len(decoded), err)
	}
	if _, err := store.Issue(2); !errors.Is(err, ErrStartGrantCollision) {
		t.Fatalf("collision err = %v", err)
	}

	short := NewStartGrantStore[int](clock, bytes.NewReader([]byte{1}), func(value int) int { return value })
	if _, err := short.Issue(1); !errors.Is(err, io.ErrUnexpectedEOF) {
		t.Fatalf("short entropy err = %v", err)
	}
}

func TestStartGrantStoreClaimIsAtomic(t *testing.T) {
	store := NewStartGrantStore[int](&manualGrantClock{now: time.Unix(100, 0).UTC()}, &grantSequenceEntropy{}, func(value int) int { return value })
	token, err := store.Issue(7)
	if err != nil {
		t.Fatal(err)
	}

	const contenders = 32
	start := make(chan struct{})
	results := make(chan error, contenders)
	var wg sync.WaitGroup
	for i := 0; i < contenders; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			_, claimErr := store.Claim(token)
			results <- claimErr
		}()
	}
	close(start)
	wg.Wait()
	close(results)
	wins := 0
	for claimErr := range results {
		if claimErr == nil {
			wins++
		} else if !errors.Is(claimErr, ErrStartGrantUnavailable) {
			t.Fatalf("unexpected claim error: %v", claimErr)
		}
	}
	if wins != 1 {
		t.Fatalf("successful claims = %d, want 1", wins)
	}
}

type manualGrantClock struct {
	mu  sync.Mutex
	now time.Time
}

func (c *manualGrantClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *manualGrantClock) Advance(duration time.Duration) {
	c.mu.Lock()
	c.now = c.now.Add(duration)
	c.mu.Unlock()
}

type grantSequenceEntropy struct {
	mu   sync.Mutex
	next byte
}

func (r *grantSequenceEntropy) Read(p []byte) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.next++
	for i := range p {
		p[i] = r.next
	}
	return len(p), nil
}
