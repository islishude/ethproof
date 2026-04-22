package proof

import (
	"bytes"
	"context"
	"fmt"
)

// GenerateStateProof fetches a state proof from every RPC source, requires normalized agreement,
// and returns the agreed proof package.
func GenerateStateProof(ctx context.Context, req StateProofRequest) (*StateProofPackage, error) {
	return withNormalizedRPCSources(ctx, req.RPCURLs, req.MinRPCSources, func(sources []*rpcSource) (*StateProofPackage, error) {
		return GenerateStateProofFromSources(ctx, StateProofSourcesRequest{
			Sources:       stateSourcesFromRPCSources(sources),
			MinRPCSources: req.MinRPCSources,
			BlockNumber:   req.BlockNumber,
			Account:       req.Account,
			Slot:          req.Slot,
		})
	})
}

// GenerateStateProofFromSources fetches a state proof from every source, requires normalized
// agreement, and returns the agreed proof package.
func GenerateStateProofFromSources(ctx context.Context, req StateProofSourcesRequest) (*StateProofPackage, error) {
	sourceNames, err := normalizeSourceNames(req.Sources, req.MinRPCSources)
	if err != nil {
		return nil, err
	}
	snapshots, err := collectFromSources(ctx, req.Sources, func(ctx context.Context, source StateSource) (*accountSnapshot, error) {
		return fetchStateSnapshot(ctx, source, req.BlockNumber, req.Account, req.Slot)
	})
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
