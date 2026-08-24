package main

import "github.com/islishude/ethproof/proof"

type resolveSlotConfig struct {
	CompilerOutput string
	Contract       string
	Variable       string
	Format         proof.StorageLayoutFormat
	Out            string
}

func parseResolveSlotArgs(args []string) (resolveSlotConfig, error) {
	fs := newFlagSet("resolve slot")
	compilerOutput := fs.String("compiler-output", "", "compiler output json path")
	contract := fs.String("contract", "", "contract name or source:contract selector (artifact/build-info only)")
	variable := fs.String("var", "", "storage variable query")
	format := fs.String("format", string(proof.StorageLayoutFormatAuto), "compiler output format (auto|layout|artifact|build-info)")
	out := fs.String("out", "", "write resolved slot json to file")

	if err := parseFlagSet(fs, args); err != nil {
		if _, ok := asUsageError(err); ok {
			return resolveSlotConfig{}, err
		}
		return resolveSlotConfig{}, newUsageError("parse resolve slot args: %v", err)
	}
	if err := ensureNoPositionalArgs(fs); err != nil {
		return resolveSlotConfig{}, err
	}

	if *compilerOutput == "" {
		return resolveSlotConfig{}, newUsageError("resolve slot requires --compiler-output")
	}
	if *variable == "" {
		return resolveSlotConfig{}, newUsageError("resolve slot requires --var")
	}

	resolvedFormat, err := proof.ParseStorageLayoutFormat(*format)
	if err != nil {
		return resolveSlotConfig{}, newUsageError("%v", err)
	}
	switch resolvedFormat {
	case proof.StorageLayoutFormatArtifact, proof.StorageLayoutFormatBuildInfo:
		if *contract == "" {
			return resolveSlotConfig{}, newUsageError("resolve slot requires --contract for %s input", resolvedFormat)
		}
	case proof.StorageLayoutFormatLayout:
		if *contract != "" {
			return resolveSlotConfig{}, newUsageError("resolve slot does not accept --contract for raw layout input")
		}
	}
	return resolveSlotConfig{
		CompilerOutput: *compilerOutput,
		Contract:       *contract,
		Variable:       *variable,
		Format:         resolvedFormat,
		Out:            *out,
	}, nil
}
