package proof

import (
	"context"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient/gethclient"
)

func TestGenerateStateProofResponse(t *testing.T) {
	req, source := testStateRequest(t)

	pkg, err := GenerateStateProof(context.Background(), source, req)
	if err != nil {
		t.Fatalf("GenerateStateProof: %v", err)
	}
	if pkg.Account != req.Account {
		t.Fatalf("unexpected account: got %s want %s", pkg.Account, req.Account)
	}
	if got, want := len(pkg.StorageProofs), len(req.Slots); got != want {
		t.Fatalf("unexpected storage proof count: got %d want %d", got, want)
	}
	for i, slot := range req.Slots {
		if pkg.StorageProofs[i].Slot != slot {
			t.Fatalf("unexpected slot[%d]: got %s want %s", i, pkg.StorageProofs[i].Slot, slot)
		}
	}
}

func TestGenerateStateProofResponseRejectsNilProof(t *testing.T) {
	req, source := testStateRequest(t)
	source.proof = nil
	_, err := GenerateStateProof(t.Context(), source, req)
	if err == nil || !strings.Contains(err.Error(), "nil account proof") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestNormalizeStorageProofResults(t *testing.T) {
	fixture := mustLoadStateFixture(t)
	expectedSlots := storageProofSlotsFromFixture(fixture)
	results := storageResultsFromFixture(fixture)

	t.Run("orders results by requested slot order", func(t *testing.T) {
		reordered := append([]common.Hash(nil), expectedSlots...)
		reordered[0], reordered[1] = reordered[1], reordered[0]

		got, err := normalizeStorageProofResults(reordered, fixture.AccountClaim.StorageRoot, results)
		if err != nil {
			t.Fatalf("normalizeStorageProofResults: %v", err)
		}
		if got[0].Slot != reordered[0] || got[1].Slot != reordered[1] {
			t.Fatalf("unexpected slot order: %+v", got)
		}
	})

	tests := []struct {
		name string
		edit func([]common.Hash, []gethclient.StorageResult) ([]common.Hash, []gethclient.StorageResult)
		want string
	}{
		{
			name: "count mismatch",
			edit: func(slots []common.Hash, in []gethclient.StorageResult) ([]common.Hash, []gethclient.StorageResult) {
				return slots, in[:1]
			},
			want: "expected 2 storage proofs, got 1",
		},
		{
			name: "unexpected key",
			edit: func(slots []common.Hash, in []gethclient.StorageResult) ([]common.Hash, []gethclient.StorageResult) {
				out := append([]gethclient.StorageResult(nil), in...)
				out[1].Key = common.HexToHash("0x03").Hex()
				return slots, out
			},
			want: "unexpected storage proof key",
		},
		{
			name: "duplicate key",
			edit: func(slots []common.Hash, in []gethclient.StorageResult) ([]common.Hash, []gethclient.StorageResult) {
				out := append([]gethclient.StorageResult(nil), in...)
				out[1].Key = out[0].Key
				return slots, out
			},
			want: "duplicate storage proof key",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			slots, edited := tt.edit(expectedSlots, results)
			_, err := normalizeStorageProofResults(slots, fixture.AccountClaim.StorageRoot, edited)
			if err == nil {
				t.Fatal("expected normalizeStorageProofResults to fail")
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestGenerateStateProof(t *testing.T) {
	req, source := testStateRequest(t)

	pkg, err := GenerateStateProof(context.Background(), source, req)
	if err != nil {
		t.Fatalf("GenerateStateProof: %v", err)
	}
	if err := VerifyStateProofPackage(pkg); err != nil {
		t.Fatalf("VerifyStateProofPackage: %v", err)
	}
	if err := VerifyStateProofPackageAgainstSource(context.Background(), pkg, source); err != nil {
		t.Fatalf("VerifyStateProofPackageAgainstSource: %v", err)
	}
}

func TestGenerateStateProofRejectsInvalidSlots(t *testing.T) {
	baseReq, source := testStateRequest(t)
	tests := []struct {
		name string
		edit func(*StateProofRequest)
		want string
	}{
		{
			name: "empty slots",
			edit: func(req *StateProofRequest) {
				req.Slots = nil
			},
			want: "at least one storage slot",
		},
		{
			name: "duplicate slots",
			edit: func(req *StateProofRequest) {
				req.Slots = []common.Hash{req.Slots[0], req.Slots[0]}
			},
			want: "duplicate storage slot",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := baseReq
			tt.edit(&req)
			_, err := GenerateStateProof(context.Background(), source, req)
			if err == nil {
				t.Fatal("expected GenerateStateProof to fail")
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}
