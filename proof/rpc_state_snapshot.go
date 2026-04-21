package proof

import (
	"context"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/islishude/ethproof/internal/proofutil"
)

func fetchStateSnapshot(ctx context.Context, source *rpcSource, blockNumber uint64, account common.Address, slot common.Hash) (*accountSnapshot, error) {
	logger := loggerFromContext(ctx)
	logger.Debug("fetching state snapshot", "rpc_url", source.url, "block_number", blockNumber, "account", account, "slot", slot)
	chainID, err := source.eth.ChainID(ctx)
	if err != nil {
		return nil, fmt.Errorf("chain id: %w", err)
	}

	// Fetch both the block header and eth_getProof payload at the requested block so the snapshot
	// can later be compared across RPCs without any additional interpretation.
	blockArg := new(big.Int).SetUint64(blockNumber)
	header, err := source.eth.HeaderByNumber(ctx, blockArg)
	if err != nil {
		return nil, fmt.Errorf("fetch header: %w", err)
	}
	proof, err := source.geth.GetProof(ctx, account, []string{slot.Hex()}, blockArg)
	if err != nil {
		return nil, fmt.Errorf("eth_getProof: %w", err)
	}
	if len(proof.StorageProof) != 1 {
		return nil, fmt.Errorf("expected exactly one storage proof, got %d", len(proof.StorageProof))
	}
	logger.Debug("fetched state rpc payloads", "rpc_url", source.url, "block_hash", header.Hash())

	// Normalize proof node ordering before consensus comparison. Different RPCs can return
	// equivalent node sets in different orders.
	accountProof, err := proofutil.NormalizeHexNodeList(proof.AccountProof)
	if err != nil {
		return nil, err
	}
	storageProof, err := proofutil.NormalizeHexNodeList(proof.StorageProof[0].Proof)
	if err != nil {
		return nil, err
	}
	chainIDValue, err := proofutil.ChainIDFromBig(chainID)
	if err != nil {
		return nil, err
	}

	headerSnapshot := blockSnapshotHeader{
		ChainID:          chainIDValue,
		BlockNumber:      header.Number.Uint64(),
		BlockHash:        header.Hash(),
		ParentHash:       header.ParentHash,
		StateRoot:        header.Root,
		TransactionsRoot: header.TxHash,
		ReceiptsRoot:     header.ReceiptHash,
	}
	accountClaim := StateAccountClaim{
		Nonce:       proof.Nonce,
		Balance:     proofutil.BalanceHex(proof.Balance),
		StorageRoot: proof.StorageHash,
		CodeHash:    proof.CodeHash,
	}

	// Re-verify the source response locally before accepting it into consensus. This prevents a
	// source from contributing malformed proof data that merely "matches" another malformed source.
	accountRLP, err := verifyAccountProof(header.Root, account, accountProof, accountClaim)
	if err != nil {
		return nil, fmt.Errorf("verify source account proof: %w", err)
	}
	expectedStorageValue := common.BigToHash(proof.StorageProof[0].Value)
	if _, err := verifyStorageProof(proof.StorageHash, slot, storageProof, expectedStorageValue); err != nil {
		return nil, fmt.Errorf("verify source storage proof: %w", err)
	}
	logger.Debug("validated state snapshot locally", "rpc_url", source.url, "block_hash", header.Hash())

	return &accountSnapshot{
		Header:       headerSnapshot,
		Account:      account,
		Slot:         slot,
		AccountRLP:   proofutil.CanonicalBytes(accountRLP),
		AccountProof: accountProof,
		AccountClaim: accountClaim,
		StorageValue: expectedStorageValue,
		StorageProof: storageProof,
	}, nil
}
