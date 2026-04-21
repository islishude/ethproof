package main

import (
	"fmt"

	"github.com/islishude/ethproof/proof"
)

func runVerify(args []string) error {
	if len(args) == 0 {
		return newUsageError("missing verify subcommand")
	}

	switch args[0] {
	case "state":
		return runVerifyState(args[1:])
	case "receipt":
		return runVerifyReceipt(args[1:])
	case "tx":
		return runVerifyTransaction(args[1:])
	default:
		return newUsageError("unknown verify subcommand %q", args[0])
	}
}

func runVerifyState(args []string) error {
	cfg, err := parseVerifyStateArgs(args)
	if err != nil {
		return err
	}

	var pkg proof.StateProofPackage
	if err := proof.LoadJSON(cfg.ProofPath, &pkg); err != nil {
		return fmt.Errorf("read state proof json: %w", err)
	}
	if err := proof.VerifyStateProofPackage(&pkg); err != nil {
		return fmt.Errorf("verify state proof: %w", err)
	}

	fmt.Println("state proof verification succeeded")
	return nil
}

func runVerifyReceipt(args []string) error {
	cfg, err := parseVerifyReceiptArgs(args)
	if err != nil {
		return err
	}

	var pkg proof.ReceiptProofPackage
	if err := proof.LoadJSON(cfg.ProofPath, &pkg); err != nil {
		return fmt.Errorf("read receipt proof json: %w", err)
	}
	if err := proof.VerifyReceiptProofPackageWithExpectations(&pkg, cfg.Expectations); err != nil {
		return fmt.Errorf("verify receipt proof: %w", err)
	}

	fmt.Println("receipt proof verification succeeded")
	return nil
}

func runVerifyTransaction(args []string) error {
	cfg, err := parseVerifyTransactionArgs(args)
	if err != nil {
		return err
	}

	var pkg proof.TransactionProofPackage
	if err := proof.LoadJSON(cfg.ProofPath, &pkg); err != nil {
		return fmt.Errorf("read transaction proof json: %w", err)
	}
	if err := proof.VerifyTransactionProofPackage(&pkg); err != nil {
		return fmt.Errorf("verify transaction proof: %w", err)
	}

	fmt.Println("transaction proof verification succeeded")
	return nil
}
