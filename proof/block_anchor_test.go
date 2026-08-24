package proof

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/holiman/uint256"
	"github.com/islishude/ethproof/internal/proofutil"
)

func TestVerifyProofPackagesAgainstBlockAnchor(t *testing.T) {
	statePkg := mustLoadStateFixture(t)
	receiptPkg := mustLoadReceiptFixture(t)
	transactionPkg := mustLoadTransactionFixture(t)

	if err := VerifyStateProofPackageAgainstBlockAnchor(&statePkg, anchorForTest(statePkg.Block)); err != nil {
		t.Fatalf("VerifyStateProofPackageAgainstBlockAnchor: %v", err)
	}
	if err := VerifyReceiptProofPackageAgainstBlockAnchor(&receiptPkg, anchorForTest(receiptPkg.Block)); err != nil {
		t.Fatalf("VerifyReceiptProofPackageAgainstBlockAnchor: %v", err)
	}
	if err := VerifyTransactionProofPackageAgainstBlockAnchor(&transactionPkg, anchorForTest(transactionPkg.Block)); err != nil {
		t.Fatalf("VerifyTransactionProofPackageAgainstBlockAnchor: %v", err)
	}
}

func TestVerifyTransactionProofPackageAgainstBlockAnchorRejectsEveryMismatch(t *testing.T) {
	pkg := mustLoadTransactionFixture(t)
	base := anchorForTest(pkg.Block)
	tests := []struct {
		name string
		edit func(*BlockAnchor)
	}{
		{name: "chain id", edit: func(anchor *BlockAnchor) { anchor.ChainID = uint256.NewInt(999) }},
		{name: "block number", edit: func(anchor *BlockAnchor) { anchor.BlockNumber++ }},
		{name: "block hash", edit: func(anchor *BlockAnchor) { anchor.BlockHash = common.HexToHash("0x01") }},
		{name: "parent hash", edit: func(anchor *BlockAnchor) { anchor.ParentHash = common.HexToHash("0x02") }},
		{name: "state root", edit: func(anchor *BlockAnchor) { anchor.StateRoot = common.HexToHash("0x03") }},
		{name: "transactions root", edit: func(anchor *BlockAnchor) { anchor.TransactionsRoot = common.HexToHash("0x04") }},
		{name: "receipts root", edit: func(anchor *BlockAnchor) { anchor.ReceiptsRoot = common.HexToHash("0x05") }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			anchor := base
			anchor.ChainID = proofutil.CloneChainID(base.ChainID)
			tt.edit(&anchor)
			if err := VerifyTransactionProofPackageAgainstBlockAnchor(&pkg, anchor); err == nil {
				t.Fatal("expected block anchor mismatch")
			}
		})
	}
}

func TestBlockAnchorFromHeader(t *testing.T) {
	header := &types.Header{
		Number:      big.NewInt(42),
		ParentHash:  common.HexToHash("0x01"),
		Root:        common.HexToHash("0x02"),
		TxHash:      common.HexToHash("0x03"),
		ReceiptHash: common.HexToHash("0x04"),
		Difficulty:  big.NewInt(1),
		GasLimit:    30_000_000,
	}
	anchor, err := BlockAnchorFromHeader(big.NewInt(11155111), header)
	if err != nil {
		t.Fatalf("BlockAnchorFromHeader: %v", err)
	}
	if anchor.BlockHash != header.Hash() || anchor.BlockNumber != 42 || anchor.ChainID.ToBig().Cmp(big.NewInt(11155111)) != 0 {
		t.Fatalf("unexpected block anchor: %+v", anchor)
	}
	if _, err := BlockAnchorFromHeader(big.NewInt(1), nil); err == nil {
		t.Fatal("expected nil header to fail")
	}
	if _, err := BlockAnchorFromHeader(nil, header); err == nil {
		t.Fatal("expected nil chain id to fail")
	}
}

func anchorForTest(block BlockContext) BlockAnchor {
	return BlockAnchor{
		ChainID:          proofutil.CloneChainID(block.ChainID),
		BlockNumber:      block.BlockNumber,
		BlockHash:        block.BlockHash,
		ParentHash:       block.ParentHash,
		StateRoot:        block.StateRoot,
		TransactionsRoot: block.TransactionsRoot,
		ReceiptsRoot:     block.ReceiptsRoot,
	}
}
