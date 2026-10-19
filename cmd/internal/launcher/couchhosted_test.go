package launcher

import "testing"

func TestCouchHostedIsEitherThreadVar(t *testing.T) {
	for _, tc := range []struct {
		scope, tag string
		want       bool
	}{{"", "", false}, {"s", "", true}, {"", "t", true}, {"s", "t", true}} {
		env := map[string]string{"COUCH_THREAD_SCOPE": tc.scope, "COUCH_THREAD_TAG": tc.tag}
		if got := CouchHostedEnv(func(k string) string { return env[k] }); got != tc.want {
			t.Errorf("CouchHostedEnv %+v = %v", tc, got)
		}
		if got := (Env{CouchThreadScope: tc.scope, CouchThreadTag: tc.tag}).CouchHosted(); got != tc.want {
			t.Errorf("Env.CouchHosted %+v = %v", tc, got)
		}
	}
}

// The one truth table for #650's slug rule (#650 review BR-1/BR-4): hosting is
// scope OR tag; the re-enable is the exact value "on", so anything else —
// "ON", "1", "0", "off" — fails closed.
func TestSlugEnabledTruthTable(t *testing.T) {
	for _, tc := range []struct {
		scope, tag, slug string
		want             bool
	}{
		{"", "", "", true},
		{"", "", "on", true},
		{"", "", "1", true},
		{"s", "", "", false},
		{"", "t", "", false},
		{"s", "t", "", false},
		{"s", "", "on", true},
		{"", "t", "on", true},
		{"s", "", "ON", false},
		{"s", "", "1", false},
		{"s", "", "0", false},
		{"s", "", "off", false},
	} {
		env := map[string]string{"COUCH_THREAD_SCOPE": tc.scope, "COUCH_THREAD_TAG": tc.tag, "PAIR_SLUG": tc.slug}
		if got := SlugEnabled(func(k string) string { return env[k] }); got != tc.want {
			t.Errorf("SlugEnabled %+v = %v", tc, got)
		}
	}
}
