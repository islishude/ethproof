package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/islishude/ethproof/internal/logutil"
	"github.com/islishude/ethproof/proof"
)

type multiStringFlag []string

type parseContext struct {
	seen    map[string]bool
	fileCfg *cliConfig
}

type loggingFlags struct {
	level  *string
	format *string
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

func addLoggingFlags(fs *flag.FlagSet) loggingFlags {
	return loggingFlags{
		level:  fs.String("log-level", "", "log level (debug|info|warn|error)"),
		format: fs.String("log-format", "", "log format (text|json)"),
	}
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

func mergeUint(seen map[string]bool, flagName string, flagValue uint, configValue *uint, defaultValue uint) uint {
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
	return mergeStringSlice(seen, "rpc", rpcValues, configRPCs), mergeInt(seen, "min-rpcs", minRPCs, configMinRPCs, proofMinRPCsDefault())
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
		addr := common.HexToAddress(expectEmitterHex)
		expect.Emitter = &addr
	}
	if expectDataHex != "" {
		expect.Data = common.FromHex(expectDataHex)
	}
	for _, topic := range topics {
		expect.Topics = append(expect.Topics, common.HexToHash(topic))
	}
	if expect.Emitter == nil && expect.Data == nil && len(expect.Topics) == 0 {
		return nil, nil
	}
	return &expect, nil
}

func proofMinRPCsDefault() int {
	return 3
}

func resolveLoggingConfig(seen map[string]bool, flags loggingFlags, fileCfg *cliLoggingConfigFile) (logutil.Config, error) {
	var configLevel string
	var configFormat string
	if fileCfg != nil {
		configLevel = fileCfg.Level
		configFormat = fileCfg.Format
	}
	cfg, err := logutil.NormalizeConfig(logutil.Config{
		Level:  mergeString(seen, "log-level", *flags.level, configLevel, logutil.DefaultLevel),
		Format: mergeString(seen, "log-format", *flags.format, configFormat, logutil.DefaultFormat),
	})
	if err != nil {
		return logutil.Config{}, newUsageError("%v", err)
	}
	return cfg, nil
}
