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
	"github.com/ethereum/go-ethereum/common"
	"github.com/golang/snappy"
	"gopkg.in/yaml.v3"
)

func TestFinalityOfficialExecutionProof(t *testing.T) {
	var vector struct {
		Leaf   string   `yaml:"leaf"`
		Index  uint64   `yaml:"leaf_index"`
		Branch []string `yaml:"branch"`
	}
	data, err := os.ReadFile("testdata/finality/spec/execution_proof.yaml")
	if err != nil {
		t.Fatal(err)
	}
	if err = yaml.Unmarshal(data, &vector); err != nil {
		t.Fatal(err)
	}
	data, err = os.ReadFile("testdata/finality/spec/execution_object.ssz_snappy")
	if err != nil {
		t.Fatal(err)
	}
	data, err = snappy.Decode(nil, data)
	if err != nil {
		t.Fatal(err)
	}
	var body capella.BeaconBlockBody
	if err = body.UnmarshalSSZ(data); err != nil {
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
