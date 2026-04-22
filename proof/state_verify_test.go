package proof

import (
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

func TestVerifyStateProofPackage(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		pkg := mustLoadStateFixture(t)
		if err := VerifyStateProofPackage(&pkg); err != nil {
			t.Fatalf("VerifyStateProofPackage: %v", err)
		}
	})

	t.Run("tampered roots and nodes fail", func(t *testing.T) {
		fixture := mustLoadStateFixture(t)
		tests := []struct {
			name string
			edit func(*StateProofPackage)
		}{
			{
				name: "state root",
				edit: func(pkg *StateProofPackage) {
					pkg.Block.StateRoot = common.HexToHash("0x1234")
				},
			},
			{
				name: "account proof node",
				edit: func(pkg *StateProofPackage) {
					pkg.AccountProofNodes[0] = mutateHexNode(t, pkg.AccountProofNodes[0])
				},
			},
			{
				name: "storage proof node",
				edit: func(pkg *StateProofPackage) {
					pkg.StorageProofs[0].ProofNodes[0] = mutateHexNode(t, pkg.StorageProofs[0].ProofNodes[0])
				},
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				pkg := cloneStatePackage(fixture)
				tt.edit(&pkg)
				if err := VerifyStateProofPackage(&pkg); err == nil {
					t.Fatal("expected tampered package to fail verification")
				}
			})
		}
	})

	t.Run("tampered account claim fails", func(t *testing.T) {
		fixture := mustLoadStateFixture(t)
		tests := []struct {
			name string
			edit func(*StateProofPackage)
		}{
			{
				name: "nonce",
				edit: func(pkg *StateProofPackage) {
					pkg.AccountClaim.Nonce++
				},
			},
			{
				name: "balance",
				edit: func(pkg *StateProofPackage) {
					pkg.AccountClaim.Balance = "0x1"
				},
			},
			{
				name: "storage root",
				edit: func(pkg *StateProofPackage) {
					pkg.AccountClaim.StorageRoot = common.HexToHash("0x8888")
				},
			},
			{
				name: "code hash",
				edit: func(pkg *StateProofPackage) {
					pkg.AccountClaim.CodeHash = common.HexToHash("0x9999")
				},
			},
			{
				name: "storage value",
				edit: func(pkg *StateProofPackage) {
					pkg.StorageProofs[0].Value = common.HexToHash("0x7777")
				},
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				pkg := cloneStatePackage(fixture)
				tt.edit(&pkg)
				if err := VerifyStateProofPackage(&pkg); err == nil {
					t.Fatal("expected tampered account claim to fail verification")
				}
			})
		}
	})

	t.Run("invalid storage proof list fails", func(t *testing.T) {
		fixture := mustLoadStateFixture(t)
		tests := []struct {
			name string
			edit func(*StateProofPackage)
			want string
		}{
			{
				name: "empty",
				edit: func(pkg *StateProofPackage) {
					pkg.StorageProofs = nil
				},
				want: "at least one storage proof",
			},
			{
				name: "duplicate slot",
				edit: func(pkg *StateProofPackage) {
					pkg.StorageProofs = append(pkg.StorageProofs, cloneStateStorageProofs(pkg.StorageProofs[:1])...)
				},
				want: "duplicate storage slot",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				pkg := cloneStatePackage(fixture)
				tt.edit(&pkg)
				err := VerifyStateProofPackage(&pkg)
				if err == nil {
					t.Fatal("expected invalid storage proofs to fail")
				}
				if !strings.Contains(err.Error(), tt.want) {
					t.Fatalf("unexpected error: %v", err)
				}
			})
		}
	})
}
