package main

import "fmt"

const usageText = `Usage:
  ethproof generate state   --rpc URL [--rpc URL ...] --min-rpcs N --block N --account 0xADDR --slot 0xSLOT --out state.json
  ethproof generate receipt --rpc URL [--rpc URL ...] --min-rpcs N --tx 0xHASH --log-index N --out receipt.json
  ethproof generate tx      --rpc URL [--rpc URL ...] --min-rpcs N --tx 0xHASH --out tx.json

  ethproof verify state   --proof state.json
  ethproof verify receipt --proof receipt.json [--expect-emitter 0xADDR] [--expect-topic 0xHASH] [--expect-data 0xDATA]
  ethproof verify tx      --proof tx.json
`

type usageError struct {
	message string
}

func (e usageError) Error() string {
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
