package broadcast

import (
	"context"
	"encoding/json"
	"io"
	"mime"
	"net/http"
	"strings"
	"time"

	"github.com/xianxu/pair/cmd/internal/castlock"
)

// The lock gate (pair#495). A link whose mount answers LockFor with a lock is
// served to sessions only: before one, the Host answers its own lock page,
// that page's two assets and POST unlock, and refuses everything else with 401
// -- event streams, the viewer's assets, every POST route. The mount never
// sees a request without a session, so no route of it can forget the check.
//
// The session rides a cookie scoped to the link's path: HttpOnly, so the
// page's script never holds it; SameSite=Strict; no expiry, so it ends with
// the browser. The browser sends it on the page's own fetches, its event
// stream and its images and fonts alike, which a bearer kept in script could
// not reach without putting it in a URL.

const (
	sessionCookie = "couch_session"
	// maxUnlockBody bounds the unlock request: a code and nothing else.
	maxUnlockBody = 256
)

// lockAssets is everything a locked link serves before a session.
var lockAssets = map[string]asset{
	"lock.js": {"web/lock.js", "text/javascript; charset=utf-8"},
	// The module lock.js imports (the shared name rule); pure code, no data.
	"collab.js": {"web/collab.js", "text/javascript; charset=utf-8"},
	"lock.css":  {"web/lock.css", "text/css; charset=utf-8"},
}

// Refusals the lock page shows; fixed text that never echoes the request.
const (
	refuseWrongCode = "That code isn't right."
	refuseMalformed = "Enter the code you were given."
	refuseSlowDown  = "Too many tries. Wait a moment."
	refuseEnded     = "This cast has ended."
	refuseTryAgain  = "Something went wrong. Try again."
)

// gate serves a request for a locked link. GET session is the gate's own: 204
// for a live session, 401 otherwise, so a page whose stream closed can tell a
// rotated code from a lost network. It returns the context the owner
// should serve under -- cancelled when the session is revoked -- or nil when
// the gate answered the request itself.
func gate(w http.ResponseWriter, r *http.Request, segment, rest string, l *castlock.Lock, secure bool) (*http.Request, func()) {
	if r.Method == http.MethodPost && rest == "unlock" {
		// The gate's own route, with or without a session: entering the code
		// again issues a fresh one.
		SetSecurityHeaders(w.Header(), contentSecurityPolicy)
		unlock(w, r, segment, l, secure)
		return nil, nil
	}
	if c, err := r.Cookie(sessionCookie); err == nil {
		if revoked, ok := l.Valid(c.Value); ok {
			owner := sessionNoteOwner(c.Value)
			if r.Method == http.MethodGet && rest == "session" {
				// The page's probe after a lost stream: the session holds.
				SetSecurityHeaders(w.Header(), contentSecurityPolicy)
				w.Header().Set(NoteOwnerHeader, owner)
				w.WriteHeader(http.StatusNoContent)
				return nil, nil
			}
			ctx, cancel := context.WithCancel(context.WithValue(r.Context(), noteOwnerKey{}, noteOwnerProof{owner: owner, revoked: revoked}))
			go func() {
				select {
				case <-revoked:
					cancel()
				case <-ctx.Done():
				}
			}()
			return r.WithContext(ctx), cancel
		}
	}
	SetSecurityHeaders(w.Header(), contentSecurityPolicy)
	switch {
	case r.Method == http.MethodGet && rest == "":
		serveAsset(w, asset{"web/lock.html", "text/html; charset=utf-8"})
	case r.Method == http.MethodGet && lockAssets[rest] != (asset{}):
		serveAsset(w, lockAssets[rest])
	default:
		// No hint of what is behind the lock, and no hint of which paths exist.
		http.Error(w, "locked", http.StatusUnauthorized)
	}
	return nil, nil
}

type unlockAnswer struct {
	OK    bool   `json:"ok"`
	Error string `json:"error,omitempty"`
}

func unlock(w http.ResponseWriter, r *http.Request, segment string, l *castlock.Lock, secure bool) {
	answer := func(status int, a unlockAnswer) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(a)
	}
	if mt, _, err := mime.ParseMediaType(r.Header.Get("Content-Type")); err != nil || mt != "application/json" {
		answer(http.StatusUnsupportedMediaType, unlockAnswer{Error: refuseMalformed})
		return
	}
	body, err := io.ReadAll(io.LimitReader(r.Body, maxUnlockBody+1))
	var req struct {
		Code string `json:"code"`
	}
	if err != nil || len(body) > maxUnlockBody || json.Unmarshal(body, &req) != nil {
		answer(http.StatusBadRequest, unlockAnswer{Error: refuseMalformed})
		return
	}
	session, out := l.Unlock(time.Now(), req.Code)
	switch out {
	case castlock.Unlocked:
		http.SetCookie(w, &http.Cookie{
			Name: sessionCookie, Value: session, Path: "/" + segment + "/",
			HttpOnly: true, Secure: secure, SameSite: http.SameSiteStrictMode,
		})
		answer(http.StatusOK, unlockAnswer{OK: true})
	case castlock.Wrong:
		answer(http.StatusForbidden, unlockAnswer{Error: refuseWrongCode})
	case castlock.Malformed:
		answer(http.StatusBadRequest, unlockAnswer{Error: refuseMalformed})
	case castlock.Limited:
		w.Header().Set("Retry-After", "1")
		answer(http.StatusTooManyRequests, unlockAnswer{Error: refuseSlowDown})
	case castlock.Failed:
		answer(http.StatusInternalServerError, unlockAnswer{Error: refuseTryAgain})
	default:
		answer(http.StatusGone, unlockAnswer{Error: refuseEnded})
	}
}

func serveAsset(w http.ResponseWriter, a asset) {
	body, err := webFS.ReadFile(a.file)
	if err != nil {
		http.Error(w, "unavailable", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", a.contentType)
	w.Write(body)
}

// NewLinkLock makes the lock for one cast link, with the unlock rate every
// link shares: a burst of 5 attempts, then one a second. Over a budget of 8
// that bounds a guesser to about one forced rotation every 8 seconds.
func NewLinkLock(digits int, onRotated func(castlock.Rotation)) (*castlock.Lock, error) {
	// The Lock calls the limiter under its own mutex, as RateLimit needs.
	return castlock.New(castlock.Config{Digits: digits, Limiter: NewRateLimit(1, 5), OnRotated: onRotated})
}

// isSecureBase reports whether a link is served over https, where the session
// cookie is marked Secure.
func isSecureBase(base string) bool { return strings.HasPrefix(base, "https://") }
