package main

import (
	"context"
	"fmt"
	"time"
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

	ctx, cancel := context.WithTimeout(context.Background(), stateProofTimeout)
	defer cancel()

	pkg, err := cliDeps.generateState(ctx, cfg.Request)
	if err != nil {
		return fmt.Errorf("generate state proof: %w", err)
	}
	if err := cliDeps.saveJSON(cfg.Out, pkg); err != nil {
		return fmt.Errorf("write state proof: %w", err)
	}
	printStatusLine(
		"wrote state proof to %s (block %d, account %s, %d %s)",
		cfg.Out,
		pkg.Block.BlockNumber,
		pkg.Account,
		len(pkg.StorageProofs),
		pluralize(len(pkg.StorageProofs), "slot", "slots"),
	)
	return nil
}

func runGenerateReceipt(args []string) error {
	cfg, err := parseGenerateReceiptArgs(args)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), receiptProofTimeout)
	defer cancel()

	pkg, err := cliDeps.generateReceipt(ctx, cfg.Request)
	if err != nil {
		return fmt.Errorf("generate receipt proof: %w", err)
	}
	if err := cliDeps.saveJSON(cfg.Out, pkg); err != nil {
		return fmt.Errorf("write receipt proof: %w", err)
	}
	printStatusLine(
		"wrote receipt proof to %s (block %d, tx %s, log %d)",
		cfg.Out,
		pkg.Block.BlockNumber,
		pkg.TxHash,
		pkg.LogIndex,
	)
	return nil
}

func runGenerateTransaction(args []string) error {
	cfg, err := parseGenerateTransactionArgs(args)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), transactionProofTimeout)
	defer cancel()

	pkg, err := cliDeps.generateTransaction(ctx, cfg.Request)
	if err != nil {
		return fmt.Errorf("generate transaction proof: %w", err)
	}
	if err := cliDeps.saveJSON(cfg.Out, pkg); err != nil {
		return fmt.Errorf("write transaction proof: %w", err)
	}
	printStatusLine(
		"wrote transaction proof to %s (block %d, tx %s)",
		cfg.Out,
		pkg.Block.BlockNumber,
		pkg.TxHash,
	)
	return nil
}
