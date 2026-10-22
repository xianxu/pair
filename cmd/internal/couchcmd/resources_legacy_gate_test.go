package couchcmd

import (
	"context"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/procutil"
)

func TestLegacySweepCarriesSameCollectionProof(t *testing.T) {
	reader := &samplerReader{rows: []procutil.Process{{PID: 53, PPID: 1, Identity: "root", Name: "zellij"}}}
	sampler := testResourceSampler(reader)
	sampler.sweeps = make(chan resourceRequest)
	sampler.policy.Cadence = time.Hour
	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan struct{})
	go func() { defer close(done); sampler.run(ctx) }()
	t.Cleanup(func() { cancel(); <-done })
	swept, err := sampler.sweep(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if !swept.proof.TableKnown || !swept.proof.InventoryComplete || len(swept.proof.Table) != 1 || swept.proof.Table[0].PID != 53 {
		t.Fatalf("fresh sweep omitted current proof: %+v", swept.proof)
	}
	// The initial pass plus this request each take one census, without an extra
	// observer whose collection could disagree with the legacy classification.
	if reader.tables != 2 {
		t.Fatalf("table collections = %d, want 2", reader.tables)
	}
}
