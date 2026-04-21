package proof

import (
	"bytes"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
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
	if base.Slot != other.Slot {
		diffs = append(diffs, "slot mismatch")
	}
	if !bytes.Equal(base.AccountRLP, other.AccountRLP) {
		diffs = append(diffs, "accountRlp mismatch")
	}
	diffs = append(diffs, compareByteSlices("accountProof", base.AccountProof, other.AccountProof)...)
	diffs = append(diffs, compareStateClaim(base.AccountClaim, other.AccountClaim)...)
	if base.StorageValue != other.StorageValue {
		diffs = append(diffs, "storageValue mismatch")
	}
	diffs = append(diffs, compareByteSlices("storageProof", base.StorageProof, other.StorageProof)...)
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
	storageProofDigest, err := proofutil.CanonicalDigest(struct {
		Slot  common.Hash     `json:"slot"`
		Value common.Hash     `json:"value"`
		Proof []hexutil.Bytes `json:"proof"`
	}{
		Slot:  base.Slot,
		Value: base.StorageValue,
		Proof: base.StorageProof,
	})
	if err != nil {
		return SourceConsensus{}, err
	}
	return sourceConsensus(
		"live-rpc",
		rpcs,
		[]ConsensusDigest{
			{Name: "header", Digest: headerDigest},
			{Name: "accountProof", Digest: accountProofDigest},
			{Name: "storageProof", Digest: storageProofDigest},
		},
		[]ConsensusField{
			{Name: "chainId", Value: proofutil.ChainIDString(base.Header.ChainID), Consistent: true},
			{Name: "blockNumber", Value: fmt.Sprintf("%d", base.Header.BlockNumber), Consistent: true},
			{Name: "blockHash", Value: base.Header.BlockHash.Hex(), Consistent: true},
			{Name: "parentHash", Value: base.Header.ParentHash.Hex(), Consistent: true},
			{Name: "stateRoot", Value: base.Header.StateRoot.Hex(), Consistent: true},
			{Name: "transactionsRoot", Value: base.Header.TransactionsRoot.Hex(), Consistent: true},
			{Name: "receiptsRoot", Value: base.Header.ReceiptsRoot.Hex(), Consistent: true},
			{Name: "account", Value: base.Account.Hex(), Consistent: true},
			{Name: "slot", Value: base.Slot.Hex(), Consistent: true},
			{Name: "account.nonce", Value: fmt.Sprintf("%d", base.AccountClaim.Nonce), Consistent: true},
			{Name: "account.balance", Value: base.AccountClaim.Balance, Consistent: true},
			{Name: "account.storageRoot", Value: base.AccountClaim.StorageRoot.Hex(), Consistent: true},
			{Name: "account.codeHash", Value: base.AccountClaim.CodeHash.Hex(), Consistent: true},
			{Name: "storage.value", Value: base.StorageValue.Hex(), Consistent: true},
		},
	), nil
}
