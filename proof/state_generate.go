package proof

import (
	"context"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/ethclient/gethclient"
	"github.com/islishude/ethproof/internal/proofutil"
)

// GenerateStateProof builds a state proof from one caller-owned source.
func GenerateStateProof(ctx context.Context, source StateSource, req StateProofRequest) (*StateProofPackage, error) {
	if err := validateSource(ctx, source); err != nil {
		return nil, err
	}
	if err := validateStateSlots(req.Slots); err != nil {
		return nil, err
	}
	blockNumber, account, slots := req.BlockNumber, req.Account, req.Slots
	chainID, err := source.ChainID(ctx)
	if err != nil {
		return nil, fmt.Errorf("chain id: %w", err)
	}

	// Fetch the block header and eth_getProof payload at the requested block.
	blockArg := new(big.Int).SetUint64(blockNumber)
	header, err := source.HeaderByNumber(ctx, blockArg)
	if err != nil {
		return nil, fmt.Errorf("fetch header: %w", err)
	}
	block, err := BlockContextFromHeader(chainID, header)
	if err != nil {
		return nil, err
	}
	if block.BlockNumber != blockNumber {
		return nil, fmt.Errorf("header block number mismatch: got %d want %d", block.BlockNumber, blockNumber)
	}
	proof, err := source.GetProof(ctx, account, stateSlotKeys(slots), blockArg)
	if err != nil {
		return nil, fmt.Errorf("eth_getProof: %w", err)
	}
	if proof == nil {
		return nil, fmt.Errorf("eth_getProof returned nil account proof")
	}
	if proof.Balance == nil {
		return nil, fmt.Errorf("eth_getProof returned nil account balance")
	}
	if proof.Balance.Sign() < 0 || proof.Balance.BitLen() > 256 {
		return nil, fmt.Errorf("eth_getProof returned invalid account balance")
	}

	// Normalize proof node ordering for deterministic proof output.
	accountProof, err := proofutil.NormalizeHexNodeList(proof.AccountProof)
	if err != nil {
		return nil, err
	}
	accountClaim := StateAccountClaim{
		Nonce:       proof.Nonce,
		Balance:     proofutil.BalanceHex(proof.Balance),
		StorageRoot: proof.StorageHash,
		CodeHash:    proof.CodeHash,
	}

	// Verify the account response locally against the fetched state root.
	accountRLP, err := verifyAccountProof(header.Root, account, accountProof, accountClaim)
	if err != nil {
		return nil, fmt.Errorf("verify source account proof: %w", err)
	}
	storageProofs, err := normalizeStorageProofResults(slots, proof.StorageHash, proof.StorageProof)
	if err != nil {
		return nil, fmt.Errorf("normalize source storage proofs: %w", err)
	}

	pkg := &StateProofPackage{
		Block:             block,
		Account:           account,
		AccountRLP:        proofutil.CanonicalBytes(accountRLP),
		AccountProofNodes: accountProof,
		AccountClaim:      accountClaim,
		StorageProofs:     storageProofs,
	}
	if err := VerifyStateProofPackageAgainstEmbeddedRoots(pkg); err != nil {
		return nil, fmt.Errorf("verify generated state proof package: %w", err)
	}
	return pkg, nil
}

func stateSlotKeys(slots []common.Hash) []string {
	keys := make([]string, len(slots))
	for i, slot := range slots {
		keys[i] = slot.Hex()
	}
	return keys
}

func normalizeStorageProofResults(expectedSlots []common.Hash, storageRoot common.Hash, results []gethclient.StorageResult) ([]StateStorageProof, error) {
	if len(results) != len(expectedSlots) {
		return nil, fmt.Errorf("expected %d storage proofs, got %d", len(expectedSlots), len(results))
	}

	expected := make(map[common.Hash]struct{}, len(expectedSlots))
	for _, slot := range expectedSlots {
		expected[slot] = struct{}{}
	}

	bySlot := make(map[common.Hash]StateStorageProof, len(results))
	for i, result := range results {
		slot, err := parseStorageProofKey(result.Key)
		if err != nil {
			return nil, fmt.Errorf("storage proof %d: %w", i, err)
		}
		if _, ok := expected[slot]; !ok {
			return nil, fmt.Errorf("unexpected storage proof key %s", slot.Hex())
		}
		if _, ok := bySlot[slot]; ok {
			return nil, fmt.Errorf("duplicate storage proof key %s", slot.Hex())
		}

		proofNodes, err := proofutil.NormalizeHexNodeList(result.Proof)
		if err != nil {
			return nil, fmt.Errorf("normalize storage proof %s: %w", slot.Hex(), err)
		}
		value, err := storageResultValueHash(result)
		if err != nil {
			return nil, fmt.Errorf("storage proof %s: %w", slot.Hex(), err)
		}
		if err := verifyStorageProof(storageRoot, slot, proofNodes, value); err != nil {
			return nil, fmt.Errorf("verify storage proof %s: %w", slot.Hex(), err)
		}
		bySlot[slot] = StateStorageProof{
			Slot:       slot,
			Value:      value,
			ProofNodes: proofNodes,
		}
	}

	ordered := make([]StateStorageProof, len(expectedSlots))
	for i, slot := range expectedSlots {
		storageProof, ok := bySlot[slot]
		if !ok {
			return nil, fmt.Errorf("missing storage proof for slot %s", slot.Hex())
		}
		ordered[i] = storageProof
	}
	return ordered, nil
}

func parseStorageProofKey(raw string) (common.Hash, error) {
	if raw == "" {
		return common.Hash{}, fmt.Errorf("storage proof key is empty")
	}

	decoded, err := hexutil.Decode(raw)
	if err != nil {
		return common.Hash{}, fmt.Errorf("decode storage proof key %q: %w", raw, err)
	}
	if len(decoded) > common.HashLength {
		return common.Hash{}, fmt.Errorf("storage proof key %q exceeds 32 bytes", raw)
	}
	return common.BytesToHash(decoded), nil
}

func storageResultValueHash(result gethclient.StorageResult) (common.Hash, error) {
	if result.Value == nil {
		return common.Hash{}, nil
	}
	if result.Value.Sign() < 0 || result.Value.BitLen() > 256 {
		return common.Hash{}, fmt.Errorf("storage value is outside uint256 range")
	}
	return common.BigToHash(result.Value), nil
}
