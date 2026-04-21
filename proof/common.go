package proof

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/islishude/ethproof/internal/proofutil"
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

func buildBlockContext(header blockSnapshotHeader, consensus SourceConsensus) BlockContext {
	return BlockContext{
		ChainID:          proofutil.CloneChainID(header.ChainID),
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
