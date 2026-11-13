package proof

import (
	"context"
	"encoding/binary"
	"errors"
	"math/big"
	"os"
	"path/filepath"
	"testing"
	"time"

	bp "github.com/ethereum/go-ethereum/beacon/params"
	"github.com/ethereum/go-ethereum/common"
	"gopkg.in/yaml.v3"
)

// TestPoSE2E uses actual Reth and Lighthouse nodes, including their BLS signatures.
// It is opt-in because starting a fresh PoS network takes several epochs.
func TestPoSE2E(t *testing.T) {
	if os.Getenv("ETH_PROOF_REQUIRE_POS_E2E") != "1" {
		t.Skip("run make pos-e2e-test")
	}
	ctx, cancel := context.WithTimeout(t.Context(), 15*time.Minute)
	defer cancel()
	const executionURL = "http://127.0.0.1:18545"
	beaconURLs := []string{"http://127.0.0.1:15052", "http://127.0.0.1:15053", "http://127.0.0.1:15054"}
	networkDir := filepath.Join("..", "bin", "pos-testnet", "network")
	net, slotSeconds := loadPoSNetwork(t, networkDir)
	source, err := openRPCSource(ctx, executionURL)
	if err != nil {
		t.Fatal(err)
	}
	defer source.Close()
	beacon, err := NewBeaconFinalitySource(beaconURLs[0], nil)
	if err != nil {
		t.Fatal(err)
	}
	counted := &countedPoSBeacon{BeaconFinalitySource: beacon}
	checkpoints := make([]CheckpointSource, len(beaconURLs))
	for i, url := range beaconURLs {
		checkpoints[i], err = NewCheckpointSource("local-node-"+big.NewInt(int64(i+1)).String(), url, nil)
		if err != nil {
			t.Fatal(err)
		}
	}
	cfg := FinalityConfig{Network: "pos-e2e", CacheDir: t.TempDir(), TimeoutSeconds: 30}
	trustChanges := 0
	deps := FinalityDependencies{Beacon: counted, Checkpoints: checkpoints, OnTrustChange: func(FinalityTrustEvent) { trustChanges++ }}
	client, err := newFinalityClientForNetwork(cfg, source, deps, net, 31337, slotSeconds)
	if err != nil {
		t.Fatal(err)
	}
	t.Log("waiting for three-node checkpoint agreement and a light-client bootstrap")
	waitPoS(t, ctx, func() error {
		cp, err := checkpoints[0].FinalizedCheckpoint(ctx)
		if err != nil {
			return err
		}
		if cp.Epoch < 2 {
			return errors.New("waiting for finalized epoch 2")
		}
		b, err := beacon.Bootstrap(ctx, cp.Root)
		if err != nil {
			return err
		}
		hash, err := verifyFinalityHeader(b.Header, net)
		if err != nil {
			return err
		}
		header, err := source.HeaderByHash(ctx, hash)
		if err != nil {
			return err
		}
		block, err := BlockContextFromHeader(big.NewInt(31337), header)
		if err != nil {
			return err
		}
		_, err = client.FinalizedAnchor(ctx, block)
		return err
	})
	before, err := client.load(ctx)
	if err != nil {
		t.Fatal(err)
	}
	initialSlot := before.State.Finalized.Beacon.Slot
	if trustChanges != 1 {
		t.Fatalf("checkpoint acquisitions: %d", trustChanges)
	}
	t.Logf("authenticated initial finalized slot %d; deploying new transactions", initialSlot)
	scenario := deployProofComplexScenario(t, ctx, source.eth, executionURL)
	tx, err := GenerateTransactionProof(ctx, source, TransactionProofRequest{TxHash: scenario.txHash})
	if err != nil {
		t.Fatal(err)
	}
	receipt, err := GenerateReceiptProof(ctx, source, ReceiptProofRequest{TxHash: scenario.txHash, LogIndex: scenario.logIndex})
	if err != nil {
		t.Fatal(err)
	}
	state, err := GenerateStateProof(ctx, source, StateProofRequest{BlockNumber: scenario.blockNumber, Account: scenario.contractAddress, Slots: []common.Hash{scenario.mainlineSlot}})
	if err != nil {
		t.Fatal(err)
	}
	t.Run("reject_before_finality", func(t *testing.T) {
		if err := VerifyTransactionProofPackageAgainstFinality(ctx, tx, client); !errors.Is(err, ErrNotFinalized) {
			t.Fatalf("expected ErrNotFinalized, got %v", err)
		}
	})
	t.Logf("waiting for transaction block %d to finalize", scenario.blockNumber)
	waitPoS(t, ctx, func() error { _, err := client.FinalizedAnchor(ctx, tx.Block); return err })
	after, err := client.load(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if after.State.Finalized.Beacon.Slot <= initialSlot || counted.finalityUpdates == 0 || trustChanges != 1 {
		t.Fatal("expected signed incremental finality without checkpoint reacquisition")
	}
	t.Logf("signed updates advanced finalized slot %d -> %d", initialSlot, after.State.Finalized.Beacon.Slot)
	t.Run("transaction", func(t *testing.T) {
		if err := VerifyTransactionProofPackageAgainstFinality(ctx, tx, client); err != nil {
			t.Fatal(err)
		}
	})
	t.Run("receipt_event", func(t *testing.T) {
		expect := &ReceiptExpectations{Emitter: &scenario.contractAddress, Topics: scenario.eventTopics, Data: scenario.eventData}
		if err := VerifyReceiptProofPackageWithExpectationsAgainstFinality(ctx, receipt, expect, client); err != nil {
			t.Fatal(err)
		}
	})
	t.Run("storage", func(t *testing.T) {
		if err := VerifyStateProofPackageAgainstFinality(ctx, state, client); err != nil {
			t.Fatal(err)
		}
		if state.StorageProofs[0].Value != common.BigToHash(scenario.mainlineValue) {
			t.Fatal("wrong proven storage value")
		}
	})
	t.Run("restart_cache", func(t *testing.T) {
		bootstrapCalls := counted.bootstraps
		restarted, err := newFinalityClientForNetwork(cfg, source, deps, net, 31337, slotSeconds)
		if err != nil {
			t.Fatal(err)
		}
		if err = VerifyTransactionProofPackageAgainstFinality(ctx, tx, restarted); err != nil {
			t.Fatal(err)
		}
		if counted.bootstraps != bootstrapCalls || trustChanges != 1 {
			t.Fatal("restart reacquired checkpoint")
		}
	})
}

type countedPoSBeacon struct {
	BeaconFinalitySource
	bootstraps, finalityUpdates int
}

func (b *countedPoSBeacon) Bootstrap(ctx context.Context, root common.Hash) (FinalityBootstrap, error) {
	b.bootstraps++
	return b.BeaconFinalitySource.Bootstrap(ctx, root)
}
func (b *countedPoSBeacon) FinalityUpdate(ctx context.Context) (FinalityUpdate, error) {
	b.finalityUpdates++
	return b.BeaconFinalitySource.FinalityUpdate(ctx)
}

func loadPoSNetwork(t *testing.T, dir string) (*bp.ChainConfig, uint64) {
	t.Helper()
	config, err := os.ReadFile(filepath.Join(dir, "config.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	genesis, err := os.ReadFile(filepath.Join(dir, "genesis.ssz"))
	if err != nil {
		t.Fatal(err)
	}
	if len(genesis) < 40 {
		t.Fatal("truncated local genesis state")
	}
	// All supported BeaconState schemas begin with genesis_time and genesis_validators_root.
	net := &bp.ChainConfig{GenesisTime: binary.LittleEndian.Uint64(genesis[:8]), GenesisValidatorsRoot: common.BytesToHash(genesis[8:40])}
	if err = net.LoadForks(config); err != nil {
		t.Fatal(err)
	}
	var timing struct {
		Preset  string `yaml:"PRESET_BASE"`
		SlotMS  uint64 `yaml:"SLOT_DURATION_MS"`
		Seconds uint64 `yaml:"SECONDS_PER_SLOT"`
	}
	if err = yaml.Unmarshal(config, &timing); err != nil {
		t.Fatal(err)
	}
	seconds := timing.SlotMS / 1000
	if seconds == 0 {
		seconds = timing.Seconds
	}
	if timing.Preset != "mainnet" || seconds != 3 {
		t.Fatal("unexpected local network preset or slot duration")
	}
	return net, seconds
}
func waitPoS(t *testing.T, ctx context.Context, check func() error) {
	t.Helper()
	var last error
	for {
		if last = check(); last == nil {
			return
		}
		select {
		case <-ctx.Done():
			t.Fatalf("PoS readiness timeout: %v (last: %v)", ctx.Err(), last)
		case <-time.After(2 * time.Second):
		}
	}
}
