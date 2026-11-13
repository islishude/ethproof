package main

import (
	"context"

	"github.com/islishude/ethproof/proof"
)

type commandDeps struct {
	generateState       func(context.Context, string, proof.StateProofRequest) (*proof.StateProofPackage, error)
	generateReceipt     func(context.Context, string, proof.ReceiptProofRequest) (*proof.ReceiptProofPackage, error)
	generateTransaction func(context.Context, string, proof.TransactionProofRequest) (*proof.TransactionProofPackage, error)
	verifyState         func(context.Context, *proof.StateProofPackage, string) error
	verifyReceipt       func(context.Context, *proof.ReceiptProofPackage, *proof.ReceiptExpectations, string) error
	verifyTransaction   func(context.Context, *proof.TransactionProofPackage, string) error
	loadJSON            func(string, any) error
	saveJSON            func(string, any) error
}

var cliDeps = commandDeps{
	generateState:       proof.GenerateStateProofFromRPC,
	generateReceipt:     proof.GenerateReceiptProofFromRPC,
	generateTransaction: proof.GenerateTransactionProofFromRPC,
	verifyState:         proof.VerifyStateProofPackageAgainstRPC,
	verifyReceipt:       proof.VerifyReceiptProofPackageWithExpectationsAgainstRPC,
	verifyTransaction:   proof.VerifyTransactionProofPackageAgainstRPC,
	loadJSON:            proof.LoadJSON,
	saveJSON:            proof.SaveJSON,
}
