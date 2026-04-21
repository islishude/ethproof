package proof

import (
	"context"

	"github.com/ethereum/go-ethereum/common"
)

type BlockContext struct {
	ChainID          string          `json:"chainId"`
	BlockNumber      uint64          `json:"blockNumber"`
	BlockHash        common.Hash     `json:"blockHash"`
	ParentHash       common.Hash     `json:"parentHash"`
	StateRoot        common.Hash     `json:"stateRoot"`
	TransactionsRoot common.Hash     `json:"transactionsRoot"`
	ReceiptsRoot     common.Hash     `json:"receiptsRoot"`
	SourceConsensus  SourceConsensus `json:"sourceConsensus"`
}

type SourceConsensus struct {
	Mode    string            `json:"mode,omitempty"`
	RPCs    []string          `json:"rpcs"`
	Digests []ConsensusDigest `json:"digests"`
	Fields  []ConsensusField  `json:"fields"`
}

type ConsensusDigest struct {
	Name   string      `json:"name"`
	Digest common.Hash `json:"digest"`
}

type ConsensusField struct {
	Name       string `json:"name"`
	Value      string `json:"value"`
	Consistent bool   `json:"consistent"`
}

type StateProofRequest struct {
	RPCURLs       []string
	MinRPCSources int
	BlockNumber   uint64
	Account       common.Address
	Slot          common.Hash
}

type ReceiptProofRequest struct {
	RPCURLs       []string
	MinRPCSources int
	TxHash        common.Hash
	LogIndex      uint
}

type TransactionProofRequest struct {
	RPCURLs       []string
	MinRPCSources int
	TxHash        common.Hash
}

type StateAccountClaim struct {
	Nonce       uint64      `json:"nonce"`
	Balance     string      `json:"balance"`
	StorageRoot common.Hash `json:"storageRoot"`
	CodeHash    common.Hash `json:"codeHash"`
}

type StateProofPackage struct {
	Block             BlockContext      `json:"block"`
	Account           common.Address    `json:"account"`
	Slot              common.Hash       `json:"slot"`
	AccountRLP        string            `json:"accountRlp"`
	AccountProofNodes []string          `json:"accountProofNodes"`
	AccountClaim      StateAccountClaim `json:"accountClaim"`
	StorageValue      common.Hash       `json:"storageValue"`
	StorageProofNodes []string          `json:"storageProofNodes"`
}

type EventClaim struct {
	Address common.Address `json:"address"`
	Topics  []common.Hash  `json:"topics"`
	Data    string         `json:"data"`
}

type ReceiptProofPackage struct {
	Block          BlockContext `json:"block"`
	TxHash         common.Hash  `json:"txHash"`
	TxIndex        uint64       `json:"txIndex"`
	LogIndex       uint         `json:"logIndex"`
	TransactionRLP string       `json:"transactionRlp"`
	ReceiptRLP     string       `json:"receiptRlp"`
	ProofNodes     []string     `json:"proofNodes"`
	Event          EventClaim   `json:"event"`
}

type TransactionProofPackage struct {
	Block          BlockContext `json:"block"`
	TxHash         common.Hash  `json:"txHash"`
	TxIndex        uint64       `json:"txIndex"`
	TransactionRLP string       `json:"transactionRlp"`
	ProofNodes     []string     `json:"proofNodes"`
}

type ReceiptExpectations struct {
	Emitter *common.Address
	Topics  []common.Hash
	Data    []byte
}

type OfflineFixtures struct {
	State       *StateProofPackage       `json:"state"`
	Receipt     *ReceiptProofPackage     `json:"receipt"`
	Transaction *TransactionProofPackage `json:"transaction"`
}

type accountSnapshot struct {
	Header       blockSnapshotHeader `json:"header"`
	Account      common.Address      `json:"account"`
	Slot         common.Hash         `json:"slot"`
	AccountRLP   string              `json:"accountRlp"`
	AccountProof []string            `json:"accountProof"`
	AccountClaim StateAccountClaim   `json:"accountClaim"`
	StorageValue common.Hash         `json:"storageValue"`
	StorageProof []string            `json:"storageProof"`
}

type receiptSnapshot struct {
	Header            blockSnapshotHeader `json:"header"`
	TxHash            common.Hash         `json:"txHash"`
	TxIndex           uint64              `json:"txIndex"`
	LogIndex          uint                `json:"logIndex"`
	TransactionRLP    string              `json:"transactionRlp"`
	ReceiptRLP        string              `json:"receiptRlp"`
	BlockTransactions []string            `json:"blockTransactions"`
	BlockReceipts     []string            `json:"blockReceipts"`
	Event             EventClaim          `json:"event"`
}

type transactionSnapshot struct {
	Header            blockSnapshotHeader `json:"header"`
	TxHash            common.Hash         `json:"txHash"`
	TxIndex           uint64              `json:"txIndex"`
	TransactionRLP    string              `json:"transactionRlp"`
	BlockTransactions []string            `json:"blockTransactions"`
}

type blockSnapshotHeader struct {
	ChainID          string      `json:"chainId"`
	BlockNumber      uint64      `json:"blockNumber"`
	BlockHash        common.Hash `json:"blockHash"`
	ParentHash       common.Hash `json:"parentHash"`
	StateRoot        common.Hash `json:"stateRoot"`
	TransactionsRoot common.Hash `json:"transactionsRoot"`
	ReceiptsRoot     common.Hash `json:"receiptsRoot"`
}

func GenerateStateProof(ctx context.Context, req StateProofRequest) (*StateProofPackage, error) {
	return generateStateProof(ctx, req)
}

func GenerateReceiptProof(ctx context.Context, req ReceiptProofRequest) (*ReceiptProofPackage, error) {
	return generateReceiptProof(ctx, req)
}

func GenerateTransactionProof(ctx context.Context, req TransactionProofRequest) (*TransactionProofPackage, error) {
	return generateTransactionProof(ctx, req)
}

func VerifyStateProofPackage(pkg *StateProofPackage) error {
	return verifyStateProofPackage(pkg)
}

func VerifyReceiptProofPackage(pkg *ReceiptProofPackage) error {
	return verifyReceiptProofPackage(pkg, nil)
}

func VerifyReceiptProofPackageWithExpectations(pkg *ReceiptProofPackage, expect *ReceiptExpectations) error {
	return verifyReceiptProofPackage(pkg, expect)
}

func VerifyTransactionProofPackage(pkg *TransactionProofPackage) error {
	return verifyTransactionProofPackage(pkg)
}
