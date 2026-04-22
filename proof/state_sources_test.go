package proof

import (
	"context"
	"slices"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient/gethclient"
)

func TestFetchStateSnapshot(t *testing.T) {
	req, _, _ := testStateProofSourcesRequest(t)
	source := req.Sources[0]

	snapshot, err := fetchStateSnapshot(context.Background(), source, req.BlockNumber, req.Account, req.Slots)
	if err != nil {
		t.Fatalf("fetchStateSnapshot: %v", err)
	}
	if snapshot.Account != req.Account {
		t.Fatalf("unexpected account: got %s want %s", snapshot.Account, req.Account)
	}
	if got, want := len(snapshot.StorageProofs), len(req.Slots); got != want {
		t.Fatalf("unexpected storage proof count: got %d want %d", got, want)
	}
	for i, slot := range req.Slots {
		if snapshot.StorageProofs[i].Slot != slot {
			t.Fatalf("unexpected slot[%d]: got %s want %s", i, snapshot.StorageProofs[i].Slot, slot)
		}
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

func TestGenerateStateProofFromSources(t *testing.T) {
	req, verifyReq, wantNames := testStateProofSourcesRequest(t)

	pkg, err := GenerateStateProofFromSources(context.Background(), req)
	if err != nil {
		t.Fatalf("GenerateStateProofFromSources: %v", err)
	}
	if got := pkg.Block.SourceConsensus.RPCs; !slices.Equal(got, wantNames) {
		t.Fatalf("unexpected source names: got %v want %v", got, wantNames)
	}
	if err := VerifyStateProofPackage(pkg); err != nil {
		t.Fatalf("VerifyStateProofPackage: %v", err)
	}
	if err := VerifyStateProofPackageAgainstSources(context.Background(), pkg, verifyReq); err != nil {
		t.Fatalf("VerifyStateProofPackageAgainstSources: %v", err)
	}
}

func TestGenerateStateProofFromSourcesRejectsInvalidSlots(t *testing.T) {
	baseReq, _, _ := testStateProofSourcesRequest(t)
	tests := []struct {
		name string
		edit func(*StateProofSourcesRequest)
		want string
	}{
		{
			name: "empty slots",
			edit: func(req *StateProofSourcesRequest) {
				req.Slots = nil
			},
			want: "at least one storage slot",
		},
		{
			name: "duplicate slots",
			edit: func(req *StateProofSourcesRequest) {
				req.Slots = []common.Hash{req.Slots[0], req.Slots[0]}
			},
			want: "duplicate storage slot",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := baseReq
			tt.edit(&req)
			_, err := GenerateStateProofFromSources(context.Background(), req)
			if err == nil {
				t.Fatal("expected GenerateStateProofFromSources to fail")
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}
