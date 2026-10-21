package remotesession

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"io"
	"mime"
	"net/http"
	"strings"
	"time"
)

// RemoteServer is mounted only on a protected Host. It never delegates an
// unknown path to the ordinary cast, drive, file or RPC handlers.
type RemoteServer struct {
	s        *Supervisor
	verifier *AccessVerifier
	requests chan struct{}
	kills    chan struct{}
}

func newRemoteServer(s *Supervisor, v *AccessVerifier) *RemoteServer {
	// Eight in total, two reserved for kill rather than added to the bound.
	return &RemoteServer{s: s, verifier: v, requests: make(chan struct{}, 6), kills: make(chan struct{}, 2)}
}

func (h *RemoteServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	exchange, cancelExchange := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancelExchange()
	r = r.WithContext(exchange)
	_ = http.NewResponseController(w).SetWriteDeadline(time.Now().Add(5 * time.Second))
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Referrer-Policy", "no-referrer")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Content-Security-Policy", "default-src 'none'; script-src 'self'; connect-src 'self'; base-uri 'none'; frame-ancestors 'none'; form-action 'none'")
	w.Header().Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
	if r.Method == http.MethodPost {
		// A bounded body must also arrive within bounded time. Context alone
		// does not interrupt a blocked socket Read, including the kill lane.
		_ = http.NewResponseController(w).SetReadDeadline(time.Now().Add(5 * time.Second))
	}
	if h.verifier == nil {
		http.Error(w, "remote authentication unavailable", 503)
		return
	}
	// A dedicated, bounded kill lane is independent of document work and the
	// four-session cap. Every lane still requires signature verification.
	kill := r.URL.Path == "/remote/kill"
	lane := h.requests
	if kill {
		lane = h.kills
	}
	select {
	case lane <- struct{}{}:
		defer func() { <-lane }()
	default:
		http.Error(w, "remote request capacity reached", 429)
		return
	}
	if _, err := h.s.policy.requestIP(r); err != nil {
		http.Error(w, "remote origin refused", 403)
		return
	}
	assertions := r.Header.Values("Cf-Access-Jwt-Assertion")
	if len(assertions) != 1 {
		h.revokePresentedSession(r)
		http.Error(w, "Access sign-in required", 401)
		return
	}
	id, err := h.verifier.Verify(r.Context(), assertions[0])
	if err != nil {
		h.revokePresentedSession(r)
		http.Error(w, "Access sign-in required", 401)
		return
	}
	if kill {
		if r.Method != http.MethodPost || !oneHeader(r, "X-Remote-Kill", "1") || !emptyRemoteJSON(r) {
			http.Error(w, "remote kill refused", 403)
			return
		}
		h.s.Kill()
		w.WriteHeader(http.StatusNoContent)
		return
	}
	h.s.gate.mu.Lock()
	e := h.s.epoch
	ready := h.s.gate.state.phase == phaseReady && e != nil
	prefix := ""
	if ready {
		prefix = "/remote/" + e.capability + "/"
	}
	validPath := ready && len(r.URL.Path) >= len(prefix) && subtle.ConstantTimeCompare([]byte(r.URL.Path[:len(prefix)]), []byte(prefix)) == 1
	if !validPath || r.URL.RawPath != "" || r.URL.RawQuery != "" {
		h.s.gate.mu.Unlock()
		http.NotFound(w, r)
		return
	}
	e.requests.Add(1)
	h.s.gate.mu.Unlock()
	defer e.requests.Done()
	ctx, cancel := context.WithDeadline(r.Context(), id.expires)
	stop := context.AfterFunc(e.ctx, cancel)
	defer func() { stop(); cancel() }()
	r = r.WithContext(ctx)
	suffix := strings.TrimPrefix(r.URL.Path, prefix)
	switch {
	case r.Method == http.MethodGet && suffix == "":
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = io.WriteString(w, remoteSecurityPage)
	case r.Method == http.MethodGet && suffix == "app.js":
		w.Header().Set("Content-Type", "text/javascript; charset=utf-8")
		_, _ = io.WriteString(w, remoteSecurityScript)
	case r.Method == http.MethodGet && suffix == "status":
		writeRemoteJSON(w, h.s.Snapshot())
	case r.Method == http.MethodPost && suffix == "handshake":
		if !oneHeader(r, "X-Remote-Handshake", "1") || !emptyRemoteJSON(r) {
			http.Error(w, "remote handshake refused", 403)
			return
		}
		h.handshake(w, r, e, id, prefix)
	case r.Method == http.MethodPost && suffix == "logout":
		if !emptyRemoteJSON(r) {
			http.Error(w, "remote logout refused", 403)
			return
		}
		cookie, err := r.Cookie("pair-remote")
		if err != nil {
			http.Error(w, "remote handshake required", 401)
			return
		}
		if _, err = e.sessions.Admit(cookie.Value, r.Header.Get("X-Remote-CSRF"), id, time.Now(), true); err != nil {
			http.Error(w, "remote handshake required", 401)
			return
		}
		e.sessions.Logout(cookie.Value)
		http.SetCookie(w, remoteCookie("", prefix, time.Unix(1, 0), -1))
		w.WriteHeader(204)
	default:
		http.NotFound(w, r)
	}
}

func (h *RemoteServer) revokePresentedSession(r *http.Request) {
	cookie, err := r.Cookie("pair-remote")
	if err != nil {
		return
	}
	h.s.gate.mu.Lock()
	e := h.s.epoch
	h.s.gate.mu.Unlock()
	if e != nil {
		e.sessions.Logout(cookie.Value)
	}
}

func oneHeader(r *http.Request, name, value string) bool {
	values := r.Header.Values(name)
	return len(values) == 1 && values[0] == value
}
func emptyRemoteJSON(r *http.Request) bool {
	media, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || media != "application/json" {
		return false
	}
	raw, err := io.ReadAll(io.LimitReader(r.Body, 65))
	return err == nil && strings.TrimSpace(string(raw)) == "{}"
}
func writeRemoteJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}
func remoteCookie(value, path string, expires time.Time, maxAge int) *http.Cookie {
	return &http.Cookie{Name: "pair-remote", Value: value, Path: path, Expires: expires, MaxAge: maxAge, Secure: true, HttpOnly: true, SameSite: http.SameSiteStrictMode}
}
func (h *RemoteServer) handshake(w http.ResponseWriter, r *http.Request, e *remoteEpoch, id AccessIdentity, path string) {
	// A refreshed tab can resume its existing cookie/CSRF pair. It does not
	// allocate another of the four sessions or extend that cookie's assertion.
	if cookie, err := r.Cookie("pair-remote"); err == nil {
		csrf := r.Header.Get("X-Remote-CSRF")
		if ref, err := e.sessions.Admit(cookie.Value, csrf, id, time.Now(), true); err == nil {
			writeRemoteJSON(w, struct {
				CSRF    string
				Expires time.Time
			}{csrf, ref.expires})
			return
		}
	}
	creds, err := e.sessions.Handshake(id, time.Now())
	if err != nil {
		status := 401
		if err == errCapacity {
			status = 429
		}
		http.Error(w, err.Error(), status)
		return
	}
	// Kill can race entropy generation. The session table is already revoked
	// on that path, so it can never turn this response into a surviving grant.
	http.SetCookie(w, remoteCookie(creds.Cookie, path, creds.Expires, 0))
	writeRemoteJSON(w, struct {
		CSRF    string
		Expires time.Time
	}{creds.CSRF, creds.Expires})
}

const remoteSecurityPage = `<!doctype html><html lang="en"><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>Pair remote</title><h1>Pair remote</h1><p id="status" role="status">Connecting…</p><p>Document editing is unavailable until the document adapter is connected.</p><button id="kill" type="button">Disable remote access</button><button id="logout" type="button">Log out</button><script src="app.js" defer></script></html>`

// #760 owns the phone editor. This is only the authenticated security surface;
// no test backend, transcript, file path or generic terminal API is exposed.
const remoteSecurityScript = `"use strict";
const statusNode=document.getElementById("status");
let csrf=sessionStorage.getItem(location.pathname+"csrf")||"";
async function post(path,headers){return fetch(path,{method:"POST",credentials:"same-origin",headers:{"Content-Type":"application/json",...headers},body:"{}",redirect:"error"});}
async function connect(){try{const r=await post("handshake",{"X-Remote-Handshake":"1","X-Remote-CSRF":csrf});if(!r.ok)throw Error(r.status===401?"Access sign-in required":"Handshake unavailable; try reconnecting");const session=await r.json();csrf=session.CSRF;sessionStorage.setItem(location.pathname+"csrf",csrf);statusNode.textContent="Remote access is on";}catch(e){statusNode.textContent=e.message;}}
document.getElementById("kill").onclick=async()=>{statusNode.textContent="Disabling…";try{const r=await post("/remote/kill",{"X-Remote-Kill":"1"});if(!r.ok)throw Error();statusNode.textContent="Remote access disabled";}catch{statusNode.textContent="Disable delivery unconfirmed; check the machine";}};
document.getElementById("logout").onclick=async()=>{try{const r=await post("logout",{"X-Remote-CSRF":csrf});if(!r.ok)throw Error();sessionStorage.removeItem(location.pathname+"csrf");csrf="";statusNode.textContent="Logged out";}catch{statusNode.textContent="Logout unconfirmed; try again";}};
connect();`
