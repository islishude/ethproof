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
	"github.com/islishude/ethproof/proof"
)

func buildOfflineTransactionReceiptFixture() (proof.BlockContext, types.Transactions, types.Receipts, error) {
	chainID := big.NewInt(1)
	signer := types.LatestSignerForChainID(chainID)
	keyA, err := crypto.HexToECDSA("59c6995e998f97a5a0044966f094538e11d1b5c5f7e36c7f5d4c1a1f5f1d5edb")
	if err != nil {
		return proof.BlockContext{}, nil, nil, err
	}
	keyB, err := crypto.HexToECDSA("8b3a350cf5c34c9194ca3c4c1b4d1a1e0e9b9e2c0f45b6d0c2b5e2d4a5f6c7d8")
	if err != nil {
		return proof.BlockContext{}, nil, nil, err
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
		return proof.BlockContext{}, nil, nil, fmt.Errorf("sign tx0: %w", err)
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
		return proof.BlockContext{}, nil, nil, fmt.Errorf("sign tx1: %w", err)
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

	header := proof.BlockContext{
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

	return header, txs, receipts, nil
}

func buildOfflineReceiptFixture(header proof.BlockContext, txs types.Transactions, receipts types.Receipts, txIndex uint64) (*ReceiptProofPackage, error) {
	header.ChainID = proofutil.CloneChainID(header.ChainID)
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
	transactionRLP, transactionProofNodes, err := proofutil.BuildIndexTrieProof(blockTransactions, txIndex, header.TransactionsRoot, "transaction")
	if err != nil {
		return nil, err
	}
	log := receipts[txIndex].Logs[0]
	return &ReceiptProofPackage{
		Block:                 header,
		TxHash:                txs[txIndex].Hash(),
		TxIndex:               txIndex,
		LogIndex:              0,
		TransactionRLP:        transactionRLP,
		TransactionProofNodes: transactionProofNodes,
		ReceiptRLP:            receiptRLP,
		ProofNodes:            proofNodes,
		Event: EventClaim{
			Address: log.Address,
			Topics:  append([]common.Hash(nil), log.Topics...),
			Data:    proofutil.CanonicalBytes(log.Data),
		},
	}, nil
}

func buildOfflineTransactionFixture(header proof.BlockContext, txs types.Transactions, txIndex uint64) (*TransactionProofPackage, error) {
	header.ChainID = proofutil.CloneChainID(header.ChainID)
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
		Block:          header,
		TxHash:         targetTx.Hash(),
		TxIndex:        txIndex,
		TransactionRLP: transactionRLP,
		ProofNodes:     proofNodes,
	}, nil
}
