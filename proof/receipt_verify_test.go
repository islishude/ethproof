package proof

import (
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/islishude/ethproof/internal/proofutil"
)

func TestVerifyReceiptProofPackage(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		pkg := mustLoadReceiptFixture(t)
		expect := &ReceiptExpectations{
			Emitter: &pkg.Event.Address,
			Topics:  append([]common.Hash(nil), pkg.Event.Topics...),
			Data:    append([]byte(nil), pkg.Event.Data...),
		}
		if err := VerifyReceiptProofPackageWithExpectations(&pkg, expect); err != nil {
			t.Fatalf("VerifyReceiptProofPackageWithExpectations: %v", err)
		}
	})

	t.Run("tampered roots and nodes fail", func(t *testing.T) {
		fixture := mustLoadReceiptFixture(t)
		tests := []struct {
			name string
			edit func(*ReceiptProofPackage)
		}{
			{
				name: "receipts root",
				edit: func(pkg *ReceiptProofPackage) {
					pkg.Block.ReceiptsRoot = common.HexToHash("0x1234")
				},
			},
			{
				name: "proof node",
				edit: func(pkg *ReceiptProofPackage) {
					pkg.ProofNodes[0] = mutateHexNode(t, pkg.ProofNodes[0])
				},
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				pkg := cloneReceiptPackage(fixture)
				tt.edit(&pkg)
				if err := VerifyReceiptProofPackage(&pkg); err == nil {
					t.Fatal("expected tampered package to fail verification")
				}
			})
		}
	})

	t.Run("event claim mismatches fail", func(t *testing.T) {
		fixture := mustLoadReceiptFixture(t)
		tests := []struct {
			name string
			edit func(*ReceiptProofPackage)
		}{
			{
				name: "log index out of range",
				edit: func(pkg *ReceiptProofPackage) {
					pkg.LogIndex = uint(len(pkg.Event.Topics) + 10)
				},
			},
			{
				name: "event address",
				edit: func(pkg *ReceiptProofPackage) {
					pkg.Event.Address = common.HexToAddress("0x9999999999999999999999999999999999999999")
				},
			},
			{
				name: "event topic",
				edit: func(pkg *ReceiptProofPackage) {
					pkg.Event.Topics[0] = common.HexToHash("0xbeef")
				},
			},
			{
				name: "event data",
				edit: func(pkg *ReceiptProofPackage) {
					pkg.Event.Data = proofutil.CanonicalBytes([]byte{0xaa})
				},
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				pkg := cloneReceiptPackage(fixture)
				tt.edit(&pkg)
				if err := VerifyReceiptProofPackage(&pkg); err == nil {
					t.Fatal("expected event mismatch to fail verification")
				}
			})
		}
	})

	t.Run("extra expectations mismatch fails", func(t *testing.T) {
		pkg := mustLoadReceiptFixture(t)
		expect := &ReceiptExpectations{
			Emitter: &[]common.Address{common.HexToAddress("0x9999999999999999999999999999999999999999")}[0],
		}
		err := VerifyReceiptProofPackageWithExpectations(&pkg, expect)
		if err == nil {
			t.Fatal("expected receipt expectations mismatch")
		}
		if !strings.Contains(err.Error(), "expected emitter mismatch") {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("decode failures surface cleanly", func(t *testing.T) {
		fixture := mustLoadReceiptFixture(t)
		tests := []struct {
			name string
			edit func(*ReceiptProofPackage)
			want string
		}{
			{
				name: "bad receipt rlp",
				edit: func(pkg *ReceiptProofPackage) {
					pkg.ReceiptRLP = nil
				},
				want: "decode claimed receipt",
			},
			{
				name: "bad transaction rlp",
				edit: func(pkg *ReceiptProofPackage) {
					pkg.TransactionRLP = nil
				},
				want: "decode claimed transaction",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				pkg := cloneReceiptPackage(fixture)
				tt.edit(&pkg)
				err := VerifyReceiptProofPackage(&pkg)
				if err == nil {
					t.Fatal("expected decode failure")
				}
				if !strings.Contains(err.Error(), tt.want) {
					t.Fatalf("unexpected error: %v", err)
				}
			})
		}
	})
}
