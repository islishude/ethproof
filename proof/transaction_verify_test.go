package proof

import (
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

func TestVerifyTransactionProofPackage(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		pkg := mustLoadTransactionFixture(t)
		if err := VerifyTransactionProofPackage(&pkg); err != nil {
			t.Fatalf("VerifyTransactionProofPackage: %v", err)
		}
	})

	t.Run("tampered roots and nodes fail", func(t *testing.T) {
		fixture := mustLoadTransactionFixture(t)
		tests := []struct {
			name string
			edit func(*TransactionProofPackage)
		}{
			{
				name: "transactions root",
				edit: func(pkg *TransactionProofPackage) {
					pkg.Block.TransactionsRoot = common.HexToHash("0x1234")
				},
			},
			{
				name: "proof node",
				edit: func(pkg *TransactionProofPackage) {
					pkg.ProofNodes[0] = mutateHexNode(t, pkg.ProofNodes[0])
				},
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				pkg := cloneTransactionPackage(fixture)
				tt.edit(&pkg)
				if err := VerifyTransactionProofPackage(&pkg); err == nil {
					t.Fatal("expected tampered package to fail verification")
				}
			})
		}
	})

	t.Run("index and hash mismatches fail", func(t *testing.T) {
		fixture := mustLoadTransactionFixture(t)
		tests := []struct {
			name string
			edit func(*TransactionProofPackage)
		}{
			{
				name: "tx index",
				edit: func(pkg *TransactionProofPackage) {
					pkg.TxIndex++
				},
			},
			{
				name: "tx hash",
				edit: func(pkg *TransactionProofPackage) {
					pkg.TxHash = common.HexToHash("0xbeef")
				},
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				pkg := cloneTransactionPackage(fixture)
				tt.edit(&pkg)
				if err := VerifyTransactionProofPackage(&pkg); err == nil {
					t.Fatal("expected mismatch to fail verification")
				}
			})
		}
	})

	t.Run("decode failure surfaces cleanly", func(t *testing.T) {
		pkg := cloneTransactionPackage(mustLoadTransactionFixture(t))
		pkg.TransactionRLP = nil
		err := VerifyTransactionProofPackage(&pkg)
		if err == nil {
			t.Fatal("expected transaction decode failure")
		}
		if !strings.Contains(err.Error(), "decode claimed transaction") {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}
