package proof

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
)

// GenerateStateProof fetches a state proof from every RPC source, requires normalized agreement,
// and returns the agreed proof package.
func GenerateStateProof(ctx context.Context, req StateProofRequest) (*StateProofPackage, error) {
	logger := loggerFromContext(ctx).With("proof_type", "state")
	logger.Info("generate proof started",
		"block_number", req.BlockNumber,
		"account", req.Account,
		"slot", req.Slot,
	)

	// Normalize and deduplicate RPC inputs before any network work so every downstream
	// step sees the exact source set that participates in consensus.
	rpcs, err := normalizeRPCURLs(req.RPCURLs, req.MinRPCSources)
	if err != nil {
		return nil, err
	}
	logger.Debug("normalized rpc sources", "rpc_count", len(rpcs))

	// Fetch the same logical snapshot from every source. Any source-specific error is
	// wrapped with the RPC URL by collectFromRPCs.
	snapshots, err := collectFromRPCs(ctx, rpcs, func(ctx context.Context, source *rpcSource) (*accountSnapshot, error) {
		return fetchStateSnapshot(ctx, source, req.BlockNumber, req.Account, req.Slot)
	})
	if err != nil {
		return nil, err
	}
	logger.Debug("fetched state snapshots", "snapshot_count", len(snapshots))

	// Require byte-level agreement across sources, then embed the canonical snapshot and
	// the consensus metadata derived from that agreed view.
	base, consensus, err := consensusForStateSnapshots(rpcs, snapshots)
	if err != nil {
		return nil, err
	}
	logger.Info("rpc consensus established", "rpc_count", len(rpcs), "block_hash", base.Header.BlockHash)
	pkg := &StateProofPackage{
		Block:             buildBlockContext(base.Header, consensus),
		Account:           base.Account,
		Slot:              base.Slot,
		AccountRLP:        base.AccountRLP,
		AccountProofNodes: base.AccountProof,
		AccountClaim:      base.AccountClaim,
		StorageValue:      base.StorageValue,
		StorageProofNodes: base.StorageProof,
	}
	logger.Info("generate proof completed", "block_number", pkg.Block.BlockNumber, "state_root", pkg.Block.StateRoot)
	return pkg, nil
}

// VerifyStateProofPackage verifies the embedded account proof and storage proof locally.
func VerifyStateProofPackage(pkg *StateProofPackage) error {
	return verifyStateProofPackageWithLogger(discardLogger, pkg)
}

func verifyStateProofPackageWithLogger(logger *slog.Logger, pkg *StateProofPackage) error {
	// First prove the account leaf against stateRoot and verify that the decoded account
	// fields match the claim embedded in the package.
	logger.Debug("verifying local state proof", "block_hash", pkg.Block.BlockHash, "account", pkg.Account, "slot", pkg.Slot)
	accountRLP, err := verifyAccountProof(pkg.Block.StateRoot, pkg.Account, pkg.AccountProofNodes, pkg.AccountClaim)
	if err != nil {
		return err
	}

	// The proof must reconstruct the exact canonical bytes stored in the package, not just
	// an equivalent decoded account.
	if !bytes.Equal(accountRLP, pkg.AccountRLP) {
		return fmt.Errorf("verified account bytes do not match claimed account bytes")
	}

	// Then prove the requested storage slot against the verified account's storage root.
	if _, err := verifyStorageProof(pkg.AccountClaim.StorageRoot, pkg.Slot, pkg.StorageProofNodes, pkg.StorageValue); err != nil {
		return err
	}
	logger.Debug("local state proof verified", "block_hash", pkg.Block.BlockHash, "account", pkg.Account, "slot", pkg.Slot)
	return nil
}
