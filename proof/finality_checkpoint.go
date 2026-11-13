package proof

import (
	"context"
	"errors"
	"fmt"
	"sync"

	bt "github.com/ethereum/go-ethereum/beacon/types"
	"github.com/ethereum/go-ethereum/common"
)

func (c *FinalityClient) acquire(ctx context.Context, old *finalityCache, now uint64) (*finalityCache, error) {
	candidates, err := c.checkpointCandidates(ctx)
	if err != nil {
		return nil, err
	}
	candidate := candidates[0]
	roots := map[uint64]common.Hash{}
	for _, cp := range candidates {
		epoch := uint64(cp.Epoch)
		if cp.Root == (common.Hash{}) || epoch > now/32 {
			return nil, errors.New("invalid checkpoint epoch or root")
		}
		if root, ok := roots[epoch]; ok && root != cp.Root {
			return nil, ErrFinalityConflict
		}
		roots[epoch] = cp.Root
		if cp.Epoch < candidate.Epoch {
			candidate = cp
		}
	}
	header, err := c.confirmCheckpoint(ctx, candidate.Root)
	if err != nil {
		return nil, err
	}
	if header.Slot/32 > uint64(candidate.Epoch) {
		return nil, errors.New("checkpoint header is after checkpoint epoch")
	}
	if !freshFinalitySlot(header.Slot, now) {
		return nil, ErrFinalityStale
	}
	if old != nil {
		previous := old.State.Finalized.Beacon
		if header.Slot < previous.Slot || (header.Slot == previous.Slot && header.Hash() != previous.Hash()) {
			return nil, ErrFinalityConflict
		}
		if _, err = c.confirmCheckpoint(ctx, previous.Hash()); err != nil {
			return nil, fmt.Errorf("old finalized history could not be confirmed: %w", err)
		}
	}
	b, err := c.beacon.Bootstrap(ctx, candidate.Root)
	if err != nil {
		return nil, err
	}
	state, err := bootstrapFinality(b, candidate.Root, now, c.network)
	if err != nil {
		return nil, err
	}
	event := FinalityTrustEvent{Root: candidate.Root, Epoch: uint64(candidate.Epoch), Slot: header.Slot, AcceptedAt: c.now().UTC()}
	for _, source := range c.checkpoints {
		event.Sources = append(event.Sources, source.Name())
	}
	cache := &finalityCache{Version: 1, Fingerprint: c.fingerprint, State: *state, InitialTrust: event, LastTrust: event}
	if old != nil {
		cache.InitialTrust = old.InitialTrust
		cache.LastTrust.PreviousRoot = old.State.Finalized.Beacon.Hash()
	}
	return cache, nil
}

func (c *FinalityClient) checkpointCandidates(ctx context.Context) ([]FinalityCheckpoint, error) {
	out := make([]FinalityCheckpoint, len(c.checkpoints))
	errs := make([]error, len(c.checkpoints))
	var wg sync.WaitGroup
	for i, source := range c.checkpoints {
		wg.Go(func() {
			timestamp, root, err := source.Genesis(ctx)
			if err != nil {
				errs[i] = err
				return
			}
			if timestamp != c.network.GenesisTime || root != c.network.GenesisValidatorsRoot {
				errs[i] = errors.New("checkpoint provider network mismatch")
				return
			}
			out[i], errs[i] = source.FinalizedCheckpoint(ctx)
		})
	}
	wg.Wait()
	for i, err := range errs {
		if err != nil {
			return nil, fmt.Errorf("checkpoint source %d: %w", i, err)
		}
	}
	return out, nil
}

func (c *FinalityClient) confirmCheckpoint(ctx context.Context, root common.Hash) (bt.Header, error) {
	out := make([]CheckpointHeader, len(c.checkpoints))
	errs := make([]error, len(c.checkpoints))
	var wg sync.WaitGroup
	for i, source := range c.checkpoints {
		wg.Go(func() { out[i], errs[i] = source.CheckpointHeader(ctx, root) })
	}
	wg.Wait()
	for i, h := range out {
		if errs[i] != nil {
			return bt.Header{}, fmt.Errorf("checkpoint source %d: %w", i, errs[i])
		}
		if !h.Canonical || !h.Finalized || h.ExecutionOptimistic || h.Header.Hash() != root {
			return bt.Header{}, fmt.Errorf("%w: checkpoint source %d does not confirm root", ErrFinalityConflict, i)
		}
		if i > 0 && h.Header != out[0].Header {
			return bt.Header{}, ErrFinalityConflict
		}
	}
	return out[0].Header, nil
}
