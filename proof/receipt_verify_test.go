package proof

import (
	"math"
	"math/big"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/holiman/uint256"
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
				name: "transactions root",
				edit: func(pkg *ReceiptProofPackage) {
					pkg.Block.TransactionsRoot = common.HexToHash("0x5678")
				},
			},
			{
				name: "proof node",
				edit: func(pkg *ReceiptProofPackage) {
					pkg.ProofNodes[0] = mutateHexNode(t, pkg.ProofNodes[0])
				},
			},
			{
				name: "transaction proof node",
				edit: func(pkg *ReceiptProofPackage) {
					pkg.TransactionProofNodes[0] = mutateHexNode(t, pkg.TransactionProofNodes[0])
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
					pkg.LogIndex = math.MaxUint64
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

	t.Run("missing transaction proof fails closed", func(t *testing.T) {
		pkg := cloneReceiptPackage(mustLoadReceiptFixture(t))
		pkg.TransactionProofNodes = nil
		err := VerifyReceiptProofPackage(&pkg)
		if err == nil || !strings.Contains(err.Error(), "must contain transaction proof nodes") {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("substituted transaction identity fails", func(t *testing.T) {
		req, _, _ := testReceiptProofSourcesRequest(t)
		pkg, err := GenerateReceiptProofFromSources(t.Context(), req)
		if err != nil {
			t.Fatalf("GenerateReceiptProofFromSources: %v", err)
		}
		source := req.Sources[0].(*fakeReceiptSource)
		otherTx := source.block.Transactions()[0]
		otherRLP, err := proofutil.EncodeTransaction(otherTx)
		if err != nil {
			t.Fatalf("EncodeTransaction: %v", err)
		}
		pkg.TransactionRLP = otherRLP
		pkg.TxHash = otherTx.Hash()
		if err := VerifyReceiptProofPackage(pkg); err == nil {
			t.Fatal("expected transaction substitution to fail")
		}
	})

	t.Run("receipt type must match transaction type", func(t *testing.T) {
		pkg := mismatchedReceiptTypePackage(t)
		err := VerifyReceiptProofPackage(pkg)
		if err == nil || !strings.Contains(err.Error(), "receipt type mismatch") {
			t.Fatalf("unexpected error: %v", err)
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

func mismatchedReceiptTypePackage(t *testing.T) *ReceiptProofPackage {
	t.Helper()

	recipient := common.HexToAddress("0x1000000000000000000000000000000000000001")
	tx := types.NewTx(&types.LegacyTx{
		Nonce:    1,
		GasPrice: big.NewInt(1),
		Gas:      21_000,
		To:       &recipient,
		Value:    big.NewInt(1),
	})
	transactionRLP, err := proofutil.EncodeTransaction(tx)
	if err != nil {
		t.Fatalf("EncodeTransaction: %v", err)
	}
	transactionRoot := singleEntryTrieRoot(t, transactionRLP)
	_, transactionProofNodes, err := proofutil.BuildIndexTrieProof([]hexutil.Bytes{transactionRLP}, 0, transactionRoot, "transaction")
	if err != nil {
		t.Fatalf("BuildIndexTrieProof(transaction): %v", err)
	}

	eventLog := &types.Log{Address: recipient, Data: []byte{0xaa}}
	receipt := &types.Receipt{
		Type:              types.DynamicFeeTxType,
		Status:            types.ReceiptStatusSuccessful,
		CumulativeGasUsed: 21_000,
		Logs:              []*types.Log{eventLog},
	}
	receipt.Bloom = types.CreateBloom(receipt)
	receiptRLP, err := proofutil.EncodeReceipt(receipt)
	if err != nil {
		t.Fatalf("EncodeReceipt: %v", err)
	}
	receiptRoot := singleEntryTrieRoot(t, receiptRLP)
	_, receiptProofNodes, err := proofutil.BuildIndexTrieProof([]hexutil.Bytes{receiptRLP}, 0, receiptRoot, "receipt")
	if err != nil {
		t.Fatalf("BuildIndexTrieProof(receipt): %v", err)
	}

	return &ReceiptProofPackage{
		Block: BlockContext{
			ChainID:          uint256.NewInt(1),
			TransactionsRoot: transactionRoot,
			ReceiptsRoot:     receiptRoot,
		},
		TxHash:                tx.Hash(),
		TransactionRLP:        transactionRLP,
		TransactionProofNodes: transactionProofNodes,
		ReceiptRLP:            receiptRLP,
		ProofNodes:            receiptProofNodes,
		Event: EventClaim{
			Address: eventLog.Address,
			Data:    proofutil.CanonicalBytes(eventLog.Data),
		},
	}
}

func singleEntryTrieRoot(t *testing.T, entry hexutil.Bytes) common.Hash {
	t.Helper()
	tr := proofutil.MakeProofTrie()
	if err := tr.Update(proofutil.TrieIndexKey(0), entry); err != nil {
		t.Fatalf("trie update: %v", err)
	}
	return tr.Hash()
}
