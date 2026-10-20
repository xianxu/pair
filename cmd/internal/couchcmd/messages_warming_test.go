package couchcmd

import (
	"bytes"
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchmessage"
)

func warmingResponse() couchmessage.Response {
	return couchmessage.Response{Code: couchmessage.CodeWarming, RetryAfterMillis: 500, Error: "slot catalog warming; nothing was queued; retry"}
}

func TestSlotCLIRetriesExplicitWarmingWithSameIntent(t *testing.T) {
	inv := cliInvocation{kind: cliMessage, messageOp: "resume", ref: "pair:1", candidateKey: "chosen", sameBinary: false}
	var intent couchmessage.Request
	got := runSlotScript(t, inv, messageRuntime(), func(n int, r couchmessage.Request) (couchmessage.Response, error) {
		if n == 0 {
			intent = r
		}
		if n < 2 {
			if !reflect.DeepEqual(r, intent) {
				t.Errorf("retry changed intent: %+v, first %+v", r, intent)
			}
			return warmingResponse(), nil
		}
		if n == 2 {
			if !reflect.DeepEqual(r, intent) {
				t.Errorf("admission changed intent: %+v, first %+v", r, intent)
			}
			return receiptResponse("accepted", couchmessage.ReceiptQueued), nil
		}
		if r.Op != "operation-status" || r.ID != intent.ID {
			t.Errorf("post-admission request %+v", r)
		}
		return receiptResponse("ok", couchmessage.ReceiptSucceeded), nil
	})
	if got.code != 0 || len(got.calls) != 4 || !strings.Contains(got.stderr, "waiting for slot catalog to warm") || strings.Contains(got.stderr, "uncertain") {
		t.Fatalf("retry %+v", got)
	}
}

func TestSlotCLIWarmingDoesNotRetryOtherOutcomes(t *testing.T) {
	for _, name := range []string{"zero", "negative", "oversized", "receipt", "refused", "uncertain", "transport"} {
		t.Run(name, func(t *testing.T) {
			got := runSlotScript(t, cliInvocation{kind: cliMessage, messageOp: "resume", ref: "pair:1"}, messageRuntime(), func(int, couchmessage.Request) (couchmessage.Response, error) {
				response := warmingResponse()
				switch name {
				case "zero":
					response.RetryAfterMillis = 0
				case "negative":
					response.RetryAfterMillis = -1
				case "oversized":
					response.RetryAfterMillis = 60000
				case "receipt":
					response.Operation = &couchmessage.OperationReceipt{Status: couchmessage.ReceiptQueued}
				case "refused":
					response.Code = "refused"
				case "uncertain":
					response.Code = "uncertain"
				case "transport":
					return couchmessage.Response{}, errors.New("connection lost after write")
				}
				return response, nil
			})
			if got.code != 1 || len(got.calls) != 1 || got.elapsed != 0 {
				t.Fatalf("unexpected retry %+v", got)
			}
			if (name == "uncertain" || name == "transport") && !strings.Contains(got.stderr, "uncertain") {
				t.Fatalf("lost uncertainty: %+v", got)
			}
		})
	}
}

func TestSlotCLIWarmingRetryBudgetIncludesCallsAndSleep(t *testing.T) {
	for _, hint := range []int{500, 3700, 5000} {
		t.Run((time.Duration(hint) * time.Millisecond).String(), func(t *testing.T) {
			now := time.Unix(5000, 0)
			start := now
			calls := 0
			clock := slotPollClock{now: func() time.Time { return now }, sleep: func(d time.Duration) {
				if d <= 0 || d > 30*time.Second-now.Sub(start) {
					t.Errorf("sleep %v exceeds remaining budget", d)
				}
				now = now.Add(d)
			}}
			call := func(ctx context.Context, _ string, _ any, response any) error {
				calls++
				deadline, ok := ctx.Deadline()
				remaining := 30*time.Second - now.Sub(start)
				duration := time.Until(deadline)
				if !ok || duration <= 0 || duration > min(couchmessage.AdmissionTimeout, remaining) {
					t.Errorf("exchange duration %v exceeds remaining %v", duration, remaining)
				}
				// A successful exchange can consume its full allowance.
				now = now.Add(min(couchmessage.AdmissionTimeout, remaining))
				result := warmingResponse()
				result.RetryAfterMillis = hint
				*response.(*couchmessage.Response) = result
				return nil
			}
			var out, errout bytes.Buffer
			code := runSlotOperationCLI(cliInvocation{kind: cliMessage, messageOp: "resume", ref: "pair:1"}, messageRuntime(), &out, &errout, call, clock)
			if code != 1 || calls < 2 || calls > 8 || now.Sub(start) > 30*time.Second || !strings.Contains(errout.String(), "nothing was queued") || strings.Contains(errout.String(), "uncertain") {
				t.Fatalf("code %d calls %d elapsed %v stderr %q", code, calls, now.Sub(start), errout.String())
			}
		})
	}
}
