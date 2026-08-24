package main

import (
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"io"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/islishude/ethproof/proof"
)

type multiStringFlag []string

type parseContext struct {
	seen    map[string]bool
	fileCfg *cliConfig
}

func (m *multiStringFlag) String() string {
	return fmt.Sprintf("%v", []string(*m))
}

func (m *multiStringFlag) Set(value string) error {
	*m = append(*m, value)
	return nil
}

func newFlagSet(name string) *flag.FlagSet {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	return fs
}

func prepareParse(fs *flag.FlagSet, args []string, configPath *string, parseErrPrefix string) (parseContext, error) {
	if err := parseFlagSet(fs, args); err != nil {
		if _, ok := asUsageError(err); ok {
			return parseContext{}, err
		}
		return parseContext{}, newUsageError("%s: %v", parseErrPrefix, err)
	}
	if err := ensureNoPositionalArgs(fs); err != nil {
		return parseContext{}, err
	}
	fileCfg, err := loadCLIConfig(*configPath)
	if err != nil {
		return parseContext{}, newUsageError("%v", err)
	}
	return parseContext{
		seen:    visitedFlags(fs),
		fileCfg: fileCfg,
	}, nil
}

func parseFlagSet(fs *flag.FlagSet, args []string) error {
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return newHelpError()
		}
		return err
	}
	return nil
}

func ensureNoPositionalArgs(fs *flag.FlagSet) error {
	if fs.NArg() == 0 {
		return nil
	}
	return newUsageError("%s does not accept positional arguments: %s", fs.Name(), strings.Join(fs.Args(), " "))
}

func visitedFlags(fs *flag.FlagSet) map[string]bool {
	out := make(map[string]bool)
	fs.Visit(func(f *flag.Flag) {
		out[f.Name] = true
	})
	return out
}

func mergeString(seen map[string]bool, flagName string, flagValue string, configValue string, defaultValue string) string {
	if seen[flagName] {
		return flagValue
	}
	if configValue != "" {
		return configValue
	}
	return defaultValue
}

func mergeStringSlice(seen map[string]bool, flagName string, flagValue []string, configValue []string) []string {
	if seen[flagName] {
		return append([]string(nil), flagValue...)
	}
	return append([]string(nil), configValue...)
}

func mergeInt(seen map[string]bool, flagName string, flagValue int, configValue *int, defaultValue int) int {
	if seen[flagName] {
		return flagValue
	}
	if configValue != nil {
		return *configValue
	}
	return defaultValue
}

func mergeUint64(seen map[string]bool, flagName string, flagValue uint64, configValue *uint64, defaultValue uint64) uint64 {
	if seen[flagName] {
		return flagValue
	}
	if configValue != nil {
		return *configValue
	}
	return defaultValue
}

func mergeRPCInputs(seen map[string]bool, rpcValues multiStringFlag, minRPCs int, configRPCs []string, configMinRPCs *int) ([]string, int) {
	return mergeStringSlice(seen, "rpc", rpcValues, configRPCs), mergeInt(seen, "min-rpcs", minRPCs, configMinRPCs, proof.DefaultMinRPCSources)
}

func validateRPCInputs(rpcURLs []string, minRPCs int, missingMessage string) error {
	if len(rpcURLs) == 0 {
		return newUsageError("%s", missingMessage)
	}
	if minRPCs < 1 {
		return newUsageError("--min-rpcs must be at least 1")
	}
	if len(rpcURLs) < minRPCs {
		return newUsageError("--min-rpcs=%d requires at least %d rpc values, got %d", minRPCs, minRPCs, len(rpcURLs))
	}
	return nil
}

func buildReceiptExpectations(expectEmitterHex string, expectDataHex string, topics []string) (*proof.ReceiptExpectations, error) {
	var expect proof.ReceiptExpectations
	if expectEmitterHex != "" {
		addr, err := parseAddressStrict(expectEmitterHex, "expected emitter")
		if err != nil {
			return nil, err
		}
		expect.Emitter = &addr
	}
	if expectDataHex != "" {
		data, err := parseHexDataStrict(expectDataHex, "expected data")
		if err != nil {
			return nil, err
		}
		expect.Data = data
	}
	for i, topic := range topics {
		parsed, err := parseHashStrict(topic, fmt.Sprintf("expected topic %d", i))
		if err != nil {
			return nil, err
		}
		expect.Topics = append(expect.Topics, parsed)
	}
	if expect.Emitter == nil && expect.Data == nil && len(expect.Topics) == 0 {
		return nil, nil
	}
	return &expect, nil
}

func parseAddressStrict(raw string, field string) (common.Address, error) {
	value := strings.TrimSpace(raw)
	if !has0xPrefix(value) || !common.IsHexAddress(value) {
		return common.Address{}, fmt.Errorf("%s must be a 20-byte 0x-prefixed hex address", field)
	}
	return common.HexToAddress(value), nil
}

func parseHashStrict(raw string, field string) (common.Hash, error) {
	value := strings.TrimSpace(raw)
	if !has0xPrefix(value) || !common.IsHexHash(value) {
		return common.Hash{}, fmt.Errorf("%s must be a 32-byte 0x-prefixed hex value", field)
	}
	return common.HexToHash(value), nil
}

func parseStorageSlotStrict(raw string) (common.Hash, error) {
	value := strings.TrimSpace(raw)
	if !has0xPrefix(value) {
		return common.Hash{}, fmt.Errorf("storage slot must be 0x-prefixed hex")
	}
	digits := value[2:]
	if len(digits) == 0 || len(digits) > 2*common.HashLength {
		return common.Hash{}, fmt.Errorf("storage slot must contain between 1 and 64 hex digits")
	}
	if len(digits)%2 == 1 {
		digits = "0" + digits
	}
	decoded, err := hex.DecodeString(digits)
	if err != nil {
		return common.Hash{}, fmt.Errorf("storage slot is not valid hex: %w", err)
	}
	return common.BytesToHash(decoded), nil
}

func parseHexDataStrict(raw string, field string) ([]byte, error) {
	value := strings.TrimSpace(raw)
	if !has0xPrefix(value) {
		return nil, fmt.Errorf("%s must be 0x-prefixed hex data", field)
	}
	decoded, err := hexutil.Decode(value)
	if err != nil {
		return nil, fmt.Errorf("%s is not valid even-length hex data: %w", field, err)
	}
	return decoded, nil
}

func has0xPrefix(value string) bool {
	return len(value) >= 2 && value[0] == '0' && (value[1] == 'x' || value[1] == 'X')
}
