package main

import (
	"fmt"
	"os"
)

func runMain(args []string) int {
	if err := run(args); err != nil {
		if _, ok := asUsageError(err); ok {
			renderError(err)
		} else {
			loggerForError(err).Error(err.Error())
		}
		return exitCode(err)
	}
	return 0
}

func run(args []string) error {
	if len(args) == 0 {
		return newUsageError("missing command")
	}
	if isHelpArg(args[0]) {
		return newHelpError()
	}

	switch args[0] {
	case "generate":
		return runGenerate(args[1:])
	case "resolve":
		return runResolve(args[1:])
	case "verify":
		return runVerify(args[1:])
	default:
		return newUsageError("unknown command %q", args[0])
	}
}

func renderError(err error) {
	if usageErr, ok := asUsageError(err); ok {
		if usageErr.help {
			_, _ = fmt.Fprint(os.Stdout, usageText)
			return
		}
		if usageErr.message != "" {
			fmt.Fprintf(os.Stderr, "error: %s\n\n", usageErr.message)
		}
		fmt.Fprint(os.Stderr, usageText)
		return
	}
}

func exitCode(err error) int {
	if usageErr, ok := asUsageError(err); ok {
		if usageErr.help {
			return 0
		}
		return 2
	}
	return 1
}
