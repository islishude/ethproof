package main

import (
	"context"
	"fmt"
	"time"

	"github.com/islishude/ethproof/proof"
)

const verifyProofTimeout = 2 * time.Minute

func runVerify(args []string) error {
	if len(args) == 0 {
		return newUsageError("missing verify subcommand")
	}
	if isHelpArg(args[0]) {
		return newHelpError()
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
	logger := newCommandLogger(cfg.Logging).With(
		"command", "verify",
		"proof_type", "state",
	)
	ctx, cancel := context.WithTimeout(context.Background(), verifyProofTimeout)
	defer cancel()
	ctx = proof.WithLogger(ctx, logger)

	var pkg proof.StateProofPackage
	if err := proof.LoadJSON(cfg.ProofPath, &pkg); err != nil {
		return wrapRuntimeError(logger, fmt.Errorf("read state proof json: %w", err))
	}
	if err := proof.VerifyStateProofPackageAgainstRPCs(ctx, &pkg, cfg.VerifyRequest); err != nil {
		return wrapRuntimeError(logger, fmt.Errorf("verify state proof: %w", err))
	}

	logger.Info("state proof verification succeeded", "proof_path", cfg.ProofPath, "block_number", pkg.Block.BlockNumber)
	return nil
}

func runVerifyReceipt(args []string) error {
	cfg, err := parseVerifyReceiptArgs(args)
	if err != nil {
		return err
	}
	logger := newCommandLogger(cfg.Logging).With(
		"command", "verify",
		"proof_type", "receipt",
	)
	ctx, cancel := context.WithTimeout(context.Background(), verifyProofTimeout)
	defer cancel()
	ctx = proof.WithLogger(ctx, logger)

	var pkg proof.ReceiptProofPackage
	if err := proof.LoadJSON(cfg.ProofPath, &pkg); err != nil {
		return wrapRuntimeError(logger, fmt.Errorf("read receipt proof json: %w", err))
	}
	if err := proof.VerifyReceiptProofPackageWithExpectationsAgainstRPCs(ctx, &pkg, cfg.Expectations, cfg.VerifyRequest); err != nil {
		return wrapRuntimeError(logger, fmt.Errorf("verify receipt proof: %w", err))
	}

	logger.Info("receipt proof verification succeeded", "proof_path", cfg.ProofPath, "block_number", pkg.Block.BlockNumber)
	return nil
}

func runVerifyTransaction(args []string) error {
	cfg, err := parseVerifyTransactionArgs(args)
	if err != nil {
		return err
	}
	logger := newCommandLogger(cfg.Logging).With(
		"command", "verify",
		"proof_type", "transaction",
	)
	ctx, cancel := context.WithTimeout(context.Background(), verifyProofTimeout)
	defer cancel()
	ctx = proof.WithLogger(ctx, logger)

	var pkg proof.TransactionProofPackage
	if err := proof.LoadJSON(cfg.ProofPath, &pkg); err != nil {
		return wrapRuntimeError(logger, fmt.Errorf("read transaction proof json: %w", err))
	}
	if err := proof.VerifyTransactionProofPackageAgainstRPCs(ctx, &pkg, cfg.VerifyRequest); err != nil {
		return wrapRuntimeError(logger, fmt.Errorf("verify transaction proof: %w", err))
	}

	logger.Info("transaction proof verification succeeded", "proof_path", cfg.ProofPath, "block_number", pkg.Block.BlockNumber)
	return nil
}
