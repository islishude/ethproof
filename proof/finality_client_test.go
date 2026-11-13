package proof

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	bp "github.com/ethereum/go-ethereum/beacon/params"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

type testFinalityExecution struct {
	headers map[common.Hash]*types.Header
	chain   *big.Int
}

func (s *testFinalityExecution) ChainID(context.Context) (*big.Int, error) { return s.chain, nil }
func (s *testFinalityExecution) HeaderByHash(ctx context.Context, h common.Hash) (*types.Header, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	header := s.headers[h]
	if header == nil {
		return nil, errors.New("unknown execution header")
	}
	return header, nil
}

type testCheckpointSource struct {
	name    string
	net     *bp.ChainConfig
	cp      FinalityCheckpoint
	headers map[common.Hash]CheckpointHeader
	fail    error
}

func (s *testCheckpointSource) Name() string { return s.name }
func (s *testCheckpointSource) Genesis(context.Context) (uint64, common.Hash, error) {
	return s.net.GenesisTime, s.net.GenesisValidatorsRoot, s.fail
}
func (s *testCheckpointSource) FinalizedCheckpoint(context.Context) (FinalityCheckpoint, error) {
	return s.cp, s.fail
}
func (s *testCheckpointSource) CheckpointHeader(_ context.Context, h common.Hash) (CheckpointHeader, error) {
	out, ok := s.headers[h]
	if !ok {
		return out, errors.New("unknown checkpoint")
	}
	return out, s.fail
}

type testBeaconSource struct {
	b            FinalityBootstrap
	updates      []FinalityUpdate
	final        FinalityUpdate
	bootCalls    int
	syncCalls    int
	finalCalls   int
	updatesErr   error
	fail         error
	afterFinal   func()
	afterUpdates func()
}

func (s *testBeaconSource) Bootstrap(context.Context, common.Hash) (FinalityBootstrap, error) {
	s.bootCalls++
	return s.b, s.fail
}
func (s *testBeaconSource) Updates(context.Context, uint64, uint64) ([]FinalityUpdate, error) {
	s.syncCalls++
	if s.afterUpdates != nil {
		s.afterUpdates()
	}
	if s.updatesErr != nil {
		return nil, s.updatesErr
	}
	return s.updates, s.fail
}
func (s *testBeaconSource) FinalityUpdate(context.Context) (FinalityUpdate, error) {
	s.finalCalls++
	if s.afterFinal != nil {
		s.afterFinal()
	}
	return s.final, s.fail
}

func finalityClientFixture(t *testing.T) (*FinalityClient, BlockContext, *testBeaconSource, []*testCheckpointSource) {
	t.Helper()
	net := testFinalityNetwork()
	execution := &testFinalityExecution{headers: map[common.Hash]*types.Header{}, chain: big.NewInt(1)}
	var parent common.Hash
	var target BlockContext
	for n := uint64(100); n <= 103; n++ {
		h := &types.Header{Number: new(big.Int).SetUint64(n), ParentHash: parent, Root: common.Hash{byte(n)}, Difficulty: new(big.Int), GasLimit: 30000000}
		execution.headers[h.Hash()] = h
		parent = h.Hash()
		if n == 100 {
			var err error
			target, err = BlockContextFromHeader(big.NewInt(1), h)
			if err != nil {
				t.Fatal(err)
			}
		}
	}
	committee, _ := finalityCommittee(t, 1)
	b := FinalityBootstrap{Header: finalityTestHeader(t, 128, parent, net), Committee: committee}
	root, branch := finalityTree(map[uint64]common.Hash{54: committee.Root()})
	b.Header.Beacon.StateRoot = root
	b.Branch = branch(54)
	beacon := &testBeaconSource{b: b}
	cps := make([]*testCheckpointSource, 3)
	sources := make([]CheckpointSource, 3)
	for i := range cps {
		cps[i] = &testCheckpointSource{name: fmt.Sprintf("operator-%d", i), net: net, cp: FinalityCheckpoint{Epoch: 4, Root: b.Header.Beacon.Hash()}, headers: map[common.Hash]CheckpointHeader{b.Header.Beacon.Hash(): {Header: b.Header.Beacon, Canonical: true, Finalized: true}}}
		sources[i] = cps[i]
	}
	client, err := newFinalityClientForNetwork(FinalityConfig{Network: "test", CacheDir: t.TempDir()}, execution, FinalityDependencies{Beacon: beacon, Checkpoints: sources, Now: func() time.Time { return time.Unix(int64(net.GenesisTime+200*12), 0) }}, net, 1, 12)
	if err != nil {
		t.Fatal(err)
	}
	return client, target, beacon, cps
}
func TestFinalityClientAncestryCacheAndTamper(t *testing.T) {
	c, target, beacon, _ := finalityClientFixture(t)
	events := 0
	c.onTrustChange = func(FinalityTrustEvent) { events++ }
	anchor, err := c.FinalizedAnchor(t.Context(), target)
	if err != nil {
		t.Fatal(err)
	}
	if anchor.BlockHash != target.BlockHash || beacon.bootCalls != 1 || events != 1 {
		t.Fatal("wrong anchor or bootstrap count")
	}
	if _, err = c.FinalizedAnchor(t.Context(), target); err != nil {
		t.Fatal(err)
	}
	if beacon.bootCalls != 1 || beacon.syncCalls != 0 {
		t.Fatal("cache did not avoid unnecessary sync")
	}
	changed := target
	changed.StateRoot[0] ^= 1
	if _, err = c.FinalizedAnchor(t.Context(), changed); err == nil {
		t.Fatal("accepted tampered root")
	}
	c.config.MaxAncestry = 2
	if _, err = c.FinalizedAnchor(t.Context(), target); !errors.Is(err, ErrFinalityHistory) {
		t.Fatal(err)
	}
}
func TestFinalityCheckpointAllSourcesRequired(t *testing.T) {
	for _, kind := range []string{"unavailable", "conflict", "optimistic", "network", "epoch", "age"} {
		t.Run(kind, func(t *testing.T) {
			c, target, _, sources := finalityClientFixture(t)
			switch kind {
			case "unavailable":
				sources[2].fail = errors.New("offline")
			case "conflict":
				sources[2].cp.Root[0] ^= 1
			case "optimistic":
				h := sources[2].headers[sources[2].cp.Root]
				h.ExecutionOptimistic = true
				sources[2].headers[sources[2].cp.Root] = h
			case "network":
				net := *sources[2].net
				net.GenesisTime++
				sources[2].net = &net
			case "epoch":
				sources[2].cp.Epoch = 999
			case "age":
				c.now = func() time.Time { return time.Unix(int64(c.network.GenesisTime+10000*12), 0) }
			}
			if _, err := c.FinalizedAnchor(t.Context(), target); err == nil {
				t.Fatal("accepted invalid checkpoint")
			}
			if _, err := os.Stat(filepath.Join(c.config.CacheDir, "state.json")); !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("persisted rejected checkpoint: %v", err)
			}
		})
	}
}
func TestFinalityCheckpointDifferentFinalizedHeads(t *testing.T) {
	c, target, _, sources := finalityClientFixture(t)
	sources[2].cp = FinalityCheckpoint{Epoch: 5, Root: common.Hash{22}}
	if _, err := c.FinalizedAnchor(t.Context(), target); err != nil {
		t.Fatal(err)
	}
}
func TestFinalityCacheCorruptionConfigAndCancellation(t *testing.T) {
	c, target, _, _ := finalityClientFixture(t)
	if _, err := c.FinalizedAnchor(t.Context(), target); err != nil {
		t.Fatal(err)
	}
	c.fingerprint[0] ^= 1
	if _, err := c.FinalizedAnchor(t.Context(), target); !errors.Is(err, ErrFinalityCache) {
		t.Fatal(err)
	}
	c.fingerprint[0] ^= 1
	if err := os.WriteFile(filepath.Join(c.config.CacheDir, "state.json"), []byte("broken"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := c.FinalizedAnchor(t.Context(), target); !errors.Is(err, ErrFinalityCache) {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if _, err := c.FinalizedAnchor(ctx, target); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
}
func TestFinalityCacheLockCancellation(t *testing.T) {
	s := NewFinalityFileStore(t.TempDir())
	unlock, err := s.Lock(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := unlock(); err != nil {
			t.Error(err)
		}
	}()
	ctx, cancel := context.WithTimeout(t.Context(), 100*time.Millisecond)
	defer cancel()
	if _, err = s.Lock(ctx); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("second lock: %v", err)
	}
}
func TestFinalityRefreshRequiresOldHistory(t *testing.T) {
	for _, compatible := range []bool{true, false} {
		t.Run(fmt.Sprint(compatible), func(t *testing.T) {
			c, target, beacon, sources := finalityClientFixture(t)
			if _, err := c.FinalizedAnchor(t.Context(), target); err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(c.config.CacheDir, "state.json")
			oldBytes, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			oldRoot := beacon.b.Header.Beacon.Hash()
			beacon.b.Header.Beacon.Slot = 8000
			newRoot := beacon.b.Header.Beacon.Hash()
			for _, s := range sources {
				s.cp = FinalityCheckpoint{Epoch: 250, Root: newRoot}
				s.headers[newRoot] = CheckpointHeader{Header: beacon.b.Header.Beacon, Canonical: true, Finalized: true}
				if !compatible {
					delete(s.headers, oldRoot)
				}
			}
			c.now = func() time.Time { return time.Unix(int64(c.network.GenesisTime+8100*12), 0) }
			_, err = c.FinalizedAnchor(t.Context(), target)
			if compatible {
				if err != nil {
					t.Fatal(err)
				}
				var cache finalityCache
				b, _ := os.ReadFile(path)
				if err = json.Unmarshal(b, &cache); err != nil {
					t.Fatal(err)
				}
				if cache.LastTrust.PreviousRoot != oldRoot || cache.InitialTrust.Root != oldRoot {
					t.Fatal("lost trust history")
				}
			} else {
				if err == nil {
					t.Fatal("accepted conflicting refresh")
				}
				b, _ := os.ReadFile(path)
				if string(b) != string(oldBytes) {
					t.Fatal("overwrote old cache")
				}
			}
		})
	}
}
func TestFinalityBadAncestryAndNotFinalized(t *testing.T) {
	c, _, beacon, _ := finalityClientFixture(t)
	exec := c.execution.(*testFinalityExecution)
	// A valid context on a different branch must not pass merely because it is older.
	fork := &types.Header{Number: big.NewInt(100), Extra: []byte{1}, Difficulty: new(big.Int)}
	exec.headers[fork.Hash()] = fork
	other, err := BlockContextFromHeader(big.NewInt(1), fork)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = c.FinalizedAnchor(t.Context(), other); !errors.Is(err, ErrFinalityConflict) {
		t.Fatal(err)
	}
	newer := &types.Header{Number: big.NewInt(104), Difficulty: new(big.Int)}
	exec.headers[newer.Hash()] = newer
	block, err := BlockContextFromHeader(big.NewInt(1), newer)
	if err != nil {
		t.Fatal(err)
	}
	beacon.final = FinalityUpdate{} // no new authenticated head; never report success.
	if _, err = c.FinalizedAnchor(t.Context(), block); !errors.Is(err, ErrNotFinalized) {
		t.Fatal(err)
	}
}

func TestFinalityClientIncrementalSync(t *testing.T) {
	c, _, beacon, _ := finalityClientFixture(t)
	exec := c.execution.(*testFinalityExecution)
	hash, err := verifyFinalityHeader(beacon.b.Header, c.network)
	if err != nil {
		t.Fatal(err)
	}
	h := &types.Header{Number: big.NewInt(104), ParentHash: hash, Difficulty: new(big.Int)}
	exec.headers[h.Hash()] = h
	target, err := BlockContextFromHeader(big.NewInt(1), h)
	if err != nil {
		t.Fatal(err)
	}
	_, key := finalityCommittee(t, 1)
	next, _ := finalityCommittee(t, 2)
	u := finalityTestUpdate(t, 200, 160, next, key, c.network)
	finalized := finalityTestHeader(t, 160, h.Hash(), c.network)
	u.Finalized = &finalized
	stateRoot, branch := finalityTree(map[uint64]common.Hash{105: finalized.Beacon.Hash(), 55: next.Root()})
	u.Attested.Beacon.StateRoot = stateRoot
	u.FinalityBranch = branch(105)
	u.NextBranch = branch(55)
	signFinalityUpdate(t, &u, key, 342, c.network)
	beacon.updatesErr = errors.New("historical updates unavailable")
	beacon.final = u
	// The response is signed in the slot reached while the request is in flight.
	beacon.afterFinal = func() {
		c.now = func() time.Time { return time.Unix(int64(c.network.GenesisTime+201*12), 0) }
	}
	if _, err = c.FinalizedAnchor(t.Context(), target); err != nil {
		t.Fatal(err)
	}
	if beacon.finalCalls != 1 || beacon.syncCalls != 0 {
		t.Fatalf("sync calls: %d", beacon.finalCalls)
	}
	if _, err = c.FinalizedAnchor(t.Context(), target); err != nil {
		t.Fatal(err)
	}
	if beacon.finalCalls != 1 || beacon.syncCalls != 0 {
		t.Fatal("did not reuse advanced cache")
	}
}

func TestFinalityProofPackageIntegration(t *testing.T) {
	for _, kind := range []string{"state", "receipt", "tx"} {
		t.Run(kind, func(t *testing.T) {
			c, _, beacon, sources := finalityClientFixture(t)
			state := mustLoadStateFixture(t)
			receipt := mustLoadReceiptFixture(t)
			tx := mustLoadTransactionFixture(t)
			var block BlockContext
			switch kind {
			case "state":
				block = state.Block
			case "receipt":
				block = receipt.Block
			case "tx":
				block = tx.Block
			}
			// Re-anchor the unchanged MPT roots under an actual RLP-hashed execution
			// header, then authenticate that header through the synthetic Beacon state.
			h := &types.Header{Number: new(big.Int).SetUint64(block.BlockNumber), Root: block.StateRoot, TxHash: block.TransactionsRoot, ReceiptHash: block.ReceiptsRoot, Difficulty: new(big.Int)}
			exec := c.execution.(*testFinalityExecution)
			exec.headers[h.Hash()] = h
			block, err := BlockContextFromHeader(big.NewInt(1), h)
			if err != nil {
				t.Fatal(err)
			}
			beacon.b.Header = finalityTestHeader(t, 128, h.Hash(), c.network)
			root, branch := finalityTree(map[uint64]common.Hash{54: beacon.b.Committee.Root()})
			beacon.b.Header.Beacon.StateRoot = root
			beacon.b.Branch = branch(54)
			for _, s := range sources {
				s.cp.Root = beacon.b.Header.Beacon.Hash()
				s.headers = map[common.Hash]CheckpointHeader{s.cp.Root: {Header: beacon.b.Header.Beacon, Canonical: true, Finalized: true}}
			}
			switch kind {
			case "state":
				state.Block = block
				err = VerifyStateProofPackageAgainstFinality(t.Context(), &state, c)
			case "tx":
				tx.Block = block
				err = VerifyTransactionProofPackageAgainstFinality(t.Context(), &tx, c)
			case "receipt":
				receipt.Block = block
				err = VerifyReceiptProofPackageWithExpectationsAgainstFinality(t.Context(), &receipt, nil, c)
			}
			if err != nil {
				t.Fatal(err)
			}
			if kind == "receipt" {
				wrong := common.Address{0xff}
				if err = VerifyReceiptProofPackageWithExpectationsAgainstFinality(t.Context(), &receipt, &ReceiptExpectations{Emitter: &wrong}, c); err == nil {
					t.Fatal("ignored event expectation")
				}
			}
		})
	}
}

func TestFinalityConcurrentCalls(t *testing.T) {
	c, target, _, _ := finalityClientFixture(t)
	errs := make(chan error, 4)
	for range 4 {
		go func() { _, err := c.FinalizedAnchor(t.Context(), target); errs <- err }()
	}
	for range 4 {
		if err := <-errs; err != nil {
			t.Fatal(err)
		}
	}
}

// Unknown execution JSON fields do not change the authenticated SSZ root. They
// must not allow a Beacon server to persist a cache that Load cannot read.
func TestFinalityOversizedCacheDoesNotReplaceTrustedState(t *testing.T) {
	c, target, beacon, sources := finalityClientFixture(t)
	if _, err := c.FinalizedAnchor(t.Context(), target); err != nil {
		t.Fatal(err)
	}
	before, err := c.store.Load(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	beacon.b.Header.Beacon.Slot = 8000
	root := beacon.b.Header.Beacon.Hash()
	for _, s := range sources {
		s.cp = FinalityCheckpoint{Epoch: 250, Root: root}
		s.headers[root] = CheckpointHeader{Header: beacon.b.Header.Beacon, Canonical: true, Finalized: true}
	}
	var execution map[string]json.RawMessage
	if err = json.Unmarshal(beacon.b.Header.Execution, &execution); err != nil {
		t.Fatal(err)
	}
	execution["unrelated"] = json.RawMessage(`"` + strings.Repeat("x", 4<<20) + `"`)
	beacon.b.Header.Execution, err = json.Marshal(execution)
	if err != nil {
		t.Fatal(err)
	}
	c.now = func() time.Time { return time.Unix(int64(c.network.GenesisTime+8100*12), 0) }
	trustChanged := false
	c.onTrustChange = func(FinalityTrustEvent) { trustChanged = true }
	if _, err = c.FinalizedAnchor(t.Context(), target); !errors.Is(err, ErrFinalityCache) {
		t.Fatalf("oversized cache accepted: %v", err)
	}
	after, err := c.store.Load(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if string(before) != string(after) || trustChanged {
		t.Fatal("failed refresh replaced trusted state")
	}
}

func TestFinalityNetworkParametersBindCache(t *testing.T) {
	c, target, _, _ := finalityClientFixture(t)
	if _, err := c.FinalizedAnchor(t.Context(), target); err != nil {
		t.Fatal(err)
	}
	deps := FinalityDependencies{Beacon: c.beacon, Checkpoints: c.checkpoints, Store: c.store, Now: c.now}
	for _, test := range []struct {
		name           string
		chain, seconds uint64
	}{{"chain", 2, 12}, {"slot-duration", 1, 3}} {
		t.Run(test.name, func(t *testing.T) {
			other, err := newFinalityClientForNetwork(c.config, c.execution, deps, c.network, test.chain, test.seconds)
			if err != nil {
				t.Fatal(err)
			}
			if _, err = other.load(t.Context()); !errors.Is(err, ErrFinalityCache) {
				t.Fatalf("accepted incompatible cached network: %v", err)
			}
		})
	}
}

func TestFinalitySyncFetchesMissingCommittee(t *testing.T) {
	c, target, beacon, _ := finalityClientFixture(t)
	if _, err := c.FinalizedAnchor(t.Context(), target); err != nil {
		t.Fatal(err)
	}
	cache, err := c.load(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	first, k1 := finalityCommittee(t, 1)
	second, k2 := finalityCommittee(t, 2)
	cache.State = finalityState{Finalized: finalityTestHeader(t, 8000, common.Hash{1}, c.network), Current: first}
	beacon.updates = []FinalityUpdate{finalityTestUpdate(t, 8100, 8064, second, k1, c.network)}
	beacon.final = finalityTestUpdate(t, 8250, 8224, nil, k2, c.network)
	c.now = func() time.Time { return time.Unix(int64(c.network.GenesisTime+8300*12), 0) }
	if err = c.sync(t.Context(), cache); err != nil {
		t.Fatal(err)
	}
	if beacon.syncCalls != 1 || beacon.finalCalls != 1 || cache.State.Finalized.Beacon.Slot != 8224 || *cache.State.Current != *second {
		t.Fatal("did not authenticate missing committee before advancing finality")
	}
}

func TestFinalitySyncRefreshesClockAfterPeriodUpdates(t *testing.T) {
	for _, future := range []bool{false, true} {
		t.Run(fmt.Sprintf("future=%t", future), func(t *testing.T) {
			c, target, beacon, _ := finalityClientFixture(t)
			if _, err := c.FinalizedAnchor(t.Context(), target); err != nil {
				t.Fatal(err)
			}
			cache, err := c.load(t.Context())
			if err != nil {
				t.Fatal(err)
			}
			first, k1 := finalityCommittee(t, 1)
			second, k2 := finalityCommittee(t, 2)
			third, _ := finalityCommittee(t, 3)
			cache.State = finalityState{Finalized: finalityTestHeader(t, 8000, common.Hash{1}, c.network), Current: first}
			beacon.final = finalityTestUpdate(t, 8250, 8224, nil, k2, c.network)
			beacon.updates = []FinalityUpdate{
				finalityTestUpdate(t, 8100, 8064, second, k1, c.network),
				finalityTestUpdate(t, 8260, 8256, third, k2, c.network),
			}
			slot := uint64(8250)
			c.now = func() time.Time { return time.Unix(int64(c.network.GenesisTime+slot*12), 0) }
			beacon.afterFinal = func() { slot = 8251 }
			beacon.afterUpdates = func() {
				slot = 8261
				if future {
					slot = 8260
				}
			}
			err = c.sync(t.Context(), cache)
			if future {
				if err == nil || cache.State.Finalized.Beacon.Slot != 8064 {
					t.Fatalf("future update advanced finality: slot=%d err=%v", cache.State.Finalized.Beacon.Slot, err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if cache.State.Finalized.Beacon.Slot != 8256 || beacon.syncCalls != 1 {
				t.Fatal("did not advance finality after delayed period response")
			}
		})
	}
}
