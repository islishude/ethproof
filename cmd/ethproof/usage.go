package main

import (
	"errors"
	"fmt"
)

const usageText = `Usage:
  ethproof generate state   [--config config.json] [--rpc URL ...] [--min-rpcs N] [--block N] [--account 0xADDR] [--slot 0xSLOT] [--out state.json] [--log-level LEVEL] [--log-format text|json]
  ethproof generate receipt [--config config.json] [--rpc URL ...] [--min-rpcs N] [--tx 0xHASH] [--log-index N] [--out receipt.json] [--log-level LEVEL] [--log-format text|json]
  ethproof generate tx      [--config config.json] [--rpc URL ...] [--min-rpcs N] [--tx 0xHASH] [--out tx.json] [--log-level LEVEL] [--log-format text|json]

  ethproof verify state   [--config config.json] [--rpc URL ...] [--min-rpcs N] [--proof state.json] [--log-level LEVEL] [--log-format text|json]
  ethproof verify receipt [--config config.json] [--rpc URL ...] [--min-rpcs N] [--proof receipt.json] [--expect-emitter 0xADDR] [--expect-topic 0xHASH] [--expect-data 0xDATA] [--log-level LEVEL] [--log-format text|json]
  ethproof verify tx      [--config config.json] [--rpc URL ...] [--min-rpcs N] [--proof tx.json] [--log-level LEVEL] [--log-format text|json]

Options:
  -h, --help                  Show this help message.
  --log-level LEVEL           Runtime log level: debug, info, warn, error.
  --log-format text|json      Runtime log format.
`

type usageError struct {
	message string
	help    bool
}

func (e usageError) Error() string {
	if e.help {
		return "help requested"
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

func asUsageError(err error) (usageError, bool) {
	return errors.AsType[usageError](err)
}

func isHelpArg(arg string) bool {
	return arg == "-h" || arg == "--help"
}
