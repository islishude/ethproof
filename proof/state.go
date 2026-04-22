package proof

import (
	"bytes"
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
)

type stateSnapshotCollector struct {
	blockNumber uint64
	account     common.Address
	slot        common.Hash
}

// GenerateStateProof fetches a state proof from every RPC source, requires normalized agreement,
// and returns the agreed proof package.
func GenerateStateProof(ctx context.Context, req StateProofRequest) (*StateProofPackage, error) {
	sourceSet, err := openNormalizedRPCSources(ctx, req.RPCURLs, req.MinRPCSources)
	if err != nil {
		return nil, err
	}
	defer sourceSet.Close()

	return GenerateStateProofFromSources(ctx, StateProofSourcesRequest{
		Sources:       sourceSet.StateSources(),
		MinRPCSources: req.MinRPCSources,
		BlockNumber:   req.BlockNumber,
		Account:       req.Account,
		Slot:          req.Slot,
	})
}

// GenerateStateProofFromSources fetches a state proof from every source, requires normalized
// agreement, and returns the agreed proof package.
func GenerateStateProofFromSources(ctx context.Context, req StateProofSourcesRequest) (*StateProofPackage, error) {
	sourceNames, err := normalizeSourceNames(req.Sources, req.MinRPCSources)
	if err != nil {
		return nil, err
	}
	snapshots, err := collectStateSnapshots(ctx, req)
	if err != nil {
		return nil, err
	}
	base, consensus, err := consensusForStateSnapshots(sourceNames, snapshots)
	if err != nil {
		return nil, err
	}
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
	return pkg, nil
}

// VerifyStateProofPackage verifies the embedded account proof and storage proof locally.
func VerifyStateProofPackage(pkg *StateProofPackage) error {
	return verifyStateProofPackage(pkg)
}

func verifyStateProofPackage(pkg *StateProofPackage) error {
	// First prove the account leaf against stateRoot and verify that the decoded account
	// fields match the claim embedded in the package.
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
	return nil
}

func collectStateSnapshots(ctx context.Context, req StateProofSourcesRequest) ([]*accountSnapshot, error) {
	collector := stateSnapshotCollector{
		blockNumber: req.BlockNumber,
		account:     req.Account,
		slot:        req.Slot,
	}
	return collectFromSources(ctx, req.Sources, collector.fetch)
}

func (c stateSnapshotCollector) fetch(ctx context.Context, source StateSource) (*accountSnapshot, error) {
	return fetchStateSnapshot(ctx, source, c.blockNumber, c.account, c.slot)
}
