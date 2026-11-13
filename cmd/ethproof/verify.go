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
	ctx, cancel := context.WithTimeout(context.Background(), verificationTimeout(cfg.Finality))
	defer cancel()

	var pkg proof.StateProofPackage
	if err := cliDeps.loadJSON(cfg.ProofPath, &pkg); err != nil {
		return fmt.Errorf("read state proof json: %w", err)
	}
	if err := cliDeps.verifyState(ctx, &pkg, cfg.RPCURL); err != nil {
		return fmt.Errorf("verify state proof: %w", err)
	}

	if err := verifyFinality(ctx, pkg.Block, cfg.RPCURL, cfg.Finality); err != nil {
		return err
	}

	printStatusLine(
		"verified state proof %s (block %d, account %s, %d %s)",
		cfg.ProofPath,
		pkg.Block.BlockNumber,
		pkg.Account,
		len(pkg.StorageProofs),
		pluralize(len(pkg.StorageProofs), "slot", "slots"),
	)
	return nil
}

func runVerifyReceipt(args []string) error {
	cfg, err := parseVerifyReceiptArgs(args)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), verificationTimeout(cfg.Finality))
	defer cancel()

	var pkg proof.ReceiptProofPackage
	if err := cliDeps.loadJSON(cfg.ProofPath, &pkg); err != nil {
		return fmt.Errorf("read receipt proof json: %w", err)
	}
	if err := cliDeps.verifyReceipt(ctx, &pkg, cfg.Expectations, cfg.RPCURL); err != nil {
		return fmt.Errorf("verify receipt proof: %w", err)
	}

	if err := verifyFinality(ctx, pkg.Block, cfg.RPCURL, cfg.Finality); err != nil {
		return err
	}

	printStatusLine(
		"verified receipt proof %s (block %d, tx %s, log %d)",
		cfg.ProofPath,
		pkg.Block.BlockNumber,
		pkg.TxHash,
		pkg.LogIndex,
	)
	return nil
}

func runVerifyTransaction(args []string) error {
	cfg, err := parseVerifyTransactionArgs(args)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), verificationTimeout(cfg.Finality))
	defer cancel()

	var pkg proof.TransactionProofPackage
	if err := cliDeps.loadJSON(cfg.ProofPath, &pkg); err != nil {
		return fmt.Errorf("read transaction proof json: %w", err)
	}
	if err := cliDeps.verifyTransaction(ctx, &pkg, cfg.RPCURL); err != nil {
		return fmt.Errorf("verify transaction proof: %w", err)
	}

	if err := verifyFinality(ctx, pkg.Block, cfg.RPCURL, cfg.Finality); err != nil {
		return err
	}

	printStatusLine(
		"verified transaction proof %s (block %d, tx %s)",
		cfg.ProofPath,
		pkg.Block.BlockNumber,
		pkg.TxHash,
	)
	return nil
}

func verificationTimeout(cfg *proof.FinalityConfig) time.Duration {
	if cfg == nil {
		return verifyProofTimeout
	}
	if cfg.TimeoutSeconds == 0 {
		return 5 * time.Minute
	}
	return time.Duration(cfg.TimeoutSeconds) * time.Second
}
func verifyFinality(ctx context.Context, block proof.BlockContext, rpc string, cfg *proof.FinalityConfig) error {
	if cfg == nil {
		return nil
	}
	anchor, err := cliDeps.verifyFinality(ctx, block, rpc, *cfg, func(event proof.FinalityTrustEvent) {
		printStatusLine("accepted finalized checkpoint %s (slot %d, previous %s)", event.Root, event.Slot, event.PreviousRoot)
	})
	if err != nil {
		return fmt.Errorf("verify consensus finality: %w", err)
	}
	printStatusLine("authenticated finalized execution block %d (%s)", anchor.BlockNumber, anchor.BlockHash)
	return nil
}
