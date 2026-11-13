package proof

import (
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/core/types"
)

// BlockAnchorFromHeader builds an anchor from a caller-trusted chain ID and header.
// The caller remains responsible for establishing header authenticity and finality.
func BlockAnchorFromHeader(chainID *big.Int, header *types.Header) (BlockAnchor, error) {
	block, err := BlockContextFromHeader(chainID, header)
	if err != nil {
		return BlockAnchor{}, err
	}
	return BlockAnchor(block), nil
}

func verifyBlockContextAgainstAnchor(block BlockContext, anchor BlockAnchor) error {
	if err := validateBlockContext(block); err != nil {
		return err
	}
	if anchor.ChainID == nil {
		return fmt.Errorf("block anchor chain id is required")
	}
	if block.ChainID.Cmp(anchor.ChainID) != 0 {
		return fmt.Errorf("block chainId mismatch")
	}
	if block.BlockNumber != anchor.BlockNumber {
		return fmt.Errorf("blockNumber mismatch")
	}
	if block.BlockHash != anchor.BlockHash {
		return fmt.Errorf("blockHash mismatch")
	}
	if block.ParentHash != anchor.ParentHash {
		return fmt.Errorf("block parentHash mismatch")
	}
	if block.StateRoot != anchor.StateRoot {
		return fmt.Errorf("block stateRoot mismatch")
	}
	if block.TransactionsRoot != anchor.TransactionsRoot {
		return fmt.Errorf("block transactionsRoot mismatch")
	}
	if block.ReceiptsRoot != anchor.ReceiptsRoot {
		return fmt.Errorf("block receiptsRoot mismatch")
	}
	return nil
}
