package proof

import (
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/islishude/ethproof/internal/proofutil"
)

// BlockContextFromHeader projects a header into proof metadata. It does not establish trust or finality.
func BlockContextFromHeader(chainID *big.Int, header *types.Header) (BlockContext, error) {
	if header == nil {
		return BlockContext{}, fmt.Errorf("block header is nil")
	}
	if header.Number == nil {
		return BlockContext{}, fmt.Errorf("block header number is nil")
	}
	if !header.Number.IsUint64() {
		return BlockContext{}, fmt.Errorf("block header number is outside uint64 range")
	}
	chainIDValue, err := proofutil.ChainIDFromBig(chainID)
	if err != nil {
		return BlockContext{}, err
	}
	return BlockContext{
		ChainID:          chainIDValue,
		BlockNumber:      header.Number.Uint64(),
		BlockHash:        header.Hash(),
		ParentHash:       header.ParentHash,
		StateRoot:        header.Root,
		TransactionsRoot: header.TxHash,
		ReceiptsRoot:     header.ReceiptHash,
	}, nil
}

func validateBlockContext(block BlockContext) error {
	if block.ChainID == nil {
		return fmt.Errorf("proof package block chain id is required")
	}
	return nil
}
