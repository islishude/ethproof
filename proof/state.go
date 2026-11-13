package proof

import (
	"bytes"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
)

// VerifyStateProofPackage verifies internal consistency against the roots embedded in pkg.
//
// Deprecated: use VerifyStateProofPackageAgainstBlockAnchor when a caller-trusted block anchor is
// available, or VerifyStateProofPackageAgainstEmbeddedRoots when only structural verification is intended.
func VerifyStateProofPackage(pkg *StateProofPackage) error {
	return VerifyStateProofPackageAgainstEmbeddedRoots(pkg)
}

// VerifyStateProofPackageAgainstEmbeddedRoots verifies account and storage inclusion against the
// state root carried by pkg. It does not authenticate that root as belonging to a trusted block.
func VerifyStateProofPackageAgainstEmbeddedRoots(pkg *StateProofPackage) error {
	if pkg == nil {
		return fmt.Errorf("state proof package is nil")
	}
	if err := validateBlockContext(pkg.Block); err != nil {
		return err
	}
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
		if err := verifyStorageProof(pkg.AccountClaim.StorageRoot, storageProof.Slot, storageProof.ProofNodes, storageProof.Value); err != nil {
			return fmt.Errorf("verify storageProofs[%d]: %w", i, err)
		}
	}
	return nil
}

// VerifyStateProofPackageAgainstBlockAnchor verifies the state proof and requires every embedded
// block field to match a caller-trusted block anchor.
func VerifyStateProofPackageAgainstBlockAnchor(pkg *StateProofPackage, anchor BlockAnchor) error {
	if err := VerifyStateProofPackageAgainstEmbeddedRoots(pkg); err != nil {
		return err
	}
	return verifyBlockContextAgainstAnchor(pkg.Block, anchor)
}

func validateStateSlots(slots []common.Hash) error {
	if len(slots) == 0 {
		return fmt.Errorf("state proof requires at least one storage slot")
	}

	seen := make(map[common.Hash]struct{}, len(slots))
	for _, slot := range slots {
		if _, ok := seen[slot]; ok {
			return fmt.Errorf("duplicate storage slot %s", slot.Hex())
		}
		seen[slot] = struct{}{}
	}
	return nil
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
