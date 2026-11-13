package proof

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"
)

// TestFinalityLive is opt-in: public provider availability is never a dependency
// of make unit-test. It accepts the CLI config shape, using verify.tx only.
func TestFinalityLive(t *testing.T) {
	path := os.Getenv("ETH_PROOF_FINALITY_LIVE_CONFIG")
	if path == "" {
		t.Skip("set ETH_PROOF_FINALITY_LIVE_CONFIG to a config with finality and verify.tx")
	}
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var cfg struct {
		Finality FinalityConfig `json:"finality"`
		Verify   struct {
			Tx struct {
				RPC   string `json:"rpc"`
				Proof string `json:"proof"`
			} `json:"tx"`
		} `json:"verify"`
	}
	if err = json.Unmarshal(b, &cfg); err != nil {
		t.Fatal(err)
	}
	cfg.Finality.CacheDir = t.TempDir()
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Minute)
	defer cancel()
	var pkg TransactionProofPackage
	if err = LoadJSON(cfg.Verify.Tx.Proof, &pkg); err != nil {
		t.Fatal(err)
	}
	source, err := openRPCSource(ctx, cfg.Verify.Tx.RPC)
	if err != nil {
		t.Fatal(err)
	}
	defer source.Close()
	client, err := NewFinalityClient(cfg.Finality, source)
	if err != nil {
		t.Fatal(err)
	}
	if err = VerifyTransactionProofPackageAgainstFinality(ctx, &pkg, client); err != nil {
		t.Fatal(err)
	}
	// A second call verifies persistence and stable results without bootstrapping again.
	client, err = NewFinalityClient(cfg.Finality, source)
	if err != nil {
		t.Fatal(err)
	}
	if err = VerifyTransactionProofPackageAgainstFinality(ctx, &pkg, client); err != nil {
		t.Fatal(err)
	}
}
