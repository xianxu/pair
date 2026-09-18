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
