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

	ctx, cancel := context.WithTimeout(context.Background(), stateProofTimeout)
	defer cancel()

	pkg, err := proof.GenerateStateProof(ctx, cfg.Request)
	if err != nil {
		return fmt.Errorf("generate state proof: %w", err)
	}
	if err := proof.SaveJSON(cfg.Out, pkg); err != nil {
		return fmt.Errorf("write state proof: %w", err)
	}

	fmt.Printf("state proof written to %s\n", cfg.Out)
	fmt.Printf("block=%d account=%s slot=%s stateRoot=%s\n", pkg.Block.BlockNumber, pkg.Account, pkg.Slot, pkg.Block.StateRoot)
	return nil
}

func runGenerateReceipt(args []string) error {
	cfg, err := parseGenerateReceiptArgs(args)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), receiptProofTimeout)
	defer cancel()

	pkg, err := proof.GenerateReceiptProof(ctx, cfg.Request)
	if err != nil {
		return fmt.Errorf("generate receipt proof: %w", err)
	}
	if err := proof.SaveJSON(cfg.Out, pkg); err != nil {
		return fmt.Errorf("write receipt proof: %w", err)
	}

	fmt.Printf("receipt proof written to %s\n", cfg.Out)
	fmt.Printf("block=%d txIndex=%d receiptsRoot=%s\n", pkg.Block.BlockNumber, pkg.TxIndex, pkg.Block.ReceiptsRoot)
	return nil
}

func runGenerateTransaction(args []string) error {
	cfg, err := parseGenerateTransactionArgs(args)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), transactionProofTimeout)
	defer cancel()

	pkg, err := proof.GenerateTransactionProof(ctx, cfg.Request)
	if err != nil {
		return fmt.Errorf("generate transaction proof: %w", err)
	}
	if err := proof.SaveJSON(cfg.Out, pkg); err != nil {
		return fmt.Errorf("write transaction proof: %w", err)
	}

	fmt.Printf("transaction proof written to %s\n", cfg.Out)
	fmt.Printf("block=%d txIndex=%d transactionsRoot=%s\n", pkg.Block.BlockNumber, pkg.TxIndex, pkg.Block.TransactionsRoot)
	return nil
}
