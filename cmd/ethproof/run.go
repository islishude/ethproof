package main

import (
	"errors"
	"fmt"
	"os"
)

func runMain(args []string) int {
	if err := run(args); err != nil {
		renderError(err)
		return exitCode(err)
	}
	return 0
}

func run(args []string) error {
	if len(args) == 0 {
		return newUsageError("missing command")
	}

	switch args[0] {
	case "generate":
		return runGenerate(args[1:])
	case "verify":
		return runVerify(args[1:])
	default:
		return newUsageError("unknown command %q", args[0])
	}
}

func renderError(err error) {
	var usageErr usageError
	if errors.As(err, &usageErr) {
		if usageErr.message != "" {
			fmt.Fprintf(os.Stderr, "error: %s\n\n", usageErr.message)
		}
		fmt.Fprint(os.Stderr, usageText)
		return
	}

	fmt.Fprintf(os.Stderr, "error: %v\n", err)
}

func exitCode(err error) int {
	var usageErr usageError
	if errors.As(err, &usageErr) {
		return 2
	}
	return 1
}
