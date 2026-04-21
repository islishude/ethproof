package proof

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/ethclient/gethclient"
	"github.com/ethereum/go-ethereum/ethdb/memorydb"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/ethereum/go-ethereum/trie"
)

type rpcSource struct {
	url  string
	raw  *rpc.Client
	eth  *ethclient.Client
	geth *gethclient.Client
}

func fetchBlockHeadersFromRPCs(ctx context.Context, urls []string, blockHash common.Hash) ([]blockHeaderSource, error) {
	sources, err := openRPCSources(ctx, urls)
	if err != nil {
		return nil, err
	}
	defer closeRPCSources(sources)

	headers := make([]blockHeaderSource, 0, len(sources))
	for _, source := range sources {
		header, err := fetchBlockHeaderSnapshotByHash(ctx, source, blockHash)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", source.url, err)
		}
		headers = append(headers, blockHeaderSource{
			source: source.url,
			header: header,
		})
	}
	return headers, nil
}

func openRPCSources(ctx context.Context, urls []string) ([]*rpcSource, error) {
	sources := make([]*rpcSource, 0, len(urls))
	for _, url := range urls {
		raw, err := rpc.DialContext(ctx, url)
		if err != nil {
			closeRPCSources(sources)
			return nil, fmt.Errorf("dial rpc %s: %w", url, err)
		}
		sources = append(sources, &rpcSource{
			url:  url,
			raw:  raw,
			eth:  ethclient.NewClient(raw),
			geth: gethclient.New(raw),
		})
	}
	return sources, nil
}

func closeRPCSources(sources []*rpcSource) {
	for _, source := range sources {
		if source != nil && source.raw != nil {
			source.raw.Close()
		}
	}
}

func fetchBlockHeader(ctx context.Context, source *rpcSource, blockHash common.Hash, chainID *big.Int) (*types.Header, blockSnapshotHeader, error) {
	header, err := source.eth.HeaderByHash(ctx, blockHash)
	if err != nil {
		return nil, blockSnapshotHeader{}, fmt.Errorf("fetch header: %w", err)
	}
	chainIDValue, err := chainIDFromBig(chainID)
	if err != nil {
		return nil, blockSnapshotHeader{}, err
	}
	snapshot := blockSnapshotHeader{
		ChainID:          chainIDValue,
		BlockNumber:      header.Number.Uint64(),
		BlockHash:        header.Hash(),
		ParentHash:       header.ParentHash,
		StateRoot:        header.Root,
		TransactionsRoot: header.TxHash,
		ReceiptsRoot:     header.ReceiptHash,
	}
	return header, snapshot, nil
}

func fetchBlockHeaderSnapshotByHash(ctx context.Context, source *rpcSource, blockHash common.Hash) (blockSnapshotHeader, error) {
	chainID, err := source.eth.ChainID(ctx)
	if err != nil {
		return blockSnapshotHeader{}, fmt.Errorf("chain id: %w", err)
	}
	_, snapshot, err := fetchBlockHeader(ctx, source, blockHash, chainID)
	if err != nil {
		return blockSnapshotHeader{}, err
	}
	return snapshot, nil
}

func fetchTransactionSnapshot(ctx context.Context, source *rpcSource, txHash common.Hash) (*transactionSnapshot, error) {
	chainID, err := source.eth.ChainID(ctx)
	if err != nil {
		return nil, fmt.Errorf("chain id: %w", err)
	}
	tx, isPending, err := source.eth.TransactionByHash(ctx, txHash)
	if err != nil {
		return nil, fmt.Errorf("fetch tx: %w", err)
	}
	if isPending {
		return nil, fmt.Errorf("transaction is still pending")
	}
	receipt, err := source.eth.TransactionReceipt(ctx, txHash)
	if err != nil {
		return nil, fmt.Errorf("fetch receipt: %w", err)
	}
	block, err := source.eth.BlockByHash(ctx, receipt.BlockHash)
	if err != nil {
		return nil, fmt.Errorf("fetch block: %w", err)
	}
	if int(receipt.TransactionIndex) >= len(block.Transactions()) {
		return nil, fmt.Errorf("transaction index %d out of range for block size %d", receipt.TransactionIndex, len(block.Transactions()))
	}
	header, headerSnapshot, err := fetchBlockHeader(ctx, source, receipt.BlockHash, chainID)
	if err != nil {
		return nil, err
	}
	if header.Hash() != block.Hash() {
		return nil, fmt.Errorf("header hash %s does not match block hash %s", header.Hash(), block.Hash())
	}
	blockTxs := block.Transactions()
	transactionRLP, err := encodeTransaction(tx)
	if err != nil {
		return nil, fmt.Errorf("encode target tx: %w", err)
	}
	blockTransactions := make([]hexutil.Bytes, len(blockTxs))
	for i, blockTx := range blockTxs {
		encoded, encErr := encodeTransaction(blockTx)
		if encErr != nil {
			return nil, fmt.Errorf("encode block tx %d: %w", i, encErr)
		}
		blockTransactions[i] = encoded
	}
	targetIndex := uint64(receipt.TransactionIndex)
	if blockTxs[targetIndex].Hash() != txHash {
		return nil, fmt.Errorf("block transaction[%d] hash mismatch: got %s want %s", targetIndex, blockTxs[targetIndex].Hash(), txHash)
	}
	if !bytes.Equal(blockTransactions[targetIndex], transactionRLP) {
		return nil, fmt.Errorf("transaction bytes mismatch between block body and tx lookup")
	}
	return &transactionSnapshot{
		Header:            headerSnapshot,
		TxHash:            txHash,
		TxIndex:           targetIndex,
		TransactionRLP:    transactionRLP,
		BlockTransactions: blockTransactions,
	}, nil
}

func fetchReceiptSnapshot(ctx context.Context, source *rpcSource, txHash common.Hash, logIndex uint) (*receiptSnapshot, error) {
	txSnapshot, err := fetchTransactionSnapshot(ctx, source, txHash)
	if err != nil {
		return nil, err
	}
	receipt, err := source.eth.TransactionReceipt(ctx, txHash)
	if err != nil {
		return nil, fmt.Errorf("fetch target receipt: %w", err)
	}
	if logIndex >= uint(len(receipt.Logs)) {
		return nil, fmt.Errorf("log-index %d out of range (receipt has %d logs)", logIndex, len(receipt.Logs))
	}
	if receipt.BlockHash != txSnapshot.Header.BlockHash {
		return nil, fmt.Errorf("target receipt block hash mismatch: got %s want %s", receipt.BlockHash, txSnapshot.Header.BlockHash)
	}
	if uint64(receipt.TransactionIndex) != txSnapshot.TxIndex {
		return nil, fmt.Errorf("target receipt transaction index mismatch: got %d want %d", receipt.TransactionIndex, txSnapshot.TxIndex)
	}
	if receipt.TxHash != txHash {
		return nil, fmt.Errorf("target receipt tx hash mismatch: got %s want %s", receipt.TxHash, txHash)
	}
	receiptRLP, err := encodeReceipt(receipt)
	if err != nil {
		return nil, fmt.Errorf("encode target receipt: %w", err)
	}

	blockReceipts, err := fetchBlockReceipts(ctx, source, txSnapshot.Header.BlockHash, len(txSnapshot.BlockTransactions))
	if err != nil {
		return nil, err
	}
	if !bytes.Equal(blockReceipts[txSnapshot.TxIndex], receiptRLP) {
		return nil, fmt.Errorf("receipt bytes mismatch between block receipts and target receipt lookup")
	}
	log := receipt.Logs[logIndex]
	return &receiptSnapshot{
		Header:            txSnapshot.Header,
		TxHash:            txHash,
		TxIndex:           txSnapshot.TxIndex,
		LogIndex:          logIndex,
		TransactionRLP:    txSnapshot.TransactionRLP,
		ReceiptRLP:        receiptRLP,
		BlockTransactions: txSnapshot.BlockTransactions,
		BlockReceipts:     blockReceipts,
		Event: EventClaim{
			Address: log.Address,
			Topics:  append([]common.Hash(nil), log.Topics...),
			Data:    canonicalBytes(log.Data),
		},
	}, nil
}

func fetchBlockReceipts(ctx context.Context, source *rpcSource, blockHash common.Hash, expectedCount int) ([]hexutil.Bytes, error) {
	receipts, err := source.eth.BlockReceipts(ctx, rpc.BlockNumberOrHashWithHash(blockHash, true))
	if err != nil {
		if isRPCMethodNotFound(err) {
			return fetchBlockReceiptsByTransactionScan(ctx, source, blockHash, expectedCount)
		}
		return nil, fmt.Errorf("fetch block receipts: %w", err)
	}
	return encodeAndValidateBlockReceipts(receipts, blockHash, expectedCount)
}

func fetchBlockReceiptsByTransactionScan(ctx context.Context, source *rpcSource, blockHash common.Hash, expectedCount int) ([]hexutil.Bytes, error) {
	block, err := source.eth.BlockByHash(ctx, blockHash)
	if err != nil {
		return nil, fmt.Errorf("fetch block for receipts: %w", err)
	}
	if len(block.Transactions()) != expectedCount {
		return nil, fmt.Errorf("block transaction count %d does not match expected count %d", len(block.Transactions()), expectedCount)
	}
	blockReceipts := make([]hexutil.Bytes, len(block.Transactions()))
	for i, blockTx := range block.Transactions() {
		receipt, receiptErr := source.eth.TransactionReceipt(ctx, blockTx.Hash())
		if receiptErr != nil {
			return nil, fmt.Errorf("fetch receipt %d/%d (%s): %w", i+1, len(block.Transactions()), blockTx.Hash(), receiptErr)
		}
		if receipt.BlockHash != blockHash {
			return nil, fmt.Errorf("receipt %d block hash mismatch: got %s want %s", i, receipt.BlockHash, blockHash)
		}
		if receipt.TransactionIndex != uint(i) {
			return nil, fmt.Errorf("receipt %d transaction index mismatch: got %d want %d", i, receipt.TransactionIndex, i)
		}
		if receipt.TxHash != blockTx.Hash() {
			return nil, fmt.Errorf("receipt %d tx hash mismatch: got %s want %s", i, receipt.TxHash, blockTx.Hash())
		}
		encoded, encErr := encodeReceipt(receipt)
		if encErr != nil {
			return nil, fmt.Errorf("encode receipt %d: %w", i, encErr)
		}
		blockReceipts[i] = encoded
	}
	return blockReceipts, nil
}

func encodeAndValidateBlockReceipts(receipts []*types.Receipt, blockHash common.Hash, expectedCount int) ([]hexutil.Bytes, error) {
	if len(receipts) != expectedCount {
		return nil, fmt.Errorf("block receipt count %d does not match expected count %d", len(receipts), expectedCount)
	}
	out := make([]hexutil.Bytes, len(receipts))
	for i, receipt := range receipts {
		if receipt == nil {
			return nil, fmt.Errorf("block receipt %d is nil", i)
		}
		if receipt.BlockHash != blockHash {
			return nil, fmt.Errorf("block receipt %d block hash mismatch: got %s want %s", i, receipt.BlockHash, blockHash)
		}
		if receipt.TransactionIndex != uint(i) {
			return nil, fmt.Errorf("block receipt %d transaction index mismatch: got %d want %d", i, receipt.TransactionIndex, i)
		}
		encoded, err := encodeReceipt(receipt)
		if err != nil {
			return nil, fmt.Errorf("encode receipt %d: %w", i, err)
		}
		out[i] = encoded
	}
	return out, nil
}

func isRPCMethodNotFound(err error) bool {
	var rpcErr rpc.Error
	return errors.As(err, &rpcErr) && rpcErr.ErrorCode() == -32601
}

func fetchStateSnapshot(ctx context.Context, source *rpcSource, blockNumber uint64, account common.Address, slot common.Hash) (*accountSnapshot, error) {
	chainID, err := source.eth.ChainID(ctx)
	if err != nil {
		return nil, fmt.Errorf("chain id: %w", err)
	}
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
	accountProof, err := normalizeHexNodeList(proof.AccountProof)
	if err != nil {
		return nil, err
	}
	storageProof, err := normalizeHexNodeList(proof.StorageProof[0].Proof)
	if err != nil {
		return nil, err
	}
	headerSnapshot := blockSnapshotHeader{
		ChainID:          nil,
		BlockNumber:      header.Number.Uint64(),
		BlockHash:        header.Hash(),
		ParentHash:       header.ParentHash,
		StateRoot:        header.Root,
		TransactionsRoot: header.TxHash,
		ReceiptsRoot:     header.ReceiptHash,
	}
	headerSnapshot.ChainID, err = chainIDFromBig(chainID)
	if err != nil {
		return nil, err
	}
	accountClaim := StateAccountClaim{
		Nonce:       proof.Nonce,
		Balance:     balanceHex(proof.Balance),
		StorageRoot: proof.StorageHash,
		CodeHash:    proof.CodeHash,
	}
	accountRLP, err := verifyAccountProof(header.Root, account, accountProof, accountClaim)
	if err != nil {
		return nil, fmt.Errorf("verify source account proof: %w", err)
	}
	expectedStorageValue := common.BigToHash(proof.StorageProof[0].Value)
	if _, err := verifyStorageProof(proof.StorageHash, slot, storageProof, expectedStorageValue); err != nil {
		return nil, fmt.Errorf("verify source storage proof: %w", err)
	}
	return &accountSnapshot{
		Header:       headerSnapshot,
		Account:      account,
		Slot:         slot,
		AccountRLP:   canonicalBytes(accountRLP),
		AccountProof: accountProof,
		AccountClaim: accountClaim,
		StorageValue: expectedStorageValue,
		StorageProof: storageProof,
	}, nil
}

func buildReceiptTrieAndProof(receipts []hexutil.Bytes, targetIndex uint64, expectedRoot common.Hash) (hexutil.Bytes, []hexutil.Bytes, error) {
	tr := makeProofTrie()
	for i, receiptHex := range receipts {
		if err := tr.Update(trieIndexKey(uint64(i)), receiptHex); err != nil {
			return nil, nil, fmt.Errorf("receipt trie update %d: %w", i, err)
		}
	}
	root := tr.Hash()
	if root != expectedRoot {
		return nil, nil, fmt.Errorf("derived receiptsRoot mismatch: local=%s expected=%s", root, expectedRoot)
	}
	proofDB := memorydb.New()
	if err := tr.Prove(trieIndexKey(targetIndex), proofDB); err != nil {
		return nil, nil, fmt.Errorf("prove receipt inclusion: %w", err)
	}
	nodes, err := dumpProofNodes(proofDB)
	if err != nil {
		return nil, nil, err
	}
	return receipts[targetIndex], nodes, nil
}

func buildTransactionTrieAndProof(transactions []hexutil.Bytes, targetIndex uint64, expectedRoot common.Hash) (hexutil.Bytes, []hexutil.Bytes, error) {
	tr := makeProofTrie()
	for i, txHex := range transactions {
		if err := tr.Update(trieIndexKey(uint64(i)), txHex); err != nil {
			return nil, nil, fmt.Errorf("transaction trie update %d: %w", i, err)
		}
	}
	root := tr.Hash()
	if root != expectedRoot {
		return nil, nil, fmt.Errorf("derived transactionsRoot mismatch: local=%s expected=%s", root, expectedRoot)
	}
	proofDB := memorydb.New()
	if err := tr.Prove(trieIndexKey(targetIndex), proofDB); err != nil {
		return nil, nil, fmt.Errorf("prove transaction inclusion: %w", err)
	}
	nodes, err := dumpProofNodes(proofDB)
	if err != nil {
		return nil, nil, err
	}
	return transactions[targetIndex], nodes, nil
}

func verifyAccountProof(stateRoot common.Hash, account common.Address, nodes []hexutil.Bytes, claim StateAccountClaim) ([]byte, error) {
	db, err := proofDBFromHexNodes(nodes)
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
	var decoded types.StateAccount
	if err := rlp.DecodeBytes(accountValue, &decoded); err != nil {
		return nil, fmt.Errorf("decode account rlp: %w", err)
	}
	if decoded.Nonce != claim.Nonce {
		return nil, fmt.Errorf("nonce mismatch: got %d want %d", decoded.Nonce, claim.Nonce)
	}
	balance, err := parseHexBig(claim.Balance)
	if err != nil {
		return nil, err
	}
	if decoded.Balance == nil || decoded.Balance.ToBig().Cmp(balance) != 0 {
		return nil, fmt.Errorf("balance mismatch: got %s want %s", balanceHex(decoded.Balance.ToBig()), claim.Balance)
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
	db, err := proofDBFromHexNodes(nodes)
	if err != nil {
		return nil, err
	}
	storageValue, err := trie.VerifyProof(storageRoot, crypto.Keccak256(slot.Bytes()), db)
	if err != nil {
		return nil, fmt.Errorf("verify storage proof: %w", err)
	}
	decodedValue, err := decodeStorageProofValue(storageValue)
	if err != nil {
		return nil, err
	}
	if decodedValue != expectedValue {
		return nil, fmt.Errorf("storage value mismatch: got %s want %s", decodedValue, expectedValue)
	}
	return storageValue, nil
}
