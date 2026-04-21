package proof

import (
	"crypto/ecdsa"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethdb/memorydb"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/trie"
	"github.com/holiman/uint256"
)

func BuildOfflineFixtures() (*OfflineFixtures, error) {
	txReceiptHeader, txs, receipts, txIndex, receiptConsensus, err := buildOfflineTransactionReceiptFixture()
	if err != nil {
		return nil, err
	}
	transactionFixture, err := transactionSnapshotFromBlock(txReceiptHeader, txs, txIndex, receiptConsensus)
	if err != nil {
		return nil, err
	}
	receiptFixture, err := buildOfflineReceiptFixture(txReceiptHeader, txs, receipts, txIndex, receiptConsensus)
	if err != nil {
		return nil, err
	}
	stateFixture, err := buildOfflineStateFixture()
	if err != nil {
		return nil, err
	}
	return &OfflineFixtures{
		State:       stateFixture,
		Receipt:     receiptFixture,
		Transaction: transactionFixture,
	}, nil
}

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
		ChainID:          chainID.String(),
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
	blockTransactions := make([]string, len(txs))
	for i, tx := range txs {
		encoded, encErr := encodeTransaction(tx)
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
	blockTransactions := make([]string, len(txs))
	blockReceipts := make([]string, len(receipts))
	for i := range txs {
		txHex, err := encodeTransaction(txs[i])
		if err != nil {
			return nil, err
		}
		receiptHex, err := encodeReceipt(receipts[i])
		if err != nil {
			return nil, err
		}
		blockTransactions[i] = txHex
		blockReceipts[i] = receiptHex
	}
	receiptRLP, proofNodes, err := buildReceiptTrieAndProof(blockReceipts, txIndex, header.ReceiptsRoot)
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
			{Name: "chainId", Value: header.ChainID, Consistent: true},
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
			{Name: "event.data", Value: canonicalHex(log.Data), Consistent: true},
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
			Data:    canonicalHex(log.Data),
		},
	}, nil
}

func buildOfflineStateFixture() (*StateProofPackage, error) {
	account := common.HexToAddress("0x4444444444444444444444444444444444444444")
	slotTarget := common.HexToHash("0x01")
	slotOther := common.HexToHash("0x02")
	slotValue := common.HexToHash("0x00000000000000000000000000000000000000000000000000000000deadbeef")
	slotOtherValue := common.HexToHash("0x00000000000000000000000000000000000000000000000000000000feedface")

	storageTrie := makeProofTrie()
	targetValueRLP, err := encodeStorageProofValue(slotValue)
	if err != nil {
		return nil, err
	}
	otherValueRLP, err := encodeStorageProofValue(slotOtherValue)
	if err != nil {
		return nil, err
	}
	if err := storageTrie.Update(crypto.Keccak256(slotTarget.Bytes()), targetValueRLP); err != nil {
		return nil, err
	}
	if err := storageTrie.Update(crypto.Keccak256(slotOther.Bytes()), otherValueRLP); err != nil {
		return nil, err
	}
	storageRoot := storageTrie.Hash()
	storageProofDB := memorydb.New()
	if err := storageTrie.Prove(crypto.Keccak256(slotTarget.Bytes()), storageProofDB); err != nil {
		return nil, err
	}
	storageProofNodes, err := dumpProofNodes(storageProofDB)
	if err != nil {
		return nil, err
	}

	accountState := types.StateAccount{
		Nonce:    7,
		Balance:  uint256.MustFromBig(big.NewInt(98_765_432_100)),
		Root:     storageRoot,
		CodeHash: crypto.Keccak256([]byte("offline-bytecode")),
	}
	accountRLPBytes, err := rlp.EncodeToBytes(&accountState)
	if err != nil {
		return nil, err
	}
	accountTrie := makeProofTrie()
	if err := accountTrie.Update(crypto.Keccak256(account.Bytes()), accountRLPBytes); err != nil {
		return nil, err
	}
	otherAccountAddress := common.HexToAddress("0x5555555555555555555555555555555555555555")
	otherAccountState := &types.StateAccount{
		Nonce:    1,
		Balance:  uint256.MustFromBig(big.NewInt(5)),
		Root:     types.EmptyRootHash,
		CodeHash: types.EmptyCodeHash.Bytes(),
	}
	otherAccountRLP, err := rlp.EncodeToBytes(otherAccountState)
	if err != nil {
		return nil, err
	}
	if err := accountTrie.Update(crypto.Keccak256(otherAccountAddress.Bytes()), otherAccountRLP); err != nil {
		return nil, err
	}
	stateRoot := accountTrie.Hash()
	accountProofDB := memorydb.New()
	if err := accountTrie.Prove(crypto.Keccak256(account.Bytes()), accountProofDB); err != nil {
		return nil, err
	}
	accountProofNodes, err := dumpProofNodes(accountProofDB)
	if err != nil {
		return nil, err
	}

	header := blockSnapshotHeader{
		ChainID:          "1",
		BlockNumber:      19_001_337,
		ParentHash:       crypto.Keccak256Hash([]byte("offline-parent-state")),
		StateRoot:        stateRoot,
		TransactionsRoot: types.EmptyTxsHash,
		ReceiptsRoot:     types.EmptyReceiptsHash,
	}
	ethHeader := &types.Header{
		ParentHash:  header.ParentHash,
		UncleHash:   types.EmptyUncleHash,
		Coinbase:    common.Address{},
		Root:        header.StateRoot,
		TxHash:      header.TransactionsRoot,
		ReceiptHash: header.ReceiptsRoot,
		Bloom:       types.Bloom{},
		Difficulty:  big.NewInt(1),
		Number:      new(big.Int).SetUint64(header.BlockNumber),
		GasLimit:    30_000_000,
		GasUsed:     0,
		Time:        1_721_000_000,
		Extra:       []byte("offline-state-fixture"),
		BaseFee:     big.NewInt(1_000_000_000),
	}
	header.BlockHash = ethHeader.Hash()

	digests, err := canonicalOfflineStateDigests(header, canonicalHex(accountRLPBytes), accountProofNodes, slotTarget, slotValue, storageProofNodes)
	if err != nil {
		return nil, err
	}
	consensus := sourceConsensus(
		"offline-fixture",
		nil,
		digests,
		[]ConsensusField{
			{Name: "chainId", Value: header.ChainID, Consistent: true},
			{Name: "blockNumber", Value: fmt.Sprintf("%d", header.BlockNumber), Consistent: true},
			{Name: "blockHash", Value: header.BlockHash.Hex(), Consistent: true},
			{Name: "parentHash", Value: header.ParentHash.Hex(), Consistent: true},
			{Name: "stateRoot", Value: header.StateRoot.Hex(), Consistent: true},
			{Name: "transactionsRoot", Value: header.TransactionsRoot.Hex(), Consistent: true},
			{Name: "receiptsRoot", Value: header.ReceiptsRoot.Hex(), Consistent: true},
			{Name: "account", Value: account.Hex(), Consistent: true},
			{Name: "slot", Value: slotTarget.Hex(), Consistent: true},
			{Name: "account.nonce", Value: "7", Consistent: true},
			{Name: "account.balance", Value: balanceHex(accountState.Balance.ToBig()), Consistent: true},
			{Name: "account.storageRoot", Value: storageRoot.Hex(), Consistent: true},
			{Name: "account.codeHash", Value: common.BytesToHash(accountState.CodeHash).Hex(), Consistent: true},
			{Name: "storage.value", Value: slotValue.Hex(), Consistent: true},
		},
	)
	return &StateProofPackage{
		Block:             buildBlockContext(header, consensus),
		Account:           account,
		Slot:              slotTarget,
		AccountRLP:        canonicalHex(accountRLPBytes),
		AccountProofNodes: accountProofNodes,
		AccountClaim: StateAccountClaim{
			Nonce:       accountState.Nonce,
			Balance:     balanceHex(accountState.Balance.ToBig()),
			StorageRoot: storageRoot,
			CodeHash:    common.BytesToHash(accountState.CodeHash),
		},
		StorageValue:      slotValue,
		StorageProofNodes: storageProofNodes,
	}, nil
}

func fixedPrivateKey(hexKey string) (*ecdsa.PrivateKey, error) {
	return crypto.HexToECDSA(hexKey)
}

func canonicalOfflineReceiptDigests(header blockSnapshotHeader, blockTransactions, blockReceipts []string, transactionRLP, receiptRLP string, log *types.Log) ([]ConsensusDigest, error) {
	headerDigest, err := canonicalDigest(header)
	if err != nil {
		return nil, err
	}
	blockTransactionsDigest, err := canonicalDigest(blockTransactions)
	if err != nil {
		return nil, err
	}
	blockReceiptsDigest, err := canonicalDigest(blockReceipts)
	if err != nil {
		return nil, err
	}
	targetDigest, err := canonicalDigest(struct {
		TransactionRLP string     `json:"transactionRlp"`
		ReceiptRLP     string     `json:"receiptRlp"`
		Event          EventClaim `json:"event"`
	}{
		TransactionRLP: transactionRLP,
		ReceiptRLP:     receiptRLP,
		Event: EventClaim{
			Address: log.Address,
			Topics:  append([]common.Hash(nil), log.Topics...),
			Data:    canonicalHex(log.Data),
		},
	})
	if err != nil {
		return nil, err
	}
	return []ConsensusDigest{
		{Name: "header", Digest: headerDigest},
		{Name: "blockTransactions", Digest: blockTransactionsDigest},
		{Name: "blockReceipts", Digest: blockReceiptsDigest},
		{Name: "targetReceipt", Digest: targetDigest},
	}, nil
}

func canonicalOfflineStateDigests(header blockSnapshotHeader, accountRLP string, accountProof []string, slot, slotValue common.Hash, storageProof []string) ([]ConsensusDigest, error) {
	headerDigest, err := canonicalDigest(header)
	if err != nil {
		return nil, err
	}
	accountDigest, err := canonicalDigest(struct {
		AccountRLP string   `json:"accountRlp"`
		Proof      []string `json:"proof"`
	}{
		AccountRLP: accountRLP,
		Proof:      accountProof,
	})
	if err != nil {
		return nil, err
	}
	storageDigest, err := canonicalDigest(struct {
		Slot  common.Hash `json:"slot"`
		Value common.Hash `json:"value"`
		Proof []string    `json:"proof"`
	}{
		Slot:  slot,
		Value: slotValue,
		Proof: storageProof,
	})
	if err != nil {
		return nil, err
	}
	return []ConsensusDigest{
		{Name: "header", Digest: headerDigest},
		{Name: "accountProof", Digest: accountDigest},
		{Name: "storageProof", Digest: storageDigest},
	}, nil
}

func offlineTransactionFields(txHash common.Hash, txIndex uint64, header blockSnapshotHeader) []ConsensusField {
	return []ConsensusField{
		{Name: "chainId", Value: header.ChainID, Consistent: true},
		{Name: "blockNumber", Value: fmt.Sprintf("%d", header.BlockNumber), Consistent: true},
		{Name: "blockHash", Value: header.BlockHash.Hex(), Consistent: true},
		{Name: "parentHash", Value: header.ParentHash.Hex(), Consistent: true},
		{Name: "stateRoot", Value: header.StateRoot.Hex(), Consistent: true},
		{Name: "transactionsRoot", Value: header.TransactionsRoot.Hex(), Consistent: true},
		{Name: "receiptsRoot", Value: header.ReceiptsRoot.Hex(), Consistent: true},
		{Name: "txHash", Value: txHash.Hex(), Consistent: true},
		{Name: "txIndex", Value: fmt.Sprintf("%d", txIndex), Consistent: true},
	}
}

func offlineTransactionDigests(header blockSnapshotHeader, blockTransactions []string, transactionRLP string) ([]ConsensusDigest, error) {
	headerDigest, err := canonicalDigest(header)
	if err != nil {
		return nil, err
	}
	blockTransactionsDigest, err := canonicalDigest(blockTransactions)
	if err != nil {
		return nil, err
	}
	targetTransactionDigest, err := canonicalDigest(transactionRLP)
	if err != nil {
		return nil, err
	}
	return []ConsensusDigest{
		{Name: "header", Digest: headerDigest},
		{Name: "blockTransactions", Digest: blockTransactionsDigest},
		{Name: "targetTransaction", Digest: targetTransactionDigest},
	}, nil
}
