package main

import (
	"context"
	"fmt"
	"time"

	"github.com/islishude/ethproof/proof"
)

const (
	stateProofTimeout       = 2 * time.Minute
	receiptProofTimeout     = 3 * time.Minute
	transactionProofTimeout = 3 * time.Minute
)

func runGenerate(args []string) error {
	if len(args) == 0 {
		return newUsageError("missing generate subcommand")
	}
	if isHelpArg(args[0]) {
		return newHelpError()
	}

	switch args[0] {
	case "state":
		return runGenerateState(args[1:])
	case "receipt":
		return runGenerateReceipt(args[1:])
	case "tx":
		return runGenerateTransaction(args[1:])
	default:
		return newUsageError("unknown generate subcommand %q", args[0])
	}
}

func runGenerateState(args []string) error {
	cfg, err := parseGenerateStateArgs(args)
	if err != nil {
		return err
	}
	logger := newCommandLogger(cfg.Logging).With(
		"command", "generate",
		"proof_type", "state",
	)

	ctx, cancel := context.WithTimeout(context.Background(), stateProofTimeout)
	defer cancel()
	ctx = proof.WithLogger(ctx, logger)

	pkg, err := proof.GenerateStateProof(ctx, cfg.Request)
	if err != nil {
		return wrapRuntimeError(logger, fmt.Errorf("generate state proof: %w", err))
	}
	if err := proof.SaveJSON(cfg.Out, pkg); err != nil {
		return wrapRuntimeError(logger, fmt.Errorf("write state proof: %w", err))
	}

	logger.Info("state proof written",
		"out_path", cfg.Out,
		"block_number", pkg.Block.BlockNumber,
		"account", pkg.Account,
		"slot", pkg.Slot,
		"state_root", pkg.Block.StateRoot,
	)
	return nil
}

func runGenerateReceipt(args []string) error {
	cfg, err := parseGenerateReceiptArgs(args)
	if err != nil {
		return err
	}
	logger := newCommandLogger(cfg.Logging).With(
		"command", "generate",
		"proof_type", "receipt",
	)

	ctx, cancel := context.WithTimeout(context.Background(), receiptProofTimeout)
	defer cancel()
	ctx = proof.WithLogger(ctx, logger)

	pkg, err := proof.GenerateReceiptProof(ctx, cfg.Request)
	if err != nil {
		return wrapRuntimeError(logger, fmt.Errorf("generate receipt proof: %w", err))
	}
	if err := proof.SaveJSON(cfg.Out, pkg); err != nil {
		return wrapRuntimeError(logger, fmt.Errorf("write receipt proof: %w", err))
	}

	logger.Info("receipt proof written",
		"out_path", cfg.Out,
		"block_number", pkg.Block.BlockNumber,
		"tx_index", pkg.TxIndex,
		"receipts_root", pkg.Block.ReceiptsRoot,
	)
	return nil
}

func runGenerateTransaction(args []string) error {
	cfg, err := parseGenerateTransactionArgs(args)
	if err != nil {
		return err
	}
	logger := newCommandLogger(cfg.Logging).With(
		"command", "generate",
		"proof_type", "transaction",
	)

	ctx, cancel := context.WithTimeout(context.Background(), transactionProofTimeout)
	defer cancel()
	ctx = proof.WithLogger(ctx, logger)

	pkg, err := proof.GenerateTransactionProof(ctx, cfg.Request)
	if err != nil {
		return wrapRuntimeError(logger, fmt.Errorf("generate transaction proof: %w", err))
	}
	if err := proof.SaveJSON(cfg.Out, pkg); err != nil {
		return wrapRuntimeError(logger, fmt.Errorf("write transaction proof: %w", err))
	}

	logger.Info("transaction proof written",
		"out_path", cfg.Out,
		"block_number", pkg.Block.BlockNumber,
		"tx_index", pkg.TxIndex,
		"transactions_root", pkg.Block.TransactionsRoot,
	)
	return nil
}
