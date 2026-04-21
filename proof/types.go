// Package proof generates and verifies Ethereum state, receipt, and transaction proofs.
package proof

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/holiman/uint256"
)

// BlockContext captures the block header fields that anchor a proof package.
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

// SourceConsensus records the normalized multi-RPC inputs used to build a proof.
type SourceConsensus struct {
	Mode    string            `json:"mode,omitempty"`
	RPCs    []string          `json:"rpcs"`
	Digests []ConsensusDigest `json:"digests"`
	Fields  []ConsensusField  `json:"fields"`
}

// ConsensusDigest stores a digest over a normalized proof input group.
type ConsensusDigest struct {
	Name   string      `json:"name"`
	Digest common.Hash `json:"digest"`
}

// ConsensusField stores a human-readable normalized field value used in consensus checks.
type ConsensusField struct {
	Name       string `json:"name"`
	Value      string `json:"value"`
	Consistent bool   `json:"consistent"`
}

// StateProofRequest describes the inputs required to generate a state proof.
type StateProofRequest struct {
	RPCURLs       []string
	MinRPCSources int
	BlockNumber   uint64
	Account       common.Address
	Slot          common.Hash
}

// ReceiptProofRequest describes the inputs required to generate a receipt proof.
type ReceiptProofRequest struct {
	RPCURLs       []string
	MinRPCSources int
	TxHash        common.Hash
	LogIndex      uint
}

// TransactionProofRequest describes the inputs required to generate a transaction proof.
type TransactionProofRequest struct {
	RPCURLs       []string
	MinRPCSources int
	TxHash        common.Hash
}

// VerifyRPCRequest describes the independent RPC set used for RPC-aware verification.
type VerifyRPCRequest struct {
	RPCURLs       []string
	MinRPCSources int
}

// StateAccountClaim is the decoded account claim embedded in a state proof package.
type StateAccountClaim struct {
	Nonce       uint64      `json:"nonce"`
	Balance     string      `json:"balance"`
	StorageRoot common.Hash `json:"storageRoot"`
	CodeHash    common.Hash `json:"codeHash"`
}

// StateProofPackage contains an account proof and one storage proof against stateRoot.
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

// EventClaim is the log payload claimed by a receipt proof package.
type EventClaim struct {
	Address common.Address `json:"address"`
	Topics  []common.Hash  `json:"topics"`
	Data    hexutil.Bytes  `json:"data"`
}

// ReceiptProofPackage contains a receipt inclusion proof plus the claimed log fields.
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

// TransactionProofPackage contains a transaction inclusion proof against transactionsRoot.
type TransactionProofPackage struct {
	Block          BlockContext    `json:"block"`
	TxHash         common.Hash     `json:"txHash"`
	TxIndex        uint64          `json:"txIndex"`
	TransactionRLP hexutil.Bytes   `json:"transactionRlp"`
	ProofNodes     []hexutil.Bytes `json:"proofNodes"`
}

// ReceiptExpectations adds optional caller-supplied assertions on top of a receipt proof package.
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
