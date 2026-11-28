package proof

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/attestantio/go-eth2-client/spec/capella"
	"github.com/attestantio/go-eth2-client/spec/gloas"
	bp "github.com/ethereum/go-ethereum/beacon/params"
	bt "github.com/ethereum/go-ethereum/beacon/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/golang/snappy"
	"gopkg.in/yaml.v3"
)

func TestFinalityOfficialExecutionProof(t *testing.T) {
	vector, data := officialProofVector(t, "execution_proof.yaml", "execution_object.ssz_snappy")
	var body capella.BeaconBlockBody
	if err := body.UnmarshalSSZ(data); err != nil {
		t.Fatal(err)
	}
	root, err := body.HashTreeRoot()
	if err != nil {
		t.Fatal(err)
	}
	leaf, err := body.ExecutionPayload.HashTreeRoot()
	if err != nil {
		t.Fatal(err)
	}
	if common.Hash(leaf) != common.HexToHash(vector.Leaf) {
		t.Fatal("official leaf mismatch")
	}
	branch := make([]common.Hash, len(vector.Branch))
	for i, h := range vector.Branch {
		branch[i] = common.HexToHash(h)
	}
	if err = verifyFinalityBranch(root, vector.Index, branch, leaf, false); err != nil {
		t.Fatal(err)
	}
	branch[0][0] ^= 1
	if err = verifyFinalityBranch(root, vector.Index, branch, leaf, false); err == nil {
		t.Fatal("accepted corrupted official proof")
	}
}

type officialMerkleProof struct {
	Leaf   string   `yaml:"leaf"`
	Index  uint64   `yaml:"leaf_index"`
	Branch []string `yaml:"branch"`
}

func officialProofVector(t *testing.T, proofFile, objectFile string) (officialMerkleProof, []byte) {
	t.Helper()
	var proof officialMerkleProof
	data, err := os.ReadFile("testdata/finality/spec/" + proofFile)
	if err != nil {
		t.Fatal(err)
	}
	if err := yaml.Unmarshal(data, &proof); err != nil {
		t.Fatal(err)
	}
	data, err = os.ReadFile("testdata/finality/spec/" + objectFile)
	if err != nil {
		t.Fatal(err)
	}
	data, err = snappy.Decode(nil, data)
	if err != nil {
		t.Fatal(err)
	}
	return proof, data
}

func TestGloasOfficialExecutionProof(t *testing.T) {
	vector, data := officialProofVector(t, "gloas/proof.yaml", "gloas/object.ssz_snappy")
	var body gloas.BeaconBlockBody
	if err := body.UnmarshalSSZ(data); err != nil {
		t.Fatal(err)
	}
	root, err := body.HashTreeRoot()
	if err != nil {
		t.Fatal(err)
	}
	bid := body.SignedExecutionPayloadBid.Message
	parent, proposed := common.Hash(bid.ParentBlockHash), common.Hash(bid.BlockHash)
	if vector.Index != bp.BodyIndexExecBlockHashGloas || common.HexToHash(vector.Leaf) != parent || parent == proposed {
		t.Fatal("unexpected bid in official Gloas proof")
	}
	h := FinalityHeader{Beacon: bt.Header{Slot: 24600, BodyRoot: root}, ExecutionBlockHash: &parent}
	for _, node := range vector.Branch {
		h.ExecutionBranch = append(h.ExecutionBranch, common.HexToHash(node))
	}
	if got, err := verifyFinalityHeader(h, testFinalityNetwork()); err != nil || got != parent {
		t.Fatalf("parent hash proof: %v", err)
	}
	h.ExecutionBlockHash = &proposed
	if _, err := verifyFinalityHeader(h, testFinalityNetwork()); err == nil {
		t.Fatal("accepted proposed execution hash")
	}
}

func TestFinalityHTTPCheckpointBootstrapFlow(t *testing.T) {
	c, target, beacon, sources := finalityClientFixture(t)
	root := beacon.b.Header.Beacon.Hash()
	headerBytes, err := json.Marshal(beacon.b.Header)
	if err != nil {
		t.Fatal(err)
	}
	var fields map[string]json.RawMessage
	if err = json.Unmarshal(headerBytes, &fields); err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		var value any
		switch r.URL.Path {
		case "/eth/v1/beacon/genesis":
			value = map[string]any{"data": map[string]any{"genesis_time": fmt.Sprint(c.network.GenesisTime), "genesis_validators_root": c.network.GenesisValidatorsRoot}}
		case "/eth/v1/beacon/states/head/finality_checkpoints":
			value = map[string]any{"execution_optimistic": false, "data": map[string]any{"finalized": sources[0].cp}}
		case "/eth/v1/beacon/headers/" + root.Hex():
			value = map[string]any{"execution_optimistic": false, "finalized": true, "data": map[string]any{"root": root, "canonical": true, "header": map[string]any{"message": fields["beacon"]}}}
		case "/eth/v1/beacon/light_client/bootstrap/" + root.Hex():
			value = map[string]any{"version": "deneb", "data": beacon.b}
		default:
			http.NotFound(w, r)
			return
		}
		if err := json.NewEncoder(w).Encode(value); err != nil {
			t.Error(err)
		}
	}))
	defer server.Close()
	c.beacon, err = NewBeaconFinalitySource(server.URL, nil)
	if err != nil {
		t.Fatal(err)
	}
	for i := range c.checkpoints {
		c.checkpoints[i], err = NewCheckpointSource(fmt.Sprintf("op-%d", i), server.URL, nil)
		if err != nil {
			t.Fatal(err)
		}
	}
	if _, err = c.FinalizedAnchor(t.Context(), target); err != nil {
		t.Fatal(err)
	}
	if _, err = c.FinalizedAnchor(t.Context(), target); err != nil {
		t.Fatal(err)
	}
}

func TestFinalityHTTPRejectsBadResponses(t *testing.T) {
	for _, test := range []struct {
		name, body string
		status     int
	}{
		{"unknown fork", `{"version":"future","data":{}}`, 200},
		{"missing data", `{"version":"deneb"}`, 200},
		{"malformed", `secret-token`, 200},
		{"http error", `secret-token`, 500},
		{"oversized", strings.Repeat(" ", (16<<20)+1), 200},
	} {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(test.status)
				_, _ = w.Write([]byte(test.body))
			}))
			defer server.Close()
			source, err := NewBeaconFinalitySource(server.URL+"/secret-token?api_key=secret-token", nil)
			if err != nil {
				t.Fatal(err)
			}
			if _, err = source.Bootstrap(t.Context(), common.Hash{}); err == nil || strings.Contains(err.Error(), "secret-token") {
				t.Fatalf("error: %v", err)
			}
		})
	}
}
func TestFinalityHTTPCancellationAndRedirect(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "http://invalid.example/secret-token", http.StatusFound)
	}))
	defer server.Close()
	source, err := NewBeaconFinalitySource(server.URL, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = source.Bootstrap(t.Context(), common.Hash{}); err == nil || strings.Contains(err.Error(), "secret-token") {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if _, err = source.Bootstrap(ctx, common.Hash{}); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
}

func TestFinalityHTTPRedactsDeferredExecutionErrors(t *testing.T) {
	for _, test := range []struct {
		name, field string
		slot        uint64
	}{
		{"capella payload", "block_number", 10},
		{"capella blob used", "blob_gas_used", 10},
		{"capella blob excess", "excess_blob_gas", 10},
		{"deneb payload", "block_number", 128},
		{"electra payload", "block_number", 8192},
		{"fulu payload", "block_number", 16384},
	} {
		t.Run(test.name, func(t *testing.T) {
			c, _, beacon, _ := finalityClientFixture(t)
			b := beacon.b
			b.Header = finalityTestHeader(t, test.slot, common.Hash{1}, c.network)
			b.Branch = make([]common.Hash, branchDepth(bp.StateIndexSyncCommittee(c.network.ForkNameAtSlot(test.slot))))
			var fields map[string]json.RawMessage
			if err := json.Unmarshal(b.Header.Execution, &fields); err != nil {
				t.Fatal(err)
			}
			fields[test.field] = json.RawMessage(`"secret-token"`)
			var err error
			b.Header.Execution, err = json.Marshal(fields)
			if err != nil {
				t.Fatal(err)
			}
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				if err := json.NewEncoder(w).Encode(map[string]any{"version": c.network.ForkNameAtSlot(test.slot), "data": b}); err != nil {
					t.Error(err)
				}
			}))
			defer server.Close()
			source, err := NewBeaconFinalitySource(server.URL, nil)
			if err != nil {
				t.Fatal(err)
			}
			decoded, err := source.Bootstrap(t.Context(), b.Header.Beacon.Hash())
			if err == nil {
				_, err = bootstrapFinality(decoded, b.Header.Beacon.Hash(), test.slot+1, c.network)
			}
			if err == nil {
				t.Fatal("accepted malformed execution payload")
			}
			for cause := err; cause != nil; cause = errors.Unwrap(cause) {
				if strings.Contains(fmt.Sprintf("%+v", cause), "secret-token") {
					t.Fatal("deferred execution decoder exposed response data")
				}
			}
		})
	}
}
func TestFinalityHTTPRequiresExplicitProviderFlags(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"data":{"root":"0x0000000000000000000000000000000000000000000000000000000000000000"}}`))
	}))
	defer server.Close()
	s, err := NewCheckpointSource("operator", server.URL, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.CheckpointHeader(t.Context(), common.Hash{}); err == nil {
		t.Fatal("accepted missing trust flags")
	}
}

func TestGloasBootstrapAtFuluSlot(t *testing.T) {
	net := testFinalityNetwork()
	const slot = 16400
	committee, _ := finalityCommittee(t, 1)
	index := bp.StateIndexSyncCommittee(net.ForkNameAtSlot(slot))
	root, branch := finalityTree(map[uint64]common.Hash{index: committee.Root()})
	b := FinalityBootstrap{Header: gloasTestHeader(slot, common.Hash{1}, net), Committee: committee}
	b.Header.Beacon.StateRoot = root
	b.Branch = append(make([]common.Hash, branchDepth(bp.StateIndexSyncCommitteeGloas)-branchDepth(index)), branch(index)...)
	env := finalityEnvelope[FinalityBootstrap]{Version: "gloas", Data: &b}
	data, err := json.Marshal(env)
	if err != nil {
		t.Fatal(err)
	}
	var decoded finalityEnvelope[FinalityBootstrap]
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatal(err)
	}
	if err := decoded.check(); err != nil {
		t.Fatal(err)
	}
	if _, err := bootstrapFinality(*decoded.Data, b.Header.Beacon.Hash(), slot+1, net); err != nil {
		t.Fatal(err)
	}
	decoded.Version = "fulu"
	if err := decoded.check(); err == nil {
		t.Fatal("accepted Gloas data labeled Fulu")
	}
	decoded.Version = "gloas"
	decoded.Data.Branch[0][0] ^= 1
	if _, err := bootstrapFinality(*decoded.Data, b.Header.Beacon.Hash(), slot+1, net); err == nil {
		t.Fatal("accepted nonzero committee padding")
	}
	decoded.Data.Branch = decoded.Data.Branch[1:]
	if err := decoded.check(); err == nil {
		t.Fatal("accepted truncated committee branch")
	}
}

func TestGloasUpdateRequiredFields(t *testing.T) {
	net := testFinalityNetwork()
	header := gloasTestHeader(24600, common.Hash{1}, net)
	committee, _ := finalityCommittee(t, 1)
	finality := FinalityUpdate{
		Attested: header, Finalized: &header, FinalityBranch: make([]common.Hash, branchDepth(bp.StateIndexFinalBlockGloas)),
	}
	full := finality
	full.Next = committee
	full.NextBranch = make([]common.Hash, branchDepth(bp.StateIndexNextSyncCommitteeGloas))
	for _, tc := range []struct {
		name   string
		full   bool
		update FinalityUpdate
		mutate func(*FinalityUpdate)
	}{
		{"full", true, full, nil},
		{"finality only", false, finality, nil},
		{"extra next committee", false, finality, func(u *FinalityUpdate) { u.Next = committee }},
		{"extra next branch", false, finality, func(u *FinalityUpdate) { u.NextBranch = full.NextBranch }},
		{"missing finalized header", false, finality, func(u *FinalityUpdate) { u.Finalized = nil }},
		{"missing next committee", true, full, func(u *FinalityUpdate) { u.Next = nil }},
		{"missing next branch", true, full, func(u *FinalityUpdate) { u.NextBranch = nil }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if tc.mutate != nil {
				tc.mutate(&tc.update)
			}
			envelope := finalityEnvelope[FinalityUpdate]{Version: "gloas", Data: &tc.update}
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				var value any = envelope
				if tc.full {
					value = []any{envelope}
				}
				if err := json.NewEncoder(w).Encode(value); err != nil {
					t.Error(err)
				}
			}))
			defer server.Close()
			source, err := NewBeaconFinalitySource(server.URL, nil)
			if err != nil {
				t.Fatal(err)
			}
			if tc.full {
				_, err = source.Updates(t.Context(), 3, 1)
			} else {
				_, err = source.FinalityUpdate(t.Context())
			}
			if (err != nil) != (tc.mutate != nil) {
				t.Fatalf("decode: %v", err)
			}
		})
	}
}

func TestLegacyExecutionWireFork(t *testing.T) {
	net := testFinalityNetwork()
	capella := finalityTestHeader(t, 16, common.Hash{1}, net)
	deneb := finalityTestHeader(t, 128, common.Hash{2}, net)
	if capella.matchesWireFork("deneb") || deneb.matchesWireFork("capella") {
		t.Fatal("accepted payload fields from the wrong wire fork")
	}
	if !capella.matchesWireFork("capella") || !deneb.matchesWireFork("deneb") {
		t.Fatal("rejected valid execution fields")
	}
}
