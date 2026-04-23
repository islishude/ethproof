// Package proofutil contains low-level proof encoding, trie, and normalization helpers shared
// across proof generation, verification, and fixture construction.
package proofutil

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"sort"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethdb/memorydb"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/trie"
	"github.com/ethereum/go-ethereum/triedb"
	"github.com/holiman/uint256"
)

// MakeProofTrie creates an in-memory trie configured for deterministic proof construction.
func MakeProofTrie() *trie.Trie {
	tdb := triedb.NewDatabase(rawdb.NewMemoryDatabase(), triedb.HashDefaults)
	return trie.NewEmpty(tdb)
}

// TrieIndexKey encodes a receipt or transaction index as the trie key used by Ethereum block tries.
func TrieIndexKey(index uint64) []byte {
	return rlp.AppendUint64(nil, index)
}

// CanonicalBytes returns a detached copy of data as hexutil.Bytes.
func CanonicalBytes(data []byte) hexutil.Bytes {
	return hexutil.Bytes(common.CopyBytes(data))
}

// ChainIDFromBig converts a big.Int chain ID into uint256 form.
func ChainIDFromBig(v *big.Int) (*uint256.Int, error) {
	if v == nil {
		return nil, errors.New("nil *big.Int")
	}
	out, overflow := uint256.FromBig(v)
	if overflow {
		return nil, fmt.Errorf("chain id %s overflows uint256", v)
	}
	return out, nil
}

// CloneChainID defensively clones a chain ID pointer.
func CloneChainID(v *uint256.Int) *uint256.Int {
	if v == nil {
		return nil
	}
	return v.Clone()
}

// ChainIDString renders a chain ID for JSON-friendly consensus field output.
func ChainIDString(v *uint256.Int) string {
	if v == nil {
		return "0"
	}
	return v.String()
}

// NormalizeHexNodeList decodes hex-encoded proof nodes and sorts them by byte value so that
// equivalent proofs from different sources compare deterministically.
func NormalizeHexNodeList(nodes []string) ([]hexutil.Bytes, error) {
	out := make([]hexutil.Bytes, 0, len(nodes))
	for _, node := range nodes {
		b, err := decodeHexBytes(node)
		if err != nil {
			return nil, fmt.Errorf("decode proof node: %w", err)
		}
		out = append(out, CanonicalBytes(b))
	}
	sort.Slice(out, func(i, j int) bool {
		return bytes.Compare(out[i], out[j]) < 0
	})
	return out, nil
}

// ProofDBFromHexNodes builds a proof database keyed by node hash for trie verification.
func ProofDBFromHexNodes(nodes []hexutil.Bytes) (*memorydb.Database, error) {
	db := memorydb.New()
	for _, nodeBytes := range nodes {
		hash := crypto.Keccak256Hash(nodeBytes)
		if err := db.Put(hash[:], nodeBytes); err != nil {
			return nil, fmt.Errorf("proof db put: %w", err)
		}
	}
	return db, nil
}

// DumpProofNodes returns proof DB values ordered by key so serialized proofs are deterministic.
func DumpProofNodes(db *memorydb.Database) ([]hexutil.Bytes, error) {
	it := db.NewIterator(nil, nil)
	defer it.Release()

	type item struct {
		key []byte
		val []byte
	}

	var items []item
	for it.Next() {
		items = append(items, item{
			key: append([]byte(nil), it.Key()...),
			val: append([]byte(nil), it.Value()...),
		})
	}
	if err := it.Error(); err != nil {
		return nil, fmt.Errorf("iterate proof db: %w", err)
	}
	sort.Slice(items, func(i, j int) bool {
		return bytes.Compare(items[i].key, items[j].key) < 0
	})

	out := make([]hexutil.Bytes, len(items))
	for i, item := range items {
		out[i] = CanonicalBytes(item.val)
	}
	return out, nil
}

// BuildIndexTrieProof rebuilds an Ethereum index trie locally, checks the derived root, and
// extracts the proof nodes for targetIndex.
func BuildIndexTrieProof(entries []hexutil.Bytes, targetIndex uint64, expectedRoot common.Hash, entryName string) (hexutil.Bytes, []hexutil.Bytes, error) {
	tr := MakeProofTrie()
	for i, entry := range entries {
		if err := tr.Update(TrieIndexKey(uint64(i)), entry); err != nil {
			return nil, nil, fmt.Errorf("%s trie update %d: %w", entryName, i, err)
		}
	}
	root := tr.Hash()
	if root != expectedRoot {
		return nil, nil, fmt.Errorf("derived %ssRoot mismatch: local=%s expected=%s", entryName, root, expectedRoot)
	}
	proofDB := memorydb.New()
	if err := tr.Prove(TrieIndexKey(targetIndex), proofDB); err != nil {
		return nil, nil, fmt.Errorf("prove %s inclusion: %w", entryName, err)
	}
	nodes, err := DumpProofNodes(proofDB)
	if err != nil {
		return nil, nil, err
	}
	return entries[targetIndex], nodes, nil
}

// EncodeTransaction returns the canonical binary form of tx.
func EncodeTransaction(tx *types.Transaction) (hexutil.Bytes, error) {
	b, err := tx.MarshalBinary()
	if err != nil {
		return nil, err
	}
	return CanonicalBytes(b), nil
}

// EncodeReceipt returns the canonical binary form of receipt.
func EncodeReceipt(receipt *types.Receipt) (hexutil.Bytes, error) {
	b, err := receipt.MarshalBinary()
	if err != nil {
		return nil, err
	}
	return CanonicalBytes(b), nil
}

// DecodeTransaction decodes raw transaction bytes and returns both the transaction and a detached copy of raw.
func DecodeTransaction(raw []byte) (*types.Transaction, []byte, error) {
	var tx types.Transaction
	if err := tx.UnmarshalBinary(raw); err != nil {
		return nil, nil, err
	}
	return &tx, common.CopyBytes(raw), nil
}

// DecodeReceipt decodes raw receipt bytes and returns both the receipt and a detached copy of raw.
func DecodeReceipt(raw []byte) (*types.Receipt, []byte, error) {
	var receipt types.Receipt
	if err := receipt.UnmarshalBinary(raw); err != nil {
		return nil, nil, err
	}
	return &receipt, common.CopyBytes(raw), nil
}

// CanonicalDigest hashes the JSON encoding of value for consensus comparison.
func CanonicalDigest(value any) (common.Hash, error) {
	b, err := json.Marshal(value)
	if err != nil {
		return common.Hash{}, err
	}
	return crypto.Keccak256Hash(b), nil
}

// BalanceHex renders a balance as an Ethereum quantity string.
func BalanceHex(v *big.Int) string {
	if v == nil {
		return hexutil.EncodeBig(big.NewInt(0))
	}
	return hexutil.EncodeBig(v)
}

// ParseHexBig parses an Ethereum quantity string into a big.Int.
func ParseHexBig(value string) (*big.Int, error) {
	n, err := hexutil.DecodeBig(value)
	if err != nil {
		return nil, fmt.Errorf("decode quantity %q: %w", value, err)
	}
	return n, nil
}

// EncodeStorageProofValue returns the RLP payload expected for a storage proof value.
func EncodeStorageProofValue(value common.Hash) ([]byte, error) {
	if value == (common.Hash{}) {
		return nil, nil
	}
	return rlp.EncodeToBytes(common.TrimLeftZeroes(value[:]))
}

// DecodeStorageProofValue decodes the raw trie value returned by storage proof verification.
func DecodeStorageProofValue(raw []byte) (common.Hash, error) {
	if len(raw) == 0 {
		return common.Hash{}, nil
	}
	var content []byte
	if err := rlp.DecodeBytes(raw, &content); err != nil {
		return common.Hash{}, fmt.Errorf("decode storage proof value: %w", err)
	}
	return common.BytesToHash(content), nil
}

func decodeHexBytes(value string) ([]byte, error) {
	return hex.DecodeString(trim0x(value))
}

func trim0x(s string) string {
	if strings.HasPrefix(s, "0x") || strings.HasPrefix(s, "0X") {
		return s[2:]
	}
	return s
}
