package couchcore

import (
	"context"
	"errors"
	"time"
)

// FleetInventoryTimeout bounds one `sdlc fleet inventory` run. sdlc's own
// claim reads take up to 15 s each with at most 8 in flight (pair#367).
const FleetInventoryTimeout = 90 * time.Second

// FleetInventorySource yields one fleet's raw `sdlc fleet inventory --json`
// document, read from a vantage checkout inside that fleet. The bytes are
// untrusted; DecodeFleetInventory is their only reader.
type FleetInventorySource interface {
	FleetInventory(ctx context.Context, vantage string) ([]byte, error)
}

// SDLCFleetSource runs sdlc through the shared ProvisionIO seam: its own
// process group, a timeout and the 1 MiB stdout cap.
type SDLCFleetSource struct {
	IO      ProvisionIO
	Timeout time.Duration
}

func (s SDLCFleetSource) FleetInventory(ctx context.Context, vantage string) ([]byte, error) {
	if s.IO == nil {
		return nil, errors.New("sdlc fleet source is unavailable")
	}
	timeout := s.Timeout
	if timeout <= 0 {
		timeout = FleetInventoryTimeout
	}
	return s.IO.Run(ctx, ProvisionCommand{
		Dir: vantage, Program: "sdlc",
		Args:    []string{"fleet", "inventory", "--json", "--path", vantage},
		Timeout: timeout,
	})
}
