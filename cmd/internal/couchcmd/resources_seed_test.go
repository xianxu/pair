package couchcmd

import (
	"context"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/couchtty"
)

// Exercise run's real initial WithSessionOwnerPass/inventory/refresh path while
// controlling only the existing artifact pass boundary.
type resourceSeedArtifacts struct {
	*couchcore.FakeThreadArtifactCollisionChecker
	entered chan struct{}
	release <-chan struct{}
	fail    bool
}

func (a *resourceSeedArtifacts) WithSessionOwnerPass(ctx context.Context) (context.Context, context.CancelFunc) {
	close(a.entered)
	select {
	case <-ctx.Done():
		return ctx, func() {}
	case <-a.release:
	}
	if a.fail {
		failed, cancel := context.WithCancel(ctx)
		cancel()
		return failed, func() {}
	}
	return ctx, func() {}
}

func TestResourceInventorySeedOrdersInitialSamplerCollection(t *testing.T) {
	for _, failed := range []bool{false, true} {
		name := "healthy"
		if failed {
			name = "failed"
		}
		t.Run(name, func(t *testing.T) {
			runtime := newRT(t)
			c, err := runtime.NewCouch()
			if err != nil {
				t.Fatal(err)
			}
			release := make(chan struct{})
			artifacts := &resourceSeedArtifacts{FakeThreadArtifactCollisionChecker: couchcore.NewFakeThreadArtifactCollisionChecker(), entered: make(chan struct{}), release: release, fail: failed}
			c.Artifacts = artifacts
			inventory := newResourceInventory(&couchtty.Console{}, c, &messageService{})
			tableRead := make(chan struct{}, 1)
			reader := &samplerReader{tableHook: func(context.Context) {
				select {
				case tableRead <- struct{}{}:
				default:
				}
			}}
			sampler := testResourceSampler(reader)
			sampler.slots = inventory.snapshot
			sampler.sweeps = make(chan resourceRequest)
			sampler.policy.Cadence = time.Hour
			ctx, cancel := context.WithCancel(t.Context())
			inventoryDone, samplerDone := make(chan struct{}), make(chan struct{})
			go func() { defer close(inventoryDone); inventory.run(ctx) }()
			go func() { defer close(samplerDone); inventory.runSampler(ctx, sampler) }()
			t.Cleanup(func() { cancel(); <-inventoryDone; <-samplerDone })
			select {
			case <-artifacts.entered:
			case <-time.After(time.Second):
				t.Fatal("seed did not begin")
			}
			select {
			case <-inventory.seedDone:
				t.Fatal("seed completed while its pass was blocked")
			default:
			}
			select {
			case <-tableRead:
				t.Fatal("sampler read before seed completed")
			case <-time.After(20 * time.Millisecond):
			}
			reader.mu.Lock()
			tables := reader.tables
			reader.mu.Unlock()
			if tables != 0 {
				t.Fatalf("sampler observed unseeded inventory with %d table reads", tables)
			}
			close(release)
			select {
			case <-inventory.seedDone:
			case <-time.After(time.Second):
				t.Fatal("seed attempt did not release sampler")
			}
			swept, err := sampler.sweep(ctx)
			if err != nil {
				t.Fatal(err)
			}
			if refused := swept.legacyRefused; refused != failed {
				t.Fatalf("failed seed=%v legacy refused=%v", failed, refused)
			}
			if inventory.snapshot().Complete == failed {
				t.Fatalf("seed outcome did not reach inventory: %+v", inventory.snapshot())
			}
		})
	}
}

func TestResourceInventorySeedCancellationJoinsWithoutSampling(t *testing.T) {
	runtime := newRT(t)
	c, err := runtime.NewCouch()
	if err != nil {
		t.Fatal(err)
	}
	artifacts := &resourceSeedArtifacts{FakeThreadArtifactCollisionChecker: couchcore.NewFakeThreadArtifactCollisionChecker(), entered: make(chan struct{}), release: make(chan struct{})}
	c.Artifacts = artifacts
	inventory := newResourceInventory(&couchtty.Console{}, c, &messageService{})
	reader := &samplerReader{}
	sampler := testResourceSampler(reader)
	sampler.slots = inventory.snapshot
	ctx, cancel := context.WithCancel(t.Context())
	inventoryDone, samplerDone := make(chan struct{}), make(chan struct{})
	go func() { defer close(inventoryDone); inventory.run(ctx) }()
	go func() { defer close(samplerDone); inventory.runSampler(ctx, sampler) }()
	select {
	case <-artifacts.entered:
	case <-time.After(time.Second):
		cancel()
		t.Fatal("seed did not begin")
	}
	cancel()
	for _, done := range []<-chan struct{}{inventoryDone, samplerDone} {
		select {
		case <-done:
		case <-time.After(time.Second):
			t.Fatal("canceled seed or sampler did not join")
		}
	}
	select {
	case <-inventory.seedDone:
	default:
		t.Fatal("canceled seed attempt did not publish completion")
	}
	reader.mu.Lock()
	defer reader.mu.Unlock()
	if reader.tables != 0 {
		t.Fatalf("canceled pre-seed sampler read process table %d times", reader.tables)
	}
}

func TestResourceInventorySeedFailureCannotPromoteKnownConsoleSnapshot(t *testing.T) {
	runtime := newRT(t)
	c, err := runtime.NewCouch()
	if err != nil {
		t.Fatal(err)
	}
	release := make(chan struct{})
	close(release)
	c.Artifacts = &resourceSeedArtifacts{FakeThreadArtifactCollisionChecker: couchcore.NewFakeThreadArtifactCollisionChecker(), entered: make(chan struct{}), release: release}
	// An unavailable store fails discovery while leaving the service context live.
	c.Threads = nil
	inventory := newResourceInventory(&couchtty.Console{}, c, &messageService{})
	inventory.seed(t.Context(), couchtty.ResourceInventorySnapshot{InventoryKnown: true})
	if inventory.snapshot().Complete {
		t.Fatal("failed seed promoted the console's cached known flag")
	}
	sampler := testResourceSampler(&samplerReader{})
	sampler.slots = inventory.snapshot
	sampler.pass(t.Context())
	if !sampler.provenanceIncomplete {
		t.Fatal("failed seed did not refuse legacy provenance")
	}
}
