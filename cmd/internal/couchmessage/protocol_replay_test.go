package couchmessage

import "testing"

// A replay request (pair#477) takes a recording and nothing else: no
// conversation identity, no message fields.
func TestValidateReplayRequests(t *testing.T) {
	for _, op := range []string{"replay-share", "replay-status"} {
		if err := ValidateRequest(Request{Op: op, Target: "20261012T100000-ab12"}); err != nil {
			t.Errorf("%s: %v", op, err)
		}
		for name, r := range map[string]Request{
			"no recording": {Op: op},
			"identity":     {Op: op, Target: "x", Scope: "s", Tag: "t", Session: "s", Nonce: "n"},
			"id":           {Op: op, Target: "x", ID: "m1"},
			"body":         {Op: op, Target: "x", Body: "hi"},
			"binding":      {Op: op, Target: "x", Binding: &Binding{}},
			"confirmed":    {Op: op, Target: "x", Confirmed: true},
			"too long":     {Op: op, Target: string(make([]byte, MaxBindingBytes+1))},
		} {
			if ValidateRequest(r) == nil {
				t.Errorf("%s %s: accepted", op, name)
			}
		}
	}
}
