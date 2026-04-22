package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/islishude/ethproof/proof"
)

func runResolve(args []string) error {
	if len(args) == 0 {
		return newUsageError("missing resolve subcommand")
	}
	if isHelpArg(args[0]) {
		return newHelpError()
	}

	switch args[0] {
	case "slot":
		return runResolveSlot(args[1:])
	default:
		return newUsageError("unknown resolve subcommand %q", args[0])
	}
}

func runResolveSlot(args []string) error {
	cfg, err := parseResolveSlotArgs(args)
	if err != nil {
		return err
	}

	layout, err := proof.LoadStorageLayout(cfg.CompilerOutput, cfg.Contract, cfg.Format)
	if err != nil {
		return fmt.Errorf("load storage layout: %w", err)
	}
	resolution, err := proof.ResolveStorageSlots(layout, cfg.Variable)
	if err != nil {
		return fmt.Errorf("resolve storage slot: %w", err)
	}

	if cfg.Out != "" {
		if err := proof.SaveJSON(cfg.Out, resolution); err != nil {
			return fmt.Errorf("write resolved slot json: %w", err)
		}
		return nil
	}

	b, err := json.MarshalIndent(resolution, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal resolved slot json: %w", err)
	}
	if _, err := os.Stdout.Write(append(b, '\n')); err != nil {
		return fmt.Errorf("write resolved slot json: %w", err)
	}
	return nil
}
