package main

import (
	"crypto/ecdsa"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/islishude/ethproof/internal/proofutil"
	"github.com/islishude/ethproof/proof"
)

func fixedPrivateKey(hexKey string) (*ecdsa.PrivateKey, error) {
	return crypto.HexToECDSA(hexKey)
}

func buildBlockContext(header blockSnapshotHeader, consensus SourceConsensus) proof.BlockContext {
	return proof.BlockContext{
		ChainID:          proofutil.CloneChainID(header.ChainID),
		BlockNumber:      header.BlockNumber,
		BlockHash:        header.BlockHash,
		ParentHash:       header.ParentHash,
		StateRoot:        header.StateRoot,
		TransactionsRoot: header.TransactionsRoot,
		ReceiptsRoot:     header.ReceiptsRoot,
		SourceConsensus:  consensus,
	}
}

func sourceConsensus(mode string, rpcs []string, digests []ConsensusDigest, fields []ConsensusField) SourceConsensus {
	outRPCs := append([]string{}, rpcs...)
	outDigests := append([]ConsensusDigest{}, digests...)
	outFields := append([]ConsensusField{}, fields...)
	return SourceConsensus{
		Mode:    mode,
		RPCs:    outRPCs,
		Digests: outDigests,
		Fields:  outFields,
	}
}
