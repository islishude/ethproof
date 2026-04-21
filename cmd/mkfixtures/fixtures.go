package main

import (
	"bytes"
	"crypto/ecdsa"
	"encoding/json"
	"fmt"
	"math/big"
	"sort"

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
	"github.com/islishude/ethproof/proof"
)

type StateProofPackage = proof.StateProofPackage
type ReceiptProofPackage = proof.ReceiptProofPackage
type TransactionProofPackage = proof.TransactionProofPackage
type StateAccountClaim = proof.StateAccountClaim
type EventClaim = proof.EventClaim
type SourceConsensus = proof.SourceConsensus
type ConsensusDigest = proof.ConsensusDigest
type ConsensusField = proof.ConsensusField

type OfflineFixtures struct {
	State       *StateProofPackage       `json:"state"`
	Receipt     *ReceiptProofPackage     `json:"receipt"`
	Transaction *TransactionProofPackage `json:"transaction"`
}

type blockSnapshotHeader struct {
	ChainID          *uint256.Int `json:"chainId"`
	BlockNumber      uint64       `json:"blockNumber"`
	BlockHash        common.Hash  `json:"blockHash"`
	ParentHash       common.Hash  `json:"parentHash"`
	StateRoot        common.Hash  `json:"stateRoot"`
	TransactionsRoot common.Hash  `json:"transactionsRoot"`
	ReceiptsRoot     common.Hash  `json:"receiptsRoot"`
}

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
	blockTransactions := make([]hexutil.Bytes, len(txs))
	blockReceipts := make([]hexutil.Bytes, len(receipts))
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
			{Name: "chainId", Value: chainIDString(header.ChainID), Consistent: true},
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
			Data:    canonicalBytes(log.Data),
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
		ChainID:          uint256.NewInt(1),
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

	digests, err := canonicalOfflineStateDigests(header, canonicalBytes(accountRLPBytes), accountProofNodes, slotTarget, slotValue, storageProofNodes)
	if err != nil {
		return nil, err
	}
	consensus := sourceConsensus(
		"offline-fixture",
		nil,
		digests,
		[]ConsensusField{
			{Name: "chainId", Value: chainIDString(header.ChainID), Consistent: true},
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
		AccountRLP:        canonicalBytes(accountRLPBytes),
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

func canonicalOfflineReceiptDigests(header blockSnapshotHeader, blockTransactions, blockReceipts []hexutil.Bytes, transactionRLP, receiptRLP hexutil.Bytes, log *types.Log) ([]ConsensusDigest, error) {
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
		TransactionRLP hexutil.Bytes `json:"transactionRlp"`
		ReceiptRLP     hexutil.Bytes `json:"receiptRlp"`
		Event          EventClaim    `json:"event"`
	}{
		TransactionRLP: transactionRLP,
		ReceiptRLP:     receiptRLP,
		Event: EventClaim{
			Address: log.Address,
			Topics:  append([]common.Hash(nil), log.Topics...),
			Data:    canonicalBytes(log.Data),
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

func canonicalOfflineStateDigests(header blockSnapshotHeader, accountRLP hexutil.Bytes, accountProof []hexutil.Bytes, slot, slotValue common.Hash, storageProof []hexutil.Bytes) ([]ConsensusDigest, error) {
	headerDigest, err := canonicalDigest(header)
	if err != nil {
		return nil, err
	}
	accountDigest, err := canonicalDigest(struct {
		AccountRLP hexutil.Bytes   `json:"accountRlp"`
		Proof      []hexutil.Bytes `json:"proof"`
	}{
		AccountRLP: accountRLP,
		Proof:      accountProof,
	})
	if err != nil {
		return nil, err
	}
	storageDigest, err := canonicalDigest(struct {
		Slot  common.Hash     `json:"slot"`
		Value common.Hash     `json:"value"`
		Proof []hexutil.Bytes `json:"proof"`
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
		{Name: "chainId", Value: chainIDString(header.ChainID), Consistent: true},
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

func offlineTransactionDigests(header blockSnapshotHeader, blockTransactions []hexutil.Bytes, transactionRLP hexutil.Bytes) ([]ConsensusDigest, error) {
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

func chainIDString(v *uint256.Int) string {
	if v == nil {
		return "0"
	}
	return v.String()
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

func encodeStorageProofValue(value common.Hash) ([]byte, error) {
	if value == (common.Hash{}) {
		return nil, nil
	}
	return rlp.EncodeToBytes(common.TrimLeftZeroes(value[:]))
}

func buildBlockContext(header blockSnapshotHeader, consensus SourceConsensus) proof.BlockContext {
	return proof.BlockContext{
		ChainID:          header.ChainID.Clone(),
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

func transactionSnapshotFromBlock(header blockSnapshotHeader, txs types.Transactions, txIndex uint64, consensus SourceConsensus) (*TransactionProofPackage, error) {
	if int(txIndex) >= len(txs) {
		return nil, fmt.Errorf("transaction index %d out of range", txIndex)
	}
	blockTransactions := make([]hexutil.Bytes, len(txs))
	for i, tx := range txs {
		encoded, err := encodeTransaction(tx)
		if err != nil {
			return nil, fmt.Errorf("encode transaction %d: %w", i, err)
		}
		blockTransactions[i] = encoded
	}
	transactionRLP, proofNodes, err := buildTransactionTrieAndProof(blockTransactions, txIndex, header.TransactionsRoot)
	if err != nil {
		return nil, err
	}
	targetTx, _, err := decodeTransaction(transactionRLP)
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
