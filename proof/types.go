// Package proof generates and verifies Ethereum state, receipt, and transaction proofs.
package proof

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/holiman/uint256"
)

// BlockContext captures the block header fields embedded in a proof package. Callers must compare
// it with a trusted BlockAnchor or a caller-trusted RPC to authenticate the embedded roots.
type BlockContext struct {
	ChainID          *uint256.Int `json:"chainId"`
	BlockNumber      uint64       `json:"blockNumber"`
	BlockHash        common.Hash  `json:"blockHash"`
	ParentHash       common.Hash  `json:"parentHash"`
	StateRoot        common.Hash  `json:"stateRoot"`
	TransactionsRoot common.Hash  `json:"transactionsRoot"`
	ReceiptsRoot     common.Hash  `json:"receiptsRoot"`
}

// BlockAnchor is a caller-trusted block header projection used to authenticate embedded proof roots.
type BlockAnchor BlockContext

// StateProofRequest describes the inputs required to generate a state proof.
type StateProofRequest struct {
	BlockNumber uint64
	Account     common.Address
	Slots       []common.Hash
}

// ReceiptProofRequest describes the inputs required to generate a receipt proof.
type ReceiptProofRequest struct {
	TxHash   common.Hash
	LogIndex uint64
}

// TransactionProofRequest describes the inputs required to generate a transaction proof.
type TransactionProofRequest struct {
	TxHash common.Hash
}

// StateAccountClaim is the decoded account claim embedded in a state proof package.
type StateAccountClaim struct {
	Nonce       uint64      `json:"nonce"`
	Balance     string      `json:"balance"`
	StorageRoot common.Hash `json:"storageRoot"`
	CodeHash    common.Hash `json:"codeHash"`
}

// StateStorageProof contains one storage proof against the account storage root.
type StateStorageProof struct {
	Slot       common.Hash     `json:"slot"`
	Value      common.Hash     `json:"value"`
	ProofNodes []hexutil.Bytes `json:"proofNodes"`
}

// StateProofPackage contains an account proof and one or more storage proofs against stateRoot.
type StateProofPackage struct {
	Block             BlockContext        `json:"block"`
	Account           common.Address      `json:"account"`
	AccountRLP        hexutil.Bytes       `json:"accountRlp"`
	AccountProofNodes []hexutil.Bytes     `json:"accountProofNodes"`
	AccountClaim      StateAccountClaim   `json:"accountClaim"`
	StorageProofs     []StateStorageProof `json:"storageProofs"`
}

// EventClaim is the log payload claimed by a receipt proof package.
type EventClaim struct {
	Address common.Address `json:"address"`
	Topics  []common.Hash  `json:"topics"`
	Data    hexutil.Bytes  `json:"data"`
}

// ReceiptProofPackage proves both transaction and receipt inclusion at TxIndex plus the claimed log fields.
type ReceiptProofPackage struct {
	Block                 BlockContext    `json:"block"`
	TxHash                common.Hash     `json:"txHash"`
	TxIndex               uint64          `json:"txIndex"`
	LogIndex              uint64          `json:"logIndex"`
	TransactionRLP        hexutil.Bytes   `json:"transactionRlp"`
	TransactionProofNodes []hexutil.Bytes `json:"transactionProofNodes"`
	ReceiptRLP            hexutil.Bytes   `json:"receiptRlp"`
	ProofNodes            []hexutil.Bytes `json:"proofNodes"`
	Event                 EventClaim      `json:"event"`
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
