package main

import (
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/trie"
	"github.com/holiman/uint256"
	"github.com/islishude/ethproof/internal/proofutil"
)

func buildOfflineTransactionReceiptFixture() (blockSnapshotHeader, types.Transactions, types.Receipts, uint64, SourceConsensus, error) {
	chainID := big.NewInt(1)
	signer := types.LatestSignerForChainID(chainID)
	keyA, err := fixedPrivateKey("59c6995e998f97a5a0044966f094538e11d1b5c5f7e36c7f5d4c1a1f5f1d5edb")
	if err != nil {
		return blockSnapshotHeader{}, nil, nil, 0, SourceConsensus{}, err
	}
	keyB, err := fixedPrivateKey("8b3a350cf5c34c9194ca3c4c1b4d1a1e0e9b9e2c0f45b6d0c2b5e2d4a5f6c7d8")
	if err != nil {
		return blockSnapshotHeader{}, nil, nil, 0, SourceConsensus{}, err
	}

	recipientA := common.HexToAddress("0x1000000000000000000000000000000000000001")
	recipientB := common.HexToAddress("0x2000000000000000000000000000000000000002")
	tx0, err := types.SignTx(types.NewTx(&types.DynamicFeeTx{
		ChainID:   chainID,
		Nonce:     3,
		GasTipCap: big.NewInt(1_000_000_000),
		GasFeeCap: big.NewInt(2_000_000_000),
		Gas:       90_000,
		To:        &recipientA,
		Value:     big.NewInt(123456789),
		Data:      common.FromHex("0xabcdef01"),
	}), signer, keyA)
	if err != nil {
		return blockSnapshotHeader{}, nil, nil, 0, SourceConsensus{}, fmt.Errorf("sign tx0: %w", err)
	}
	tx1, err := types.SignTx(types.NewTx(&types.LegacyTx{
		Nonce:    11,
		GasPrice: big.NewInt(3_000_000_000),
		Gas:      21_000,
		To:       &recipientB,
		Value:    big.NewInt(42),
		Data:     common.FromHex("0x010203"),
	}), signer, keyB)
	if err != nil {
		return blockSnapshotHeader{}, nil, nil, 0, SourceConsensus{}, fmt.Errorf("sign tx1: %w", err)
	}

	txs := types.Transactions{tx0, tx1}
	eventLog := &types.Log{
		Address: common.HexToAddress("0xCcCCccccCCCCcCCCCCCcCcCccCcCCCcCcccccccC"),
		Topics: []common.Hash{
			crypto.Keccak256Hash([]byte("Transfer(address,address,uint256)")),
			common.HexToHash("0x0000000000000000000000001111111111111111111111111111111111111111"),
			common.HexToHash("0x0000000000000000000000002222222222222222222222222222222222222222"),
		},
		Data: common.LeftPadBytes(big.NewInt(777).Bytes(), 32),
	}
	receipt0 := &types.Receipt{
		Type:              tx0.Type(),
		Status:            types.ReceiptStatusSuccessful,
		CumulativeGasUsed: 90_000,
		Logs:              []*types.Log{eventLog},
	}
	receipt0.Bloom = types.CreateBloom(receipt0)
	receipt1 := &types.Receipt{
		Type:              tx1.Type(),
		Status:            types.ReceiptStatusSuccessful,
		CumulativeGasUsed: 111_000,
		Logs: []*types.Log{{
			Address: common.HexToAddress("0x3333333333333333333333333333333333333333"),
			Topics: []common.Hash{
				crypto.Keccak256Hash([]byte("Ping(uint256)")),
			},
			Data: common.FromHex("0xdeadbeef"),
		}},
	}
	receipt1.Bloom = types.CreateBloom(receipt1)
	receipts := types.Receipts{receipt0, receipt1}

	header := blockSnapshotHeader{
		ChainID:          uint256.MustFromBig(chainID),
		BlockNumber:      18_765_432,
		ParentHash:       crypto.Keccak256Hash([]byte("offline-parent-transaction-receipt")),
		StateRoot:        crypto.Keccak256Hash([]byte("offline-state-root-transaction-receipt")),
		TransactionsRoot: types.DeriveSha(txs, trie.NewStackTrie(nil)),
		ReceiptsRoot:     types.DeriveSha(receipts, trie.NewStackTrie(nil)),
	}
	ethHeader := &types.Header{
		ParentHash:  header.ParentHash,
		UncleHash:   types.EmptyUncleHash,
		Coinbase:    common.Address{},
		Root:        header.StateRoot,
		TxHash:      header.TransactionsRoot,
		ReceiptHash: header.ReceiptsRoot,
		Bloom:       types.CreateBloom(receipt0),
		Difficulty:  big.NewInt(1),
		Number:      new(big.Int).SetUint64(header.BlockNumber),
		GasLimit:    30_000_000,
		GasUsed:     receipts[len(receipts)-1].CumulativeGasUsed,
		Time:        1_720_000_000,
		Extra:       []byte("offline-fixture"),
		BaseFee:     big.NewInt(1_000_000_000),
	}
	header.BlockHash = ethHeader.Hash()

	blockTransactions := make([]hexutil.Bytes, len(txs))
	for i, tx := range txs {
		encoded, encErr := proofutil.EncodeTransaction(tx)
		if encErr != nil {
			return blockSnapshotHeader{}, nil, nil, 0, SourceConsensus{}, encErr
		}
		blockTransactions[i] = encoded
	}
	targetTransactionDigest, err := offlineTransactionDigests(header, blockTransactions, blockTransactions[0])
	if err != nil {
		return blockSnapshotHeader{}, nil, nil, 0, SourceConsensus{}, err
	}
	consensus := sourceConsensus(
		"offline-fixture",
		nil,
		targetTransactionDigest,
		offlineTransactionFields(tx0.Hash(), 0, header),
	)
	return header, txs, receipts, 0, consensus, nil
}

func buildOfflineReceiptFixture(header blockSnapshotHeader, txs types.Transactions, receipts types.Receipts, txIndex uint64, consensus SourceConsensus) (*ReceiptProofPackage, error) {
	blockTransactions := make([]hexutil.Bytes, len(txs))
	blockReceipts := make([]hexutil.Bytes, len(receipts))
	for i := range txs {
		txHex, err := proofutil.EncodeTransaction(txs[i])
		if err != nil {
			return nil, err
		}
		receiptHex, err := proofutil.EncodeReceipt(receipts[i])
		if err != nil {
			return nil, err
		}
		blockTransactions[i] = txHex
		blockReceipts[i] = receiptHex
	}
	receiptRLP, proofNodes, err := proofutil.BuildIndexTrieProof(blockReceipts, txIndex, header.ReceiptsRoot, "receipt")
	if err != nil {
		return nil, err
	}
	log := receipts[txIndex].Logs[0]
	digests, err := canonicalOfflineReceiptDigests(header, blockTransactions, blockReceipts, blockTransactions[txIndex], receiptRLP, log)
	if err != nil {
		return nil, err
	}
	receiptConsensus := sourceConsensus(
		"offline-fixture",
		nil,
		digests,
		[]ConsensusField{
			{Name: "chainId", Value: proofutil.ChainIDString(header.ChainID), Consistent: true},
			{Name: "blockNumber", Value: fmt.Sprintf("%d", header.BlockNumber), Consistent: true},
			{Name: "blockHash", Value: header.BlockHash.Hex(), Consistent: true},
			{Name: "parentHash", Value: header.ParentHash.Hex(), Consistent: true},
			{Name: "stateRoot", Value: header.StateRoot.Hex(), Consistent: true},
			{Name: "transactionsRoot", Value: header.TransactionsRoot.Hex(), Consistent: true},
			{Name: "receiptsRoot", Value: header.ReceiptsRoot.Hex(), Consistent: true},
			{Name: "txHash", Value: txs[txIndex].Hash().Hex(), Consistent: true},
			{Name: "txIndex", Value: fmt.Sprintf("%d", txIndex), Consistent: true},
			{Name: "logIndex", Value: "0", Consistent: true},
			{Name: "event.address", Value: log.Address.Hex(), Consistent: true},
			{Name: "event.topics", Value: fmt.Sprintf("%v", log.Topics), Consistent: true},
			{Name: "event.data", Value: hexutil.Encode(log.Data), Consistent: true},
		},
	)
	return &ReceiptProofPackage{
		Block:          buildBlockContext(header, receiptConsensus),
		TxHash:         txs[txIndex].Hash(),
		TxIndex:        txIndex,
		LogIndex:       0,
		TransactionRLP: blockTransactions[txIndex],
		ReceiptRLP:     receiptRLP,
		ProofNodes:     proofNodes,
		Event: EventClaim{
			Address: log.Address,
			Topics:  append([]common.Hash(nil), log.Topics...),
			Data:    proofutil.CanonicalBytes(log.Data),
		},
	}, nil
}

func transactionSnapshotFromBlock(header blockSnapshotHeader, txs types.Transactions, txIndex uint64, consensus SourceConsensus) (*TransactionProofPackage, error) {
	if int(txIndex) >= len(txs) {
		return nil, fmt.Errorf("transaction index %d out of range", txIndex)
	}
	blockTransactions := make([]hexutil.Bytes, len(txs))
	for i, tx := range txs {
		encoded, err := proofutil.EncodeTransaction(tx)
		if err != nil {
			return nil, fmt.Errorf("encode transaction %d: %w", i, err)
		}
		blockTransactions[i] = encoded
	}
	transactionRLP, proofNodes, err := proofutil.BuildIndexTrieProof(blockTransactions, txIndex, header.TransactionsRoot, "transaction")
	if err != nil {
		return nil, err
	}
	targetTx, _, err := proofutil.DecodeTransaction(transactionRLP)
	if err != nil {
		return nil, err
	}
	return &TransactionProofPackage{
		Block:          buildBlockContext(header, consensus),
		TxHash:         targetTx.Hash(),
		TxIndex:        txIndex,
		TransactionRLP: transactionRLP,
		ProofNodes:     proofNodes,
	}, nil
}
