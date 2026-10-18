package zellijreply

import "context"

// Effect checks the existing admission token immediately before one synchronous
// effect. Once dispatched, the effect keeps its existing lifetime and result;
// cancellation prevents later effects without rolling this one back. Owed
// cleanup and refusal presentation must remain outside this guard.
func Effect(ctx context.Context, invoke func() error) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	return invoke()
}
