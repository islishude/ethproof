package main

import (
	"context"

	"github.com/islishude/ethproof/proof"
)

type commandDeps struct {
	generateState       func(context.Context, proof.StateProofRequest) (*proof.StateProofPackage, error)
	generateReceipt     func(context.Context, proof.ReceiptProofRequest) (*proof.ReceiptProofPackage, error)
	generateTransaction func(context.Context, proof.TransactionProofRequest) (*proof.TransactionProofPackage, error)
	verifyState         func(context.Context, *proof.StateProofPackage, proof.VerifyRPCRequest) error
	verifyReceipt       func(context.Context, *proof.ReceiptProofPackage, *proof.ReceiptExpectations, proof.VerifyRPCRequest) error
	verifyTransaction   func(context.Context, *proof.TransactionProofPackage, proof.VerifyRPCRequest) error
	loadJSON            func(string, any) error
	saveJSON            func(string, any) error
}

var cliDeps = commandDeps{
	generateState:       proof.GenerateStateProof,
	generateReceipt:     proof.GenerateReceiptProof,
	generateTransaction: proof.GenerateTransactionProof,
	verifyState:         proof.VerifyStateProofPackageAgainstRPCs,
	verifyReceipt:       proof.VerifyReceiptProofPackageWithExpectationsAgainstRPCs,
	verifyTransaction:   proof.VerifyTransactionProofPackageAgainstRPCs,
	loadJSON:            proof.LoadJSON,
	saveJSON:            proof.SaveJSON,
}
