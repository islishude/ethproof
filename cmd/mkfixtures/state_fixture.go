package main

import (
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethdb/memorydb"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/trie"
	"github.com/holiman/uint256"
	"github.com/islishude/ethproof/internal/proofutil"
)

func buildOfflineStateFixture() (*StateProofPackage, error) {
	account := common.HexToAddress("0x4444444444444444444444444444444444444444")
	slotTarget := common.HexToHash("0x01")
	slotOther := common.HexToHash("0x02")
	slotValue := common.HexToHash("0x00000000000000000000000000000000000000000000000000000000deadbeef")
	slotOtherValue := common.HexToHash("0x00000000000000000000000000000000000000000000000000000000feedface")

	storageTrie := proofutil.MakeProofTrie()
	targetValueRLP, err := proofutil.EncodeStorageProofValue(slotValue)
	if err != nil {
		return nil, err
	}
	otherValueRLP, err := proofutil.EncodeStorageProofValue(slotOtherValue)
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
	targetStorageProof, err := buildOfflineStorageProof(storageTrie, slotTarget, slotValue)
	if err != nil {
		return nil, err
	}
	otherStorageProof, err := buildOfflineStorageProof(storageTrie, slotOther, slotOtherValue)
	if err != nil {
		return nil, err
	}
	storageProofs := []StateStorageProof{targetStorageProof, otherStorageProof}

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

	accountTrie := proofutil.MakeProofTrie()
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
	accountProofNodes, err := proofutil.DumpProofNodes(accountProofDB)
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

	digests, err := canonicalOfflineStateDigests(header, proofutil.CanonicalBytes(accountRLPBytes), accountProofNodes, storageProofs)
	if err != nil {
		return nil, err
	}
	fields := []ConsensusField{
		{Name: "chainId", Value: proofutil.ChainIDString(header.ChainID), Consistent: true},
		{Name: "blockNumber", Value: fmt.Sprintf("%d", header.BlockNumber), Consistent: true},
		{Name: "blockHash", Value: header.BlockHash.Hex(), Consistent: true},
		{Name: "parentHash", Value: header.ParentHash.Hex(), Consistent: true},
		{Name: "stateRoot", Value: header.StateRoot.Hex(), Consistent: true},
		{Name: "transactionsRoot", Value: header.TransactionsRoot.Hex(), Consistent: true},
		{Name: "receiptsRoot", Value: header.ReceiptsRoot.Hex(), Consistent: true},
		{Name: "account", Value: account.Hex(), Consistent: true},
		{Name: "account.nonce", Value: "7", Consistent: true},
		{Name: "account.balance", Value: proofutil.BalanceHex(accountState.Balance.ToBig()), Consistent: true},
		{Name: "account.storageRoot", Value: storageRoot.Hex(), Consistent: true},
		{Name: "account.codeHash", Value: common.BytesToHash(accountState.CodeHash).Hex(), Consistent: true},
	}
	for i, storageProof := range storageProofs {
		fields = append(fields,
			ConsensusField{Name: fmt.Sprintf("storageProofs[%d].slot", i), Value: storageProof.Slot.Hex(), Consistent: true},
			ConsensusField{Name: fmt.Sprintf("storageProofs[%d].value", i), Value: storageProof.Value.Hex(), Consistent: true},
		)
	}
	consensus := sourceConsensus(
		"offline-fixture",
		nil,
		digests,
		fields,
	)
	return &StateProofPackage{
		Block:             buildBlockContext(header, consensus),
		Account:           account,
		AccountRLP:        proofutil.CanonicalBytes(accountRLPBytes),
		AccountProofNodes: accountProofNodes,
		AccountClaim: StateAccountClaim{
			Nonce:       accountState.Nonce,
			Balance:     proofutil.BalanceHex(accountState.Balance.ToBig()),
			StorageRoot: storageRoot,
			CodeHash:    common.BytesToHash(accountState.CodeHash),
		},
		StorageProofs: storageProofs,
	}, nil
}

func buildOfflineStorageProof(storageTrie *trie.Trie, slot common.Hash, value common.Hash) (StateStorageProof, error) {
	storageProofDB := memorydb.New()
	if err := storageTrie.Prove(crypto.Keccak256(slot.Bytes()), storageProofDB); err != nil {
		return StateStorageProof{}, err
	}
	storageProofNodes, err := proofutil.DumpProofNodes(storageProofDB)
	if err != nil {
		return StateStorageProof{}, err
	}
	return StateStorageProof{
		Slot:       slot,
		Value:      value,
		ProofNodes: storageProofNodes,
	}, nil
}
