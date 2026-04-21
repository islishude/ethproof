package proof

import (
	"bytes"
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

func GenerateStateProof(ctx context.Context, req StateProofRequest) (*StateProofPackage, error) {
	rpcs, err := normalizeRPCURLs(req.RPCURLs, req.MinRPCSources)
	if err != nil {
		return nil, err
	}
	sources, err := openRPCSources(ctx, rpcs)
	if err != nil {
		return nil, err
	}
	defer closeRPCSources(sources)

	snapshots := make([]*accountSnapshot, 0, len(sources))
	for _, source := range sources {
		snapshot, snapErr := fetchStateSnapshot(ctx, source, req.BlockNumber, req.Account, req.Slot)
		if snapErr != nil {
			return nil, fmt.Errorf("%s: %w", source.url, snapErr)
		}
		snapshots = append(snapshots, snapshot)
	}
	base, consensus, err := consensusForStateSnapshots(rpcs, snapshots)
	if err != nil {
		return nil, err
	}
	return &StateProofPackage{
		Block:             buildBlockContext(base.Header, consensus),
		Account:           base.Account,
		Slot:              base.Slot,
		AccountRLP:        base.AccountRLP,
		AccountProofNodes: base.AccountProof,
		AccountClaim:      base.AccountClaim,
		StorageValue:      base.StorageValue,
		StorageProofNodes: base.StorageProof,
	}, nil
}

func VerifyStateProofPackage(pkg *StateProofPackage) error {
	accountRLP, err := verifyAccountProof(pkg.Block.StateRoot, pkg.Account, pkg.AccountProofNodes, pkg.AccountClaim)
	if err != nil {
		return err
	}
	if !bytes.Equal(accountRLP, pkg.AccountRLP) {
		return fmt.Errorf("verified account bytes do not match claimed account bytes")
	}
	if _, err := verifyStorageProof(pkg.AccountClaim.StorageRoot, pkg.Slot, pkg.StorageProofNodes, pkg.StorageValue); err != nil {
		return err
	}
	return nil
}

func consensusForStateSnapshots(rpcs []string, snapshots []*accountSnapshot) (*accountSnapshot, SourceConsensus, error) {
	base := snapshots[0]
	for i := 1; i < len(snapshots); i++ {
		other := snapshots[i]
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
		if err := combineMismatch(rpcs[0], rpcs[i], diffs); err != nil {
			return nil, SourceConsensus{}, err
		}
	}
	headerDigest, err := canonicalDigest(base.Header)
	if err != nil {
		return nil, SourceConsensus{}, err
	}
	accountProofDigest, err := canonicalDigest(struct {
		AccountRLP hexutil.Bytes     `json:"accountRlp"`
		Proof      []hexutil.Bytes   `json:"proof"`
		Claim      StateAccountClaim `json:"claim"`
	}{
		AccountRLP: base.AccountRLP,
		Proof:      base.AccountProof,
		Claim:      base.AccountClaim,
	})
	if err != nil {
		return nil, SourceConsensus{}, err
	}
	storageProofDigest, err := canonicalDigest(struct {
		Slot  common.Hash     `json:"slot"`
		Value common.Hash     `json:"value"`
		Proof []hexutil.Bytes `json:"proof"`
	}{
		Slot:  base.Slot,
		Value: base.StorageValue,
		Proof: base.StorageProof,
	})
	if err != nil {
		return nil, SourceConsensus{}, err
	}
	consensus := sourceConsensus(
		"live-rpc",
		rpcs,
		[]ConsensusDigest{
			{Name: "header", Digest: headerDigest},
			{Name: "accountProof", Digest: accountProofDigest},
			{Name: "storageProof", Digest: storageProofDigest},
		},
		[]ConsensusField{
			{Name: "chainId", Value: chainIDString(base.Header.ChainID), Consistent: true},
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
	)
	return base, consensus, nil
}
