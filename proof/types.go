// Package proof generates and verifies Ethereum state, receipt, and transaction proofs.
package proof

import (
	"encoding/json"
	"time"

	bt "github.com/ethereum/go-ethereum/beacon/types"
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

// FinalityHeader is a post-Capella Beacon API light-client header. Execution is
// kept in its original JSON form so the cache preserves all fork-specific fields.
type FinalityHeader struct {
	Beacon          bt.Header       `json:"beacon"`
	Execution       json.RawMessage `json:"execution"`
	ExecutionBranch []common.Hash   `json:"execution_branch"`
}

// FinalityBootstrap authenticates the starting sync committee against a trusted root.
type FinalityBootstrap struct {
	Header    FinalityHeader              `json:"header"`
	Committee *bt.SerializedSyncCommittee `json:"current_sync_committee"`
	Branch    []common.Hash               `json:"current_sync_committee_branch"`
}

// FinalityUpdate represents both Beacon API update and finality_update data.
// A zero branch denotes an absent optional proof, as in the consensus specification.
type FinalityUpdate struct {
	Attested       FinalityHeader              `json:"attested_header"`
	Next           *bt.SerializedSyncCommittee `json:"next_sync_committee,omitempty"`
	NextBranch     []common.Hash               `json:"next_sync_committee_branch,omitempty"`
	Finalized      *FinalityHeader             `json:"finalized_header,omitempty"`
	FinalityBranch []common.Hash               `json:"finality_branch"`
	Aggregate      bt.SyncAggregate            `json:"sync_aggregate"`
	SignatureSlot  common.Decimal              `json:"signature_slot"`
}

// FinalityConfig configures online finality authentication, independently of
// proof inputs and of the execution RPC. Checkpoint providers are trust inputs.
type FinalityConfig struct {
	Network        string   `json:"network"`
	BeaconRPC      string   `json:"beaconRpc"`
	CheckpointRPCs []string `json:"checkpointRpcs"`
	CacheDir       string   `json:"cacheDir,omitempty"`
	MaxAncestry    uint64   `json:"maxAncestry,omitempty"`
	TimeoutSeconds uint64   `json:"timeoutSeconds,omitempty"`
}

// FinalityDependencies allows embedders to control transport, persistence and
// time. Injected sources and stores remain caller-owned. Now must be trustworthy.
type FinalityDependencies struct {
	Beacon        BeaconFinalitySource
	Checkpoints   []CheckpointSource
	Store         FinalityStore
	Now           func() time.Time
	OnTrustChange func(FinalityTrustEvent)
}

// FinalityCheckpoint is an external trust statement, not a cryptographic proof.
type FinalityCheckpoint struct {
	Epoch common.Decimal `json:"epoch"`
	Root  common.Hash    `json:"root"`
}

// CheckpointHeader carries a provider's canonical/finalized assertions.
type CheckpointHeader struct {
	Header              bt.Header
	Canonical           bool
	Finalized           bool
	ExecutionOptimistic bool
}

// FinalityTrustEvent records an externally trusted checkpoint acquisition.
// It contains only opaque source identifiers, never endpoint credentials.
type FinalityTrustEvent struct {
	PreviousRoot common.Hash `json:"previousRoot"`
	Root         common.Hash `json:"root"`
	Epoch        uint64      `json:"epoch"`
	Slot         uint64      `json:"slot"`
	AcceptedAt   time.Time   `json:"acceptedAt"`
	Sources      []string    `json:"sources"`
}
