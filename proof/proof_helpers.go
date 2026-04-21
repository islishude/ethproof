package proof

import (
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/trie"
	"github.com/islishude/ethproof/internal/proofutil"
)

func buildReceiptTrieAndProof(receipts []hexutil.Bytes, targetIndex uint64, expectedRoot common.Hash) (hexutil.Bytes, []hexutil.Bytes, error) {
	return proofutil.BuildIndexTrieProof(receipts, targetIndex, expectedRoot, "receipt")
}

func buildTransactionTrieAndProof(transactions []hexutil.Bytes, targetIndex uint64, expectedRoot common.Hash) (hexutil.Bytes, []hexutil.Bytes, error) {
	return proofutil.BuildIndexTrieProof(transactions, targetIndex, expectedRoot, "transaction")
}

func verifyAccountProof(stateRoot common.Hash, account common.Address, nodes []hexutil.Bytes, claim StateAccountClaim) ([]byte, error) {
	// Reconstruct the proof DB exactly as trie.VerifyProof expects it: node hash -> node bytes.
	db, err := proofutil.ProofDBFromHexNodes(nodes)
	if err != nil {
		return nil, err
	}
	accountValue, err := trie.VerifyProof(stateRoot, crypto.Keccak256(account.Bytes()), db)
	if err != nil {
		return nil, fmt.Errorf("verify account proof: %w", err)
	}
	if len(accountValue) == 0 {
		return nil, fmt.Errorf("account proof resolved to empty value")
	}

	// Decode the proven account bytes and compare the decoded fields against the explicit claim
	// carried in the package or source snapshot.
	var decoded types.StateAccount
	if err := rlp.DecodeBytes(accountValue, &decoded); err != nil {
		return nil, fmt.Errorf("decode account rlp: %w", err)
	}
	if decoded.Nonce != claim.Nonce {
		return nil, fmt.Errorf("nonce mismatch: got %d want %d", decoded.Nonce, claim.Nonce)
	}
	balance, err := proofutil.ParseHexBig(claim.Balance)
	if err != nil {
		return nil, err
	}
	if decoded.Balance == nil || decoded.Balance.ToBig().Cmp(balance) != 0 {
		return nil, fmt.Errorf("balance mismatch: got %s want %s", proofutil.BalanceHex(decoded.Balance.ToBig()), claim.Balance)
	}
	if decoded.Root != claim.StorageRoot {
		return nil, fmt.Errorf("storageRoot mismatch: got %s want %s", decoded.Root, claim.StorageRoot)
	}
	if common.BytesToHash(decoded.CodeHash) != claim.CodeHash {
		return nil, fmt.Errorf("codeHash mismatch: got %s want %s", common.BytesToHash(decoded.CodeHash), claim.CodeHash)
	}
	return accountValue, nil
}

func verifyStorageProof(storageRoot common.Hash, slot common.Hash, nodes []hexutil.Bytes, expectedValue common.Hash) ([]byte, error) {
	// Storage proofs are verified against keccak(slot), not the raw slot bytes.
	db, err := proofutil.ProofDBFromHexNodes(nodes)
	if err != nil {
		return nil, err
	}
	storageValue, err := trie.VerifyProof(storageRoot, crypto.Keccak256(slot.Bytes()), db)
	if err != nil {
		return nil, fmt.Errorf("verify storage proof: %w", err)
	}

	// Decode the raw trie value and compare it to the normalized 32-byte storage value claim.
	decodedValue, err := proofutil.DecodeStorageProofValue(storageValue)
	if err != nil {
		return nil, err
	}
	if decodedValue != expectedValue {
		return nil, fmt.Errorf("storage value mismatch: got %s want %s", decodedValue, expectedValue)
	}
	return storageValue, nil
}
