package proof

import (
	"context"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/holiman/uint256"
)

type BlockContext struct {
	ChainID          *uint256.Int    `json:"chainId"`
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

type VerifyRPCRequest struct {
	RPCURLs       []string
	MinRPCSources int
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
	AccountRLP        hexutil.Bytes     `json:"accountRlp"`
	AccountProofNodes []hexutil.Bytes   `json:"accountProofNodes"`
	AccountClaim      StateAccountClaim `json:"accountClaim"`
	StorageValue      common.Hash       `json:"storageValue"`
	StorageProofNodes []hexutil.Bytes   `json:"storageProofNodes"`
}

type EventClaim struct {
	Address common.Address `json:"address"`
	Topics  []common.Hash  `json:"topics"`
	Data    hexutil.Bytes  `json:"data"`
}

type ReceiptProofPackage struct {
	Block          BlockContext    `json:"block"`
	TxHash         common.Hash     `json:"txHash"`
	TxIndex        uint64          `json:"txIndex"`
	LogIndex       uint            `json:"logIndex"`
	TransactionRLP hexutil.Bytes   `json:"transactionRlp"`
	ReceiptRLP     hexutil.Bytes   `json:"receiptRlp"`
	ProofNodes     []hexutil.Bytes `json:"proofNodes"`
	Event          EventClaim      `json:"event"`
}

type TransactionProofPackage struct {
	Block          BlockContext    `json:"block"`
	TxHash         common.Hash     `json:"txHash"`
	TxIndex        uint64          `json:"txIndex"`
	TransactionRLP hexutil.Bytes   `json:"transactionRlp"`
	ProofNodes     []hexutil.Bytes `json:"proofNodes"`
}

type ReceiptExpectations struct {
	Emitter *common.Address
	Topics  []common.Hash
	Data    []byte
}

type accountSnapshot struct {
	Header       blockSnapshotHeader `json:"header"`
	Account      common.Address      `json:"account"`
	Slot         common.Hash         `json:"slot"`
	AccountRLP   hexutil.Bytes       `json:"accountRlp"`
	AccountProof []hexutil.Bytes     `json:"accountProof"`
	AccountClaim StateAccountClaim   `json:"accountClaim"`
	StorageValue common.Hash         `json:"storageValue"`
	StorageProof []hexutil.Bytes     `json:"storageProof"`
}

type receiptSnapshot struct {
	Header            blockSnapshotHeader `json:"header"`
	TxHash            common.Hash         `json:"txHash"`
	TxIndex           uint64              `json:"txIndex"`
	LogIndex          uint                `json:"logIndex"`
	TransactionRLP    hexutil.Bytes       `json:"transactionRlp"`
	ReceiptRLP        hexutil.Bytes       `json:"receiptRlp"`
	BlockTransactions []hexutil.Bytes     `json:"blockTransactions"`
	BlockReceipts     []hexutil.Bytes     `json:"blockReceipts"`
	Event             EventClaim          `json:"event"`
}

type transactionSnapshot struct {
	Header            blockSnapshotHeader `json:"header"`
	TxHash            common.Hash         `json:"txHash"`
	TxIndex           uint64              `json:"txIndex"`
	TransactionRLP    hexutil.Bytes       `json:"transactionRlp"`
	BlockTransactions []hexutil.Bytes     `json:"blockTransactions"`
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

func VerifyStateProofPackageAgainstRPCs(ctx context.Context, pkg *StateProofPackage, req VerifyRPCRequest) error {
	return verifyStateProofPackageAgainstRPCs(ctx, pkg, req, fetchBlockHeadersFromRPCs)
}

func VerifyReceiptProofPackage(pkg *ReceiptProofPackage) error {
	return verifyReceiptProofPackage(pkg, nil)
}

func VerifyReceiptProofPackageWithExpectations(pkg *ReceiptProofPackage, expect *ReceiptExpectations) error {
	return verifyReceiptProofPackage(pkg, expect)
}

func VerifyReceiptProofPackageWithExpectationsAgainstRPCs(ctx context.Context, pkg *ReceiptProofPackage, expect *ReceiptExpectations, req VerifyRPCRequest) error {
	return verifyReceiptProofPackageWithExpectationsAgainstRPCs(ctx, pkg, expect, req, fetchBlockHeadersFromRPCs)
}

func VerifyTransactionProofPackage(pkg *TransactionProofPackage) error {
	return verifyTransactionProofPackage(pkg)
}

func VerifyTransactionProofPackageAgainstRPCs(ctx context.Context, pkg *TransactionProofPackage, req VerifyRPCRequest) error {
	return verifyTransactionProofPackageAgainstRPCs(ctx, pkg, req, fetchBlockHeadersFromRPCs)
}
