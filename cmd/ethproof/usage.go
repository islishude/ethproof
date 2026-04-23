package main

import (
	"errors"
	"fmt"
	"runtime/debug"
)

const usageText = `Usage:
  ethproof generate state   [--config config.json] [--rpc URL ...] [--min-rpcs N] [--block N] [--account 0xADDR] [--slot 0xSLOT ...] [--out state.json]
  ethproof generate receipt [--config config.json] [--rpc URL ...] [--min-rpcs N] [--tx 0xHASH] [--log-index N] [--out receipt.json]
  ethproof generate tx      [--config config.json] [--rpc URL ...] [--min-rpcs N] [--tx 0xHASH] [--out tx.json]
  ethproof resolve slot     [--compiler-output path] [--contract name-or-source:name] [--var query] [--format auto|layout|artifact|build-info] [--out result.json]

  ethproof verify state   [--config config.json] [--rpc URL ...] [--min-rpcs N] [--proof state.json]
  ethproof verify receipt [--config config.json] [--rpc URL ...] [--min-rpcs N] [--proof receipt.json] [--expect-emitter 0xADDR] [--expect-topic 0xHASH] [--expect-data 0xDATA]
  ethproof verify tx      [--config config.json] [--rpc URL ...] [--min-rpcs N] [--proof tx.json]

Options:
  -h, --help                  Show this help message.
`

type usageError struct {
	message string
	help    bool
	version bool
}

func (e usageError) Error() string {
	if e.help {
		return "help requested"
	}
	if e.version {
		return "version requested"
	}
	if e.message == "" {
		return "invalid usage"
	}
	return e.message
}

func newUsageError(format string, args ...any) error {
	if format == "" {
		return usageError{}
	}
	return usageError{message: fmt.Sprintf(format, args...)}
}

func newHelpError() error {
	return usageError{help: true}
}

func newVersionError() error {
	return usageError{version: true}
}

func asUsageError(err error) (usageError, bool) {
	return errors.AsType[usageError](err)
}

func isHelpArg(arg string) bool {
	return arg == "-h" || arg == "--help"
}

func isVersionArg(arg string) bool {
	return arg == "-v" || arg == "--version"
}

// buildVersion returns the embedded module version for the running binary.
func buildVersion() string {
	info, ok := debug.ReadBuildInfo()
	if !ok || info == nil || info.Main.Version == "" {
		return "devel"
	}
	return info.Main.Version
}
