package proof

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
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

const defaultMinRPCSources = 3

func normalizeRPCURLs(urls []string, minSources int) ([]string, error) {
	seen := make(map[string]struct{}, len(urls))
	out := make([]string, 0, len(urls))
	for _, raw := range urls {
		url := strings.TrimSpace(raw)
		if url == "" {
			continue
		}
		if _, ok := seen[url]; ok {
			continue
		}
		seen[url] = struct{}{}
		out = append(out, url)
	}
	if minSources == 0 {
		minSources = defaultMinRPCSources
	}
	if minSources < 1 {
		return nil, fmt.Errorf("min rpc sources must be at least 1, got %d", minSources)
	}
	if len(out) < minSources {
		return nil, fmt.Errorf("need at least %d distinct rpc sources, got %d", minSources, len(out))
	}
	return out, nil
}

func makeProofTrie() *trie.Trie {
	tdb := triedb.NewDatabase(rawdb.NewMemoryDatabase(), triedb.HashDefaults)
	return trie.NewEmpty(tdb)
}

func trieIndexKey(index uint64) []byte {
	return rlp.AppendUint64(nil, index)
}

func canonicalBytes(data []byte) hexutil.Bytes {
	return hexutil.Bytes(common.CopyBytes(data))
}

func chainIDFromBig(v *big.Int) (*uint256.Int, error) {
	if v == nil {
		return nil, nil
	}
	out, overflow := uint256.FromBig(v)
	if overflow {
		return nil, fmt.Errorf("chain id %s overflows uint256", v)
	}
	return out, nil
}

func cloneChainID(v *uint256.Int) *uint256.Int {
	if v == nil {
		return nil
	}
	return v.Clone()
}

func chainIDString(v *uint256.Int) string {
	if v == nil {
		return "0"
	}
	return v.String()
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

func normalizeHexNodeList(nodes []string) ([]hexutil.Bytes, error) {
	out := make([]hexutil.Bytes, 0, len(nodes))
	for _, node := range nodes {
		b, err := decodeHexBytes(node)
		if err != nil {
			return nil, fmt.Errorf("decode proof node: %w", err)
		}
		out = append(out, canonicalBytes(b))
	}
	sort.Slice(out, func(i, j int) bool {
		return bytes.Compare(out[i], out[j]) < 0
	})
	return out, nil
}

func proofDBFromHexNodes(nodes []hexutil.Bytes) (*memorydb.Database, error) {
	db := memorydb.New()
	for _, nodeBytes := range nodes {
		hash := crypto.Keccak256Hash(nodeBytes)
		if err := db.Put(hash[:], nodeBytes); err != nil {
			return nil, fmt.Errorf("proof db put: %w", err)
		}
	}
	return db, nil
}

func dumpProofNodes(db *memorydb.Database) ([]hexutil.Bytes, error) {
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
		out[i] = canonicalBytes(item.val)
	}
	return out, nil
}

func encodeTransaction(tx *types.Transaction) (hexutil.Bytes, error) {
	b, err := tx.MarshalBinary()
	if err != nil {
		return nil, err
	}
	return canonicalBytes(b), nil
}

func encodeReceipt(receipt *types.Receipt) (hexutil.Bytes, error) {
	b, err := receipt.MarshalBinary()
	if err != nil {
		return nil, err
	}
	return canonicalBytes(b), nil
}

func decodeTransaction(raw []byte) (*types.Transaction, []byte, error) {
	var tx types.Transaction
	if err := tx.UnmarshalBinary(raw); err != nil {
		return nil, nil, err
	}
	return &tx, common.CopyBytes(raw), nil
}

func decodeReceipt(raw []byte) (*types.Receipt, []byte, error) {
	var receipt types.Receipt
	if err := receipt.UnmarshalBinary(raw); err != nil {
		return nil, nil, err
	}
	return &receipt, common.CopyBytes(raw), nil
}

func canonicalDigest(value any) (common.Hash, error) {
	b, err := json.Marshal(value)
	if err != nil {
		return common.Hash{}, err
	}
	return crypto.Keccak256Hash(b), nil
}

func balanceHex(v *big.Int) string {
	if v == nil {
		return hexutil.EncodeBig(big.NewInt(0))
	}
	return hexutil.EncodeBig(v)
}

func parseHexBig(value string) (*big.Int, error) {
	n, err := hexutil.DecodeBig(value)
	if err != nil {
		return nil, fmt.Errorf("decode quantity %q: %w", value, err)
	}
	return n, nil
}

func decodeStorageProofValue(raw []byte) (common.Hash, error) {
	if len(raw) == 0 {
		return common.Hash{}, nil
	}
	var content []byte
	if err := rlp.DecodeBytes(raw, &content); err != nil {
		return common.Hash{}, fmt.Errorf("decode storage proof value: %w", err)
	}
	return common.BytesToHash(content), nil
}

func encodeStorageProofValue(value common.Hash) ([]byte, error) {
	if value == (common.Hash{}) {
		return nil, nil
	}
	return rlp.EncodeToBytes(common.TrimLeftZeroes(value[:]))
}

func buildBlockContext(header blockSnapshotHeader, consensus SourceConsensus) BlockContext {
	return BlockContext{
		ChainID:          cloneChainID(header.ChainID),
		BlockNumber:      header.BlockNumber,
		BlockHash:        header.BlockHash,
		ParentHash:       header.ParentHash,
		StateRoot:        header.StateRoot,
		TransactionsRoot: header.TransactionsRoot,
		ReceiptsRoot:     header.ReceiptsRoot,
		SourceConsensus:  consensus,
	}
}

func sourceConsensus(mode string, rpcs []string, digests []ConsensusDigest, fields []ConsensusField) SourceConsensus {
	outRPCs := append([]string{}, rpcs...)
	outDigests := append([]ConsensusDigest{}, digests...)
	outFields := append([]ConsensusField{}, fields...)
	return SourceConsensus{
		Mode:    mode,
		RPCs:    outRPCs,
		Digests: outDigests,
		Fields:  outFields,
	}
}

func compareHashSlices(name string, a, b []common.Hash) []string {
	if len(a) != len(b) {
		return []string{fmt.Sprintf("%s length mismatch: %d != %d", name, len(a), len(b))}
	}
	var diffs []string
	for i := range a {
		if a[i] != b[i] {
			diffs = append(diffs, fmt.Sprintf("%s[%d] mismatch", name, i))
		}
	}
	return diffs
}

func compareByteSlices(name string, a, b []hexutil.Bytes) []string {
	if len(a) != len(b) {
		return []string{fmt.Sprintf("%s length mismatch: %d != %d", name, len(a), len(b))}
	}
	var diffs []string
	for i := range a {
		if !bytes.Equal(a[i], b[i]) {
			diffs = append(diffs, fmt.Sprintf("%s[%d] mismatch", name, i))
		}
	}
	return diffs
}

func compareHeader(a, b blockSnapshotHeader) []string {
	var diffs []string
	switch {
	case a.ChainID == nil && b.ChainID == nil:
	case a.ChainID == nil || b.ChainID == nil:
		diffs = append(diffs, "header.chainId mismatch")
	case a.ChainID.Cmp(b.ChainID) != 0:
		diffs = append(diffs, "header.chainId mismatch")
	}
	if a.BlockNumber != b.BlockNumber {
		diffs = append(diffs, "header.blockNumber mismatch")
	}
	if a.BlockHash != b.BlockHash {
		diffs = append(diffs, "header.blockHash mismatch")
	}
	if a.ParentHash != b.ParentHash {
		diffs = append(diffs, "header.parentHash mismatch")
	}
	if a.StateRoot != b.StateRoot {
		diffs = append(diffs, "header.stateRoot mismatch")
	}
	if a.TransactionsRoot != b.TransactionsRoot {
		diffs = append(diffs, "header.transactionsRoot mismatch")
	}
	if a.ReceiptsRoot != b.ReceiptsRoot {
		diffs = append(diffs, "header.receiptsRoot mismatch")
	}
	return diffs
}

func compareStateClaim(a, b StateAccountClaim) []string {
	var diffs []string
	if a.Nonce != b.Nonce {
		diffs = append(diffs, "accountClaim.nonce mismatch")
	}
	if a.Balance != b.Balance {
		diffs = append(diffs, "accountClaim.balance mismatch")
	}
	if a.StorageRoot != b.StorageRoot {
		diffs = append(diffs, "accountClaim.storageRoot mismatch")
	}
	if a.CodeHash != b.CodeHash {
		diffs = append(diffs, "accountClaim.codeHash mismatch")
	}
	return diffs
}

func compareEvent(a, b EventClaim) []string {
	var diffs []string
	if a.Address != b.Address {
		diffs = append(diffs, "event.address mismatch")
	}
	if !bytes.Equal(a.Data, b.Data) {
		diffs = append(diffs, "event.data mismatch")
	}
	diffs = append(diffs, compareHashSlices("event.topics", a.Topics, b.Topics)...)
	return diffs
}

func combineMismatch(sourceA, sourceB string, diffs []string) error {
	if len(diffs) == 0 {
		return nil
	}
	if len(diffs) > 12 {
		diffs = append(diffs[:12], "...")
	}
	return fmt.Errorf("normalized data mismatch between %s and %s: %s", sourceA, sourceB, strings.Join(diffs, ", "))
}
