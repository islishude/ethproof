package proof

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"os"
	"testing"

	"github.com/attestantio/go-eth2-client/spec/capella"
	"github.com/attestantio/go-eth2-client/spec/deneb"
	bp "github.com/ethereum/go-ethereum/beacon/params"
	bt "github.com/ethereum/go-ethereum/beacon/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"
	bls "github.com/protolambda/bls12-381-util"
)

func testFinalityNetwork() *bp.ChainConfig {
	return (&bp.ChainConfig{GenesisTime: 100, GenesisValidatorsRoot: common.Hash{8}}).
		AddFork("CAPELLA", 0, []byte{3, 0, 0, 0}).AddFork("DENEB", 1, []byte{4, 0, 0, 0}).
		AddFork("ELECTRA", 256, []byte{5, 0, 0, 0}).AddFork("FULU", 512, []byte{6, 0, 0, 0})
}
func finalityCommittee(t *testing.T, id byte) (*bt.SerializedSyncCommittee, *bls.SecretKey) {
	t.Helper()
	var seed [32]byte
	seed[31] = id
	sk := new(bls.SecretKey)
	if err := sk.Deserialize(&seed); err != nil {
		t.Fatal(err)
	}
	pk, err := bls.SkToPk(sk)
	if err != nil {
		t.Fatal(err)
	}
	encoded := pk.Serialize()
	committee := new(bt.SerializedSyncCommittee)
	keys := make([]*bls.Pubkey, 512)
	for i := range keys {
		keys[i] = pk
		copy(committee[i*48:], encoded[:])
	}
	aggregate, err := bls.AggregatePubkeys(keys)
	if err != nil {
		t.Fatal(err)
	}
	encoded = aggregate.Serialize()
	copy(committee[512*48:], encoded[:])
	return committee, sk
}
func finalityTree(leaves map[uint64]common.Hash) (common.Hash, func(uint64) []common.Hash) {
	maxDepth := uint64(0)
	for index := range leaves {
		for d := uint64(0); index>>d > 0; d++ {
			maxDepth = max(maxDepth, d)
		}
	}
	var node func(uint64) common.Hash
	node = func(index uint64) common.Hash {
		if leaf, ok := leaves[index]; ok {
			return leaf
		}
		if index >= 1<<maxDepth {
			return common.Hash{}
		}
		a, b := node(index*2), node(index*2+1)
		return sha256.Sum256(append(a[:], b[:]...))
	}
	branch := func(index uint64) []common.Hash {
		var out []common.Hash
		for index > 1 {
			out = append(out, node(index^1))
			index /= 2
		}
		return out
	}
	return node(1), branch
}
func finalityTestHeader(t *testing.T, slot uint64, hash common.Hash, net *bp.ChainConfig) FinalityHeader {
	t.Helper()
	var object any
	if net.ForkNameAtSlot(slot) == "capella" {
		object = &capella.ExecutionPayloadHeader{BlockHash: [32]byte(hash), ExtraData: []byte{}}
	} else {
		object = &deneb.ExecutionPayloadHeader{BlockHash: [32]byte(hash), BaseFeePerGas: new(uint256.Int), ExtraData: []byte{}}
	}
	data, err := json.Marshal(object)
	if err != nil {
		t.Fatal(err)
	}
	payload, err := bt.ExecutionHeaderFromJSON(net.ForkNameAtSlot(slot), data)
	if err != nil {
		t.Fatal(err)
	}
	root, branch := finalityTree(map[uint64]common.Hash{25: common.Hash(payload.PayloadRoot())})
	return FinalityHeader{Beacon: bt.Header{Slot: slot, BodyRoot: root}, Execution: data, ExecutionBranch: branch(25)}
}
func signFinalityUpdate(t *testing.T, u *FinalityUpdate, key *bls.SecretKey, count int, net *bp.ChainConfig) {
	t.Helper()
	root, err := net.Forks.SigningRoot((uint64(u.SignatureSlot)-1)/32, u.Attested.Beacon.Hash())
	if err != nil {
		t.Fatal(err)
	}
	signature := bls.Sign(key, root[:])
	signatures := make([]*bls.Signature, count)
	u.Aggregate = bt.SyncAggregate{}
	for i := range count {
		u.Aggregate.Signers[i/8] |= 1 << uint(i%8)
		signatures[i] = signature
	}
	aggregate, err := bls.Aggregate(signatures)
	if err != nil {
		t.Fatal(err)
	}
	u.Aggregate.Signature = aggregate.Serialize()
}
func finalityTestUpdate(t *testing.T, att, fin uint64, next *bt.SerializedSyncCommittee, key *bls.SecretKey, net *bp.ChainConfig) FinalityUpdate {
	t.Helper()
	h := finalityTestHeader(t, fin, common.Hash{byte(fin), 9}, net)
	u := FinalityUpdate{Attested: finalityTestHeader(t, att, common.Hash{7}, net), Finalized: &h, Next: next, SignatureSlot: common.Decimal(att + 1)}
	fork := net.ForkNameAtSlot(att)
	leaves := map[uint64]common.Hash{bp.StateIndexFinalBlock(fork): h.Beacon.Hash()}
	if next != nil {
		leaves[bp.StateIndexNextSyncCommittee(fork)] = next.Root()
	}
	root, branch := finalityTree(leaves)
	u.Attested.Beacon.StateRoot = root
	u.FinalityBranch = branch(bp.StateIndexFinalBlock(fork))
	if next != nil {
		u.NextBranch = branch(bp.StateIndexNextSyncCommittee(fork))
	}
	signFinalityUpdate(t, &u, key, 342, net)
	return u
}
func TestFinalityRealBLSThresholdAndTampering(t *testing.T) {
	net := testFinalityNetwork()
	committee, key := finalityCommittee(t, 1)
	next, _ := finalityCommittee(t, 2)
	initial := finalityState{Finalized: finalityTestHeader(t, 100, common.Hash{1}, net), Current: committee}
	tests := []struct {
		name   string
		mutate func(*FinalityUpdate)
		want   error
	}{
		{"342", func(*FinalityUpdate) {}, nil},
		{"341", func(u *FinalityUpdate) { signFinalityUpdate(t, u, key, 341, net) }, ErrFinalityParticipation},
		{"signature", func(u *FinalityUpdate) { u.Aggregate.Signature[0] ^= 1 }, errors.New("invalid")},
		{"branch", func(u *FinalityUpdate) { u.FinalityBranch[0][0] ^= 1 }, errors.New("invalid")},
		{"next committee", func(u *FinalityUpdate) { replacement := *u.Next; replacement[0] ^= 1; u.Next = &replacement }, errors.New("invalid")},
		{"future", func(u *FinalityUpdate) { u.SignatureSlot = 10001 }, errors.New("invalid")},
		{"skip", func(u *FinalityUpdate) { u.SignatureSlot = 16385 }, errors.New("invalid")},
		{"wrong domain", func(u *FinalityUpdate) {
			other := testFinalityNetwork()
			other.GenesisValidatorsRoot[0] ^= 1
			other.AddFork("DENEB", 1, []byte{9, 0, 0, 0})
			signFinalityUpdate(t, u, key, 342, other)
		}, errors.New("invalid")},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			s := initial
			u := finalityTestUpdate(t, 160, 128, next, key, net)
			tc.mutate(&u)
			before, _ := json.Marshal(s)
			now := uint64(200)
			if tc.name == "skip" {
				now = 20000 // Exercise the unknown-committee check, not the future-slot check.
			}
			changed, err := s.processUpdate(u, now, net)
			if tc.want == nil {
				if err != nil || !changed || s.Finalized.Beacon.Slot != 128 || s.Next == nil {
					t.Fatalf("changed=%v err=%v", changed, err)
				}
			} else {
				if err == nil {
					t.Fatal("accepted invalid update")
				}
				if errors.Is(tc.want, ErrFinalityParticipation) && !errors.Is(err, tc.want) {
					t.Fatal(err)
				}
				after, _ := json.Marshal(s)
				if string(before) != string(after) {
					t.Fatal("invalid update changed state")
				}
			}
		})
	}
}
func TestFinalityCommitteeRotationAndForkBoundary(t *testing.T) {
	net := testFinalityNetwork()
	first, k1 := finalityCommittee(t, 1)
	second, k2 := finalityCommittee(t, 2)
	third, k3 := finalityCommittee(t, 3)
	s := finalityState{Finalized: finalityTestHeader(t, 8000, common.Hash{1}, net), Current: first}
	u := finalityTestUpdate(t, 8100, 8064, second, k1, net)
	if _, err := s.processUpdate(u, 8200, net); err != nil {
		t.Fatal(err)
	}
	// The first slot of Electra signs the final Deneb slot using the previous
	// fork's domain, but the next period's committee.
	boundary := finalityTestUpdate(t, 8191, 8160, second, k2, net)
	if _, err := s.processUpdate(boundary, 8200, net); err != nil {
		t.Fatal(err)
	}
	u = finalityTestUpdate(t, 8250, 8224, third, k2, net)
	if _, err := s.processUpdate(u, 8300, net); err != nil {
		t.Fatal(err)
	}
	if *s.Current != *second || *s.Next != *third {
		t.Fatal("committee rotation failed")
	}
	if changed, err := s.processUpdate(u, 8300, net); err != nil || changed {
		t.Fatalf("duplicate update: %v %v", changed, err)
	}
	// Repeat the period/domain transition at Fulu, whose execution layout stays Deneb.
	boundary = finalityTestUpdate(t, 16383, 16352, third, k3, net)
	if _, err := s.processUpdate(boundary, 16400, net); err != nil {
		t.Fatal(err)
	}
	fourth, _ := finalityCommittee(t, 4)
	u = finalityTestUpdate(t, 16450, 16416, fourth, k3, net)
	if _, err := s.processUpdate(u, 16500, net); err != nil {
		t.Fatal(err)
	}
	if *s.Current != *third || *s.Next != *fourth {
		t.Fatal("Fulu committee rotation failed")
	}
}
func TestFinalityBootstrapRootAndCommittee(t *testing.T) {
	net := testFinalityNetwork()
	committee, _ := finalityCommittee(t, 1)
	b := FinalityBootstrap{Header: finalityTestHeader(t, 100, common.Hash{1}, net), Committee: committee}
	root, branch := finalityTree(map[uint64]common.Hash{54: committee.Root()})
	b.Header.Beacon.StateRoot = root
	b.Branch = branch(54)
	trusted := b.Header.Beacon.Hash()
	if _, err := bootstrapFinality(b, trusted, 200, net); err != nil {
		t.Fatal(err)
	}
	if _, err := bootstrapFinality(b, common.Hash{3}, 200, net); err == nil {
		t.Fatal("accepted wrong root")
	}
	b.Branch[0][0] ^= 1
	if _, err := bootstrapFinality(b, trusted, 200, net); err == nil {
		t.Fatal("accepted wrong committee branch")
	}
}
func TestFinalityHeliosHistoricalResponses(t *testing.T) {
	var b finalityEnvelope[FinalityBootstrap]
	readFinalityFixture(t, "helios/bootstrap.json", &b)
	if b.Data == nil {
		t.Fatal("missing bootstrap")
	}
	root := b.Data.Header.Beacon.Hash()
	const now = 7500000
	state, err := bootstrapFinality(*b.Data, root, now, bp.MainnetLightConfig)
	if err != nil {
		t.Fatal(err)
	}
	var updates []finalityEnvelope[FinalityUpdate]
	readFinalityFixture(t, "helios/updates.json", &updates)
	for i, u := range updates {
		if _, err = state.processUpdate(*u.Data, now, bp.MainnetLightConfig); err != nil {
			t.Fatalf("update %d: %v", i, err)
		}
	}
	var f finalityEnvelope[FinalityUpdate]
	readFinalityFixture(t, "helios/finality.json", &f)
	if _, err = state.processUpdate(*f.Data, now, bp.MainnetLightConfig); err != nil {
		t.Fatal(err)
	}
	if state.Finalized.Beacon.Slot <= b.Data.Header.Beacon.Slot {
		t.Fatal("no finalized progress")
	}
}
func readFinalityFixture(t *testing.T, name string, out any) {
	t.Helper()
	b, err := os.ReadFile("testdata/finality/" + name)
	if err != nil {
		t.Fatal(err)
	}
	if err = json.Unmarshal(b, out); err != nil {
		t.Fatal(err)
	}
}

func TestFinalityDenebTransitionAndUnknownFork(t *testing.T) {
	net := testFinalityNetwork()
	committee, key := finalityCommittee(t, 1)
	next, _ := finalityCommittee(t, 2)
	s := finalityState{Finalized: finalityTestHeader(t, 10, common.Hash{1}, net), Current: committee}
	for _, slots := range [][2]uint64{{31, 16}, {70, 64}} {
		u := finalityTestUpdate(t, slots[0], slots[1], next, key, net)
		// Electra-normalized zero prefixes must also work for an earlier fork.
		u.NextBranch = append([]common.Hash{{}}, u.NextBranch...)
		u.FinalityBranch = append([]common.Hash{{}}, u.FinalityBranch...)
		if _, err := s.processUpdate(u, 100, net); err != nil {
			t.Fatal(err)
		}
	}
	net.AddFork("GLOAS", 768, []byte{7, 0, 0, 0})
	header := s.Finalized
	header.Beacon.Slot = 768 * 32
	if _, err := verifyFinalityHeader(header, net); !errors.Is(err, ErrFinalityUnsupported) {
		t.Fatal(err)
	}
}

func TestFinalityGenesisCheckpointOnlyLearnsCommittee(t *testing.T) {
	for _, fork := range []string{"CAPELLA", "DENEB", "ELECTRA", "FULU"} {
		t.Run(fork, func(t *testing.T) {
			net := (&bp.ChainConfig{GenesisTime: 100, GenesisValidatorsRoot: common.Hash{8}}).AddFork(fork, 0, []byte{3, 0, 0, 0})
			committee, key := finalityCommittee(t, 1)
			next, _ := finalityCommittee(t, 2)
			var payload any = &deneb.ExecutionPayloadHeader{BaseFeePerGas: new(uint256.Int), ExtraData: []byte{}}
			if fork == "CAPELLA" {
				payload = &capella.ExecutionPayloadHeader{ExtraData: []byte{}}
			}
			execution, err := json.Marshal(payload)
			if err != nil {
				t.Fatal(err)
			}
			empty := FinalityHeader{Execution: execution, ExecutionBranch: make([]common.Hash, 4)}
			u := FinalityUpdate{Attested: finalityTestHeader(t, 40, common.Hash{1}, net), Finalized: &empty, Next: next, SignatureSlot: 41}
			finalIndex := bp.StateIndexFinalBlock(net.ForkNameAtSlot(40))
			nextIndex := bp.StateIndexNextSyncCommittee(net.ForkNameAtSlot(40))
			root, branch := finalityTree(map[uint64]common.Hash{finalIndex: {}, nextIndex: next.Root()})
			u.Attested.Beacon.StateRoot = root
			u.FinalityBranch = branch(finalIndex)
			u.NextBranch = branch(nextIndex)
			signFinalityUpdate(t, &u, key, 342, net)
			for _, bad := range []string{"", "beacon", "execution", "branch", "signature", "finality-proof"} {
				t.Run(bad, func(t *testing.T) {
					s := finalityState{Finalized: finalityTestHeader(t, 64, common.Hash{2}, net), Current: committee}
					original := s.Finalized.Beacon.Hash()
					// Deep copy to ensure mutations are isolated from subsequent cases.
					encoded, err := json.Marshal(u)
					if err != nil {
						t.Fatal(err)
					}
					var update FinalityUpdate
					if err = json.Unmarshal(encoded, &update); err != nil {
						t.Fatal(err)
					}
					switch bad {
					case "beacon":
						update.Finalized.Beacon.ParentRoot[0] = 1
					case "execution":
						update.Finalized.Execution = finalityTestHeader(t, 40, common.Hash{1}, net).Execution
					case "branch":
						update.Finalized.ExecutionBranch[0][0] = 1
					case "signature":
						update.Aggregate.Signature[0] ^= 1
					case "finality-proof":
						update.FinalityBranch[0][0] ^= 1
					}
					changed, err := s.processUpdate(update, 100, net)
					if bad == "" {
						if err != nil || !changed || s.Next == nil {
							t.Fatalf("genesis committee update failed: %v", err)
						}
					} else if err == nil || changed || s.Next != nil {
						t.Fatal("accepted tampered genesis update")
					}
					if s.Finalized.Beacon.Hash() != original {
						t.Fatal("genesis sentinel changed finalized anchor")
					}
				})
			}
			if _, err = verifyFinalityHeader(empty, net); err == nil {
				t.Fatal("accepted empty header as execution anchor")
			}
		})
	}
}
