package proof

import (
	"bytes"
	"fmt"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/islishude/ethproof/internal/proofutil"
)

func consensusForStateSnapshots(rpcs []string, snapshots []*accountSnapshot) (*accountSnapshot, SourceConsensus, error) {
	// Pick a canonical snapshot only after every source agrees on the normalized state view.
	base, err := requireMatchingSnapshots(rpcs, snapshots, compareStateSnapshot)
	if err != nil {
		return nil, SourceConsensus{}, err
	}
	// Persist both machine-checkable digests and human-readable fields in the package so callers
	// can audit which inputs were forced to agree.
	consensus, err := buildStateConsensus(base, rpcs)
	if err != nil {
		return nil, SourceConsensus{}, err
	}
	return base, consensus, nil
}

func compareStateSnapshot(base, other *accountSnapshot) []string {
	var diffs []string
	diffs = append(diffs, compareHeader(base.Header, other.Header)...)
	if base.Account != other.Account {
		diffs = append(diffs, "account mismatch")
	}
	if !bytes.Equal(base.AccountRLP, other.AccountRLP) {
		diffs = append(diffs, "accountRlp mismatch")
	}
	diffs = append(diffs, compareByteSlices("accountProof", base.AccountProof, other.AccountProof)...)
	diffs = append(diffs, compareStateClaim(base.AccountClaim, other.AccountClaim)...)
	diffs = append(diffs, compareStateStorageProofs(base.StorageProofs, other.StorageProofs)...)
	return diffs
}

func buildStateConsensus(base *accountSnapshot, rpcs []string) (SourceConsensus, error) {
	headerDigest, err := proofutil.CanonicalDigest(base.Header)
	if err != nil {
		return SourceConsensus{}, err
	}
	accountProofDigest, err := proofutil.CanonicalDigest(struct {
		AccountRLP hexutil.Bytes     `json:"accountRlp"`
		Proof      []hexutil.Bytes   `json:"proof"`
		Claim      StateAccountClaim `json:"claim"`
	}{
		AccountRLP: base.AccountRLP,
		Proof:      base.AccountProof,
		Claim:      base.AccountClaim,
	})
	if err != nil {
		return SourceConsensus{}, err
	}
	storageProofDigest, err := proofutil.CanonicalDigest(base.StorageProofs)
	if err != nil {
		return SourceConsensus{}, err
	}
	fields := []ConsensusField{
		{Name: "chainId", Value: proofutil.ChainIDString(base.Header.ChainID), Consistent: true},
		{Name: "blockNumber", Value: fmt.Sprintf("%d", base.Header.BlockNumber), Consistent: true},
		{Name: "blockHash", Value: base.Header.BlockHash.Hex(), Consistent: true},
		{Name: "parentHash", Value: base.Header.ParentHash.Hex(), Consistent: true},
		{Name: "stateRoot", Value: base.Header.StateRoot.Hex(), Consistent: true},
		{Name: "transactionsRoot", Value: base.Header.TransactionsRoot.Hex(), Consistent: true},
		{Name: "receiptsRoot", Value: base.Header.ReceiptsRoot.Hex(), Consistent: true},
		{Name: "account", Value: base.Account.Hex(), Consistent: true},
		{Name: "account.nonce", Value: fmt.Sprintf("%d", base.AccountClaim.Nonce), Consistent: true},
		{Name: "account.balance", Value: base.AccountClaim.Balance, Consistent: true},
		{Name: "account.storageRoot", Value: base.AccountClaim.StorageRoot.Hex(), Consistent: true},
		{Name: "account.codeHash", Value: base.AccountClaim.CodeHash.Hex(), Consistent: true},
	}
	for i, storageProof := range base.StorageProofs {
		fields = append(fields,
			ConsensusField{Name: fmt.Sprintf("storageProofs[%d].slot", i), Value: storageProof.Slot.Hex(), Consistent: true},
			ConsensusField{Name: fmt.Sprintf("storageProofs[%d].value", i), Value: storageProof.Value.Hex(), Consistent: true},
		)
	}
	return sourceConsensus(
		"live-rpc",
		rpcs,
		[]ConsensusDigest{
			{Name: "header", Digest: headerDigest},
			{Name: "accountProof", Digest: accountProofDigest},
			{Name: "storageProofs", Digest: storageProofDigest},
		},
		fields,
	), nil
}

func compareStateStorageProofs(base, other []StateStorageProof) []string {
	var diffs []string
	if len(base) != len(other) {
		diffs = append(diffs, fmt.Sprintf("storageProofs length mismatch: got %d want %d", len(other), len(base)))
	}

	limit := min(len(other), len(base))
	for i := range limit {
		if base[i].Slot != other[i].Slot {
			diffs = append(diffs, fmt.Sprintf("storageProofs[%d].slot mismatch", i))
		}
		if base[i].Value != other[i].Value {
			diffs = append(diffs, fmt.Sprintf("storageProofs[%d].value mismatch", i))
		}
		diffs = append(diffs, compareByteSlices(fmt.Sprintf("storageProofs[%d].proofNodes", i), base[i].ProofNodes, other[i].ProofNodes)...)
	}
	return diffs
}
