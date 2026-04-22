package proof

import (
	"bytes"
	"context"
	"fmt"
	"slices"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

type stateSnapshotCollector struct {
	blockNumber uint64
	account     common.Address
	slots       []common.Hash
}

// GenerateStateProof fetches a state proof from every RPC source, requires normalized agreement,
// and returns the agreed proof package.
func GenerateStateProof(ctx context.Context, req StateProofRequest) (*StateProofPackage, error) {
	slots, err := validateStateSlots(req.Slots)
	if err != nil {
		return nil, err
	}
	req.Slots = slots

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
		Slots:         req.Slots,
	})
}

// GenerateStateProofFromSources fetches a state proof from every source, requires normalized
// agreement, and returns the agreed proof package.
func GenerateStateProofFromSources(ctx context.Context, req StateProofSourcesRequest) (*StateProofPackage, error) {
	slots, err := validateStateSlots(req.Slots)
	if err != nil {
		return nil, err
	}
	req.Slots = slots

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
		AccountRLP:        base.AccountRLP,
		AccountProofNodes: base.AccountProof,
		AccountClaim:      base.AccountClaim,
		StorageProofs:     cloneStateStorageProofs(base.StorageProofs),
	}
	return pkg, nil
}

// VerifyStateProofPackage verifies the embedded account proof and storage proof locally.
func VerifyStateProofPackage(pkg *StateProofPackage) error {
	return verifyStateProofPackage(pkg)
}

func verifyStateProofPackage(pkg *StateProofPackage) error {
	if err := validateStateStorageProofs(pkg.StorageProofs); err != nil {
		return err
	}

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

	// Then prove each requested storage slot against the verified account's storage root.
	for i, storageProof := range pkg.StorageProofs {
		if _, err := verifyStorageProof(pkg.AccountClaim.StorageRoot, storageProof.Slot, storageProof.ProofNodes, storageProof.Value); err != nil {
			return fmt.Errorf("verify storageProofs[%d]: %w", i, err)
		}
	}
	return nil
}

func collectStateSnapshots(ctx context.Context, req StateProofSourcesRequest) ([]*accountSnapshot, error) {
	collector := stateSnapshotCollector{
		blockNumber: req.BlockNumber,
		account:     req.Account,
		slots:       slices.Clone(req.Slots),
	}
	return collectFromSources(ctx, req.Sources, collector.fetch)
}

func (c stateSnapshotCollector) fetch(ctx context.Context, source StateSource) (*accountSnapshot, error) {
	return fetchStateSnapshot(ctx, source, c.blockNumber, c.account, c.slots)
}

func validateStateSlots(slots []common.Hash) ([]common.Hash, error) {
	if len(slots) == 0 {
		return nil, fmt.Errorf("state proof requires at least one storage slot")
	}

	out := make([]common.Hash, len(slots))
	seen := make(map[common.Hash]struct{}, len(slots))
	for i, slot := range slots {
		if _, ok := seen[slot]; ok {
			return nil, fmt.Errorf("duplicate storage slot %s", slot.Hex())
		}
		seen[slot] = struct{}{}
		out[i] = slot
	}
	return out, nil
}

func validateStateStorageProofs(storageProofs []StateStorageProof) error {
	if len(storageProofs) == 0 {
		return fmt.Errorf("state proof package must contain at least one storage proof")
	}

	seen := make(map[common.Hash]struct{}, len(storageProofs))
	for _, storageProof := range storageProofs {
		if _, ok := seen[storageProof.Slot]; ok {
			return fmt.Errorf("state proof package contains duplicate storage slot %s", storageProof.Slot.Hex())
		}
		seen[storageProof.Slot] = struct{}{}
	}
	return nil
}

func cloneStateStorageProofs(in []StateStorageProof) []StateStorageProof {
	out := make([]StateStorageProof, len(in))
	for i, storageProof := range in {
		out[i] = StateStorageProof{
			Slot:  storageProof.Slot,
			Value: storageProof.Value,
		}
		out[i].ProofNodes = make([]hexutil.Bytes, len(storageProof.ProofNodes))
		for j, node := range storageProof.ProofNodes {
			out[i].ProofNodes[j] = common.CopyBytes(node)
		}
	}
	return out
}
