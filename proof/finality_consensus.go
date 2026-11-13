package proof

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/bits"
	"strconv"

	"github.com/attestantio/go-eth2-client/spec/capella"
	"github.com/attestantio/go-eth2-client/spec/deneb"
	"github.com/ethereum/go-ethereum/beacon/merkle"
	bp "github.com/ethereum/go-ethereum/beacon/params"
	bt "github.com/ethereum/go-ethereum/beacon/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"
)

type finalityState struct {
	Finalized FinalityHeader              `json:"finalized"`
	Current   *bt.SerializedSyncCommittee `json:"current"`
	Next      *bt.SerializedSyncCommittee `json:"next,omitempty"`
}

var (
	ErrNotFinalized          = errors.New("target block is not finalized")
	ErrFinalityConflict      = errors.New("conflicting finalized history")
	ErrFinalityCache         = errors.New("invalid finality cache")
	ErrFinalityStale         = errors.New("finality checkpoint is stale")
	ErrFinalityUnsupported   = errors.New("unsupported consensus fork")
	ErrFinalityParticipation = errors.New("insufficient sync committee participation")
	ErrFinalityHistory       = errors.New("finality ancestry limit exceeded")
)

func supportedFinalityFork(name string) bool {
	switch name {
	case "capella", "deneb", "electra", "fulu":
		return true
	}
	return false
}

func verifyFinalityHeader(h FinalityHeader, network *bp.ChainConfig) (common.Hash, error) {
	fork := network.ForkNameAtSlot(h.Beacon.Slot)
	if !supportedFinalityFork(fork) {
		return common.Hash{}, fmt.Errorf("%w: %s", ErrFinalityUnsupported, fork)
	}
	if fork == "capella" {
		// Later wire formats can upgrade a Capella header, but blob fields must stay zero.
		var blob struct {
			Used   common.Decimal `json:"blob_gas_used"`
			Excess common.Decimal `json:"excess_blob_gas"`
		}
		if err := json.Unmarshal(h.Execution, &blob); err != nil {
			return common.Hash{}, errors.New("invalid execution header JSON")
		}
		if blob.Used != 0 || blob.Excess != 0 {
			return common.Hash{}, errors.New("nonzero blob fields before Deneb")
		}
	}
	payload, err := bt.ExecutionHeaderFromJSON(fork, h.Execution)
	if err != nil {
		// Deferred decoding happens outside the HTTP sanitizer. Parser errors
		// can quote response data, including credentials echoed by a provider.
		return common.Hash{}, errors.New("invalid execution header JSON")
	}
	if err := verifyFinalityBranch(h.Beacon.BodyRoot, 25, h.ExecutionBranch, common.Hash(payload.PayloadRoot()), false); err != nil {
		return common.Hash{}, fmt.Errorf("execution branch: %w", err)
	}
	if payload.BlockHash() == (common.Hash{}) {
		return common.Hash{}, errors.New("zero execution block hash")
	}
	return payload.BlockHash(), nil
}

func verifyFinalityBranch(root common.Hash, index uint64, branch []common.Hash, leaf common.Hash, normalized bool) error {
	depth := bits.Len64(index) - 1
	if depth < 0 || len(branch) > 7 || (!normalized && len(branch) != depth) {
		return errors.New("invalid Merkle branch length")
	}
	values := make(merkle.Values, len(branch))
	for i := range branch {
		values[i] = merkle.Value(branch[i])
	}
	if normalized {
		return merkle.VerifyNormalizedProof(root, index, values, merkle.Value(leaf))
	}
	return merkle.VerifyProof(root, index, values, merkle.Value(leaf))
}

func hasFinalityBranch(branch []common.Hash) bool {
	for _, h := range branch {
		if h != (common.Hash{}) {
			return true
		}
	}
	return false
}

func bootstrapFinality(b FinalityBootstrap, root common.Hash, now uint64, net *bp.ChainConfig) (*finalityState, error) {
	if b.Header.Beacon.Hash() != root {
		return nil, errors.New("bootstrap root mismatch")
	}
	if b.Header.Beacon.Slot > now {
		return nil, errors.New("future bootstrap slot")
	}
	if b.Committee == nil {
		return nil, errors.New("missing bootstrap committee")
	}
	if _, err := verifyFinalityHeader(b.Header, net); err != nil {
		return nil, err
	}
	index := bp.StateIndexSyncCommittee(net.ForkNameAtSlot(b.Header.Beacon.Slot))
	if err := verifyFinalityBranch(b.Header.Beacon.StateRoot, index, b.Branch, b.Committee.Root(), true); err != nil {
		return nil, fmt.Errorf("bootstrap committee: %w", err)
	}
	if _, err := b.Committee.Deserialize(); err != nil {
		return nil, fmt.Errorf("bootstrap BLS keys: %w", err)
	}
	return &finalityState{Finalized: b.Header, Current: b.Committee}, nil
}

// validateUpdate has no side effects. Applying a verified update is a separate,
// supermajority-only step. No optimistic or forced-update state is maintained.
func (s *finalityState) validateUpdate(u FinalityUpdate, now uint64, net *bp.ChainConfig) error {
	if s.Current == nil {
		return errors.New("missing current committee")
	}
	sig := uint64(u.SignatureSlot)
	att := u.Attested.Beacon.Slot
	if sig > now || sig <= att {
		return errors.New("invalid update signature slot")
	}
	if !supportedFinalityFork(net.ForkNameAtSlot(sig - 1)) {
		return ErrFinalityUnsupported
	}
	if _, err := verifyFinalityHeader(u.Attested, net); err != nil {
		return err
	}
	period, signPeriod := s.Finalized.Beacon.SyncPeriod(), bt.SyncPeriod(sig)
	if signPeriod != period && (signPeriod != period+1 || s.Next == nil) {
		return errors.New("update skips an unknown sync committee")
	}
	if !s.updateRelevant(u) {
		return errors.New("update is not relevant")
	}

	if err := s.validateUpdateProofs(u, net); err != nil {
		return err
	}
	committee := s.Current
	if signPeriod == period+1 {
		committee = s.Next
	}
	keys, err := committee.Deserialize()
	if err != nil {
		return fmt.Errorf("committee BLS keys: %w", err)
	}
	signingRoot, err := net.Forks.SigningRoot((sig-1)/bp.EpochLength, u.Attested.Beacon.Hash())
	if err != nil {
		return err
	}
	if u.Aggregate.SignerCount() == 0 || !keys.VerifySignature(signingRoot, &u.Aggregate) {
		return errors.New("invalid sync committee signature")
	}
	return nil
}

func (s *finalityState) validateUpdateProofs(u FinalityUpdate, net *bp.ChainConfig) error {
	fork := net.ForkNameAtSlot(u.Attested.Beacon.Slot)
	if hasFinalityBranch(u.FinalityBranch) {
		if u.Finalized == nil || u.Finalized.Beacon.Slot > u.Attested.Beacon.Slot {
			return errors.New("invalid finalized slot")
		}
		root := u.Finalized.Beacon.Hash()
		if u.Finalized.Beacon.Slot == 0 {
			// The genesis checkpoint is represented by an empty light-client
			// header and a zero leaf, not hash_tree_root(BeaconBlockHeader()).
			if !emptyFinalityHeader(*u.Finalized, fork) {
				return errors.New("nonempty genesis finalized header")
			}
			root = common.Hash{}
		} else if _, err := verifyFinalityHeader(*u.Finalized, net); err != nil {
			return err
		}
		if err := verifyFinalityBranch(u.Attested.Beacon.StateRoot, bp.StateIndexFinalBlock(fork), u.FinalityBranch, root, true); err != nil {
			return fmt.Errorf("finality branch: %w", err)
		}
		if u.Finalized.Beacon.Slot == s.Finalized.Beacon.Slot && root != s.Finalized.Beacon.Hash() {
			return ErrFinalityConflict
		}
	} else if u.Finalized != nil && u.Finalized.Beacon != (bt.Header{}) {
		return errors.New("finalized header without proof")
	}
	if hasFinalityBranch(u.NextBranch) {
		if u.Next == nil {
			return errors.New("missing next committee")
		}
		if err := verifyFinalityBranch(u.Attested.Beacon.StateRoot, bp.StateIndexNextSyncCommittee(fork), u.NextBranch, u.Next.Root(), true); err != nil {
			return fmt.Errorf("next committee branch: %w", err)
		}
		if u.Attested.Beacon.SyncPeriod() == s.Finalized.Beacon.SyncPeriod() && s.Next != nil && *u.Next != *s.Next {
			return errors.New("next committee mismatch")
		}
		if _, err := u.Next.Deserialize(); err != nil {
			return fmt.Errorf("next committee BLS keys: %w", err)
		}
	} else if u.Next != nil && *u.Next != (bt.SerializedSyncCommittee{}) {
		return errors.New("next committee without proof")
	}
	return nil
}

// This sentinel is valid only inside an update's finalized-checkpoint proof.
// verifyFinalityHeader still rejects it as a bootstrap or execution anchor.
func emptyFinalityHeader(h FinalityHeader, fork string) bool {
	if h.Beacon != (bt.Header{}) || len(h.ExecutionBranch) != 4 || hasFinalityBranch(h.ExecutionBranch) {
		return false
	}
	payload, err := bt.ExecutionHeaderFromJSON(fork, h.Execution)
	if err != nil {
		return false
	}
	var empty *bt.ExecutionHeader
	if fork == "capella" {
		var blob struct {
			Used   common.Decimal `json:"blob_gas_used"`
			Excess common.Decimal `json:"excess_blob_gas"`
		}
		if json.Unmarshal(h.Execution, &blob) != nil || blob.Used != 0 || blob.Excess != 0 {
			return false
		}
		empty = bt.NewExecutionHeader(&capella.ExecutionPayloadHeader{})
	} else {
		empty = bt.NewExecutionHeader(&deneb.ExecutionPayloadHeader{BaseFeePerGas: new(uint256.Int)})
	}
	return payload.PayloadRoot() == empty.PayloadRoot()
}

func (s *finalityState) processUpdate(u FinalityUpdate, now uint64, net *bp.ChainConfig) (bool, error) {
	if err := s.validateUpdate(u, now, net); err != nil {
		return false, err
	}
	if u.Aggregate.SignerCount() < bp.SyncCommitteeSupermajority {
		return false, ErrFinalityParticipation
	}
	if !hasFinalityBranch(u.FinalityBranch) || u.Finalized == nil {
		return false, nil
	}
	period := s.Finalized.Beacon.SyncPeriod()
	finalizedPeriod := u.Finalized.Beacon.SyncPeriod()
	hasNext := hasFinalityBranch(u.NextBranch)
	learnNext := s.Next == nil && hasNext && finalizedPeriod == u.Attested.Beacon.SyncPeriod()
	if u.Finalized.Beacon.Slot <= s.Finalized.Beacon.Slot && !learnNext {
		return false, nil
	}
	if s.Next == nil {
		if finalizedPeriod != period {
			return false, errors.New("cannot rotate unknown committee")
		}
		if hasNext {
			s.Next = u.Next
		}
	} else if finalizedPeriod == period+1 {
		s.Current = s.Next
		s.Next = nil
		if hasNext {
			s.Next = u.Next
		}
	}
	if u.Finalized.Beacon.Slot > s.Finalized.Beacon.Slot {
		s.Finalized = *u.Finalized
	}
	return true, nil
}

func (s *finalityState) updateRelevant(u FinalityUpdate) bool {
	return u.Attested.Beacon.Slot > s.Finalized.Beacon.Slot || (hasFinalityBranch(u.NextBranch) && s.Next == nil && u.Attested.Beacon.SyncPeriod() == s.Finalized.Beacon.SyncPeriod())
}

// geth's Beacon Header decoder requires decimal strings, while its encoder
// emits numbers. Use the Beacon API representation explicitly for cache round trips.
func (h FinalityHeader) MarshalJSON() ([]byte, error) {
	type header struct {
		Slot     string      `json:"slot"`
		Proposer string      `json:"proposer_index"`
		Parent   common.Hash `json:"parent_root"`
		State    common.Hash `json:"state_root"`
		Body     common.Hash `json:"body_root"`
	}
	return json.Marshal(struct {
		Beacon    header          `json:"beacon"`
		Execution json.RawMessage `json:"execution"`
		Branch    []common.Hash   `json:"execution_branch"`
	}{header{strconv.FormatUint(h.Beacon.Slot, 10), strconv.FormatUint(h.Beacon.ProposerIndex, 10), h.Beacon.ParentRoot, h.Beacon.StateRoot, h.Beacon.BodyRoot}, h.Execution, h.ExecutionBranch})
}

// MarshalJSON preserves the decimal-string representation required by Beacon APIs.
func (u FinalityUpdate) MarshalJSON() ([]byte, error) {
	type plain FinalityUpdate
	return json.Marshal(struct {
		plain
		Signature string `json:"signature_slot"`
	}{plain(u), strconv.FormatUint(uint64(u.SignatureSlot), 10)})
}
