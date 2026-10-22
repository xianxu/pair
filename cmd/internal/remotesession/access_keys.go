package remotesession

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"sync"
	"time"

	jose "github.com/go-jose/go-jose/v4"
)

const accessKeyTTL = 5 * time.Minute
const accessRefreshInterval = 5 * time.Second

// accessKeys is the single cache owner and oidc.KeySet implementation. The JOSE
// library parses JWS/JWK and verifies signatures; this owner only pins and bounds
// retrieval, cache lifetime and refresh concurrency.
type accessKeys struct {
	mu                 sync.Mutex
	url                string
	client             *http.Client
	now                func() time.Time
	keys               []jose.JSONWebKey
	fetched, attempted time.Time
	fetching           chan struct{}
}

var errAccessKeys = errors.New("Access keys unavailable")

func (s *accessKeys) VerifySignature(ctx context.Context, raw string) ([]byte, error) {
	signed, err := jose.ParseSigned(raw, []jose.SignatureAlgorithm{jose.RS256})
	if err != nil || len(signed.Signatures) != 1 {
		return nil, errAccessKeys
	}
	kid := signed.Signatures[0].Header.KeyID
	if kid == "" || len(kid) > 256 {
		return nil, errAccessKeys
	}
	keys, err := s.lookup(ctx, kid)
	if err != nil {
		return nil, err
	}
	for _, key := range keys {
		if key.KeyID == kid {
			if payload, err := signed.Verify(key.Key); err == nil {
				return payload, nil
			}
		}
	}
	return nil, errAccessKeys
}
func (s *accessKeys) lookup(ctx context.Context, kid string) ([]jose.JSONWebKey, error) {
	for {
		if ctx.Err() != nil {
			return nil, errAccessKeys
		}
		s.mu.Lock()
		now := s.now()
		if now.Before(s.fetched.Add(accessKeyTTL)) {
			for _, key := range s.keys {
				if key.KeyID == kid {
					keys := s.keys
					s.mu.Unlock()
					return keys, nil
				}
			}
		}
		if pending := s.fetching; pending != nil {
			s.mu.Unlock()
			select {
			case <-ctx.Done():
				return nil, errAccessKeys
			case <-pending:
				continue
			}
		}
		if !s.attempted.IsZero() && now.Before(s.attempted.Add(accessRefreshInterval)) {
			s.mu.Unlock()
			return nil, errAccessKeys
		}
		pending := make(chan struct{})
		s.fetching = pending
		s.attempted = now
		s.mu.Unlock()
		keys, err := s.fetch(ctx)
		s.mu.Lock()
		if err == nil {
			s.keys = keys
			s.fetched = s.now()
		}
		s.fetching = nil
		close(pending)
		s.mu.Unlock()
		if err != nil {
			return nil, errAccessKeys
		}
	}
}
func (s *accessKeys) fetch(ctx context.Context) ([]jose.JSONWebKey, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.url, nil)
	if err != nil {
		return nil, errAccessKeys
	}
	response, err := s.client.Do(req)
	if err != nil {
		return nil, errAccessKeys
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil, errAccessKeys
	}
	data, err := io.ReadAll(io.LimitReader(response.Body, 64*1024+1))
	if err != nil || len(data) > 64*1024 {
		return nil, errAccessKeys
	}
	var set jose.JSONWebKeySet
	if json.Unmarshal(data, &set) != nil || len(set.Keys) == 0 || len(set.Keys) > 8 {
		return nil, errAccessKeys
	}
	seen := map[string]bool{}
	for _, key := range set.Keys {
		if !key.Valid() || !key.IsPublic() || key.KeyID == "" || len(key.KeyID) > 256 || seen[key.KeyID] || key.Algorithm != "RS256" || (key.Use != "" && key.Use != "sig") {
			return nil, errAccessKeys
		}
		seen[key.KeyID] = true
	}
	return set.Keys, nil
}
