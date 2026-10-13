package broadcast

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/castlock"
)

// FuzzUnlockBody drives the gate's unlock decoder with arbitrary bodies: it
// never panics, issues a session only for a well-formed body carrying the
// current code, and never sets a cookie on a refusal.
func FuzzUnlockBody(f *testing.F) {
	for _, s := range []string{`{"code":"1234"}`, `{}`, `{"code":1234}`, `[]`, `{"code":"12 34"}`, `{"code":"` + strings.Repeat("9", 300) + `"}`, "\x00", `{"code":"1234","extra":true}`} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, body string) {
		l, err := castlock.New(castlock.Config{Budget: 1 << 20})
		if err != nil {
			t.Fatal(err)
		}
		req := httptest.NewRequest(http.MethodPost, "/tok/unlock", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		unlock(w, req, "tok", l, false)
		var parsed struct {
			Code string `json:"code"`
		}
		wellFormed := len(body) <= maxUnlockBody && json.Unmarshal([]byte(body), &parsed) == nil
		code, ok := castlock.ParseCode(parsed.Code, castlock.DefaultDigits)
		shouldUnlock := wellFormed && ok && code == l.Code()
		cookie := w.Result().Header.Get("Set-Cookie") != ""
		switch {
		case w.Code == http.StatusOK && !shouldUnlock:
			t.Fatalf("%q unlocked", body)
		case w.Code == http.StatusOK && !cookie:
			t.Fatal("unlocked without a session cookie")
		case w.Code != http.StatusOK && cookie:
			t.Fatalf("refusal %d set a cookie", w.Code)
		}
	})
}
