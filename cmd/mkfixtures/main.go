package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/islishude/ethproof/proof"
)

const mkfixturesUsageText = `Usage:
  mkfixtures [--out-dir DIR]

Options:
  -h, --help                  Show this help message.
`

type fixturesConfig struct {
	OutDir string
}

func main() {
	os.Exit(runMain(os.Args[1:]))
}

func runMain(args []string) int {
	cfg, err := parseArgs(args)
	if err != nil {
		if errors.Is(err, flag.ErrHelp) {
			_, _ = fmt.Fprint(os.Stdout, mkfixturesUsageText)
			return 0
		}
		if isUsageError(err) {
			fmt.Fprintf(os.Stderr, "error: %v\n\n", err)
			_, _ = fmt.Fprint(os.Stderr, mkfixturesUsageText)
			return 2
		}
		fmt.Fprintf(os.Stderr, "error: %s\n", err)
		return 1
	}

	if err := run(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "error: %s\n", err)
		return 1
	}
	return 0
}

func run(cfg fixturesConfig) error {
	fixtures, err := BuildOfflineFixtures()
	if err != nil {
		return fmt.Errorf("build offline fixtures: %w", err)
	}
	if err := os.MkdirAll(cfg.OutDir, 0o755); err != nil {
		return fmt.Errorf("create output dir: %w", err)
	}
	if err := proof.SaveJSON(filepath.Join(cfg.OutDir, "state_fixture.json"), fixtures.State); err != nil {
		return fmt.Errorf("write state fixture: %w", err)
	}
	if err := proof.SaveJSON(filepath.Join(cfg.OutDir, "receipt_fixture.json"), fixtures.Receipt); err != nil {
		return fmt.Errorf("write receipt fixture: %w", err)
	}
	if err := proof.SaveJSON(filepath.Join(cfg.OutDir, "transaction_fixture.json"), fixtures.Transaction); err != nil {
		return fmt.Errorf("write transaction fixture: %w", err)
	}
	_, _ = fmt.Fprintf(os.Stderr, "wrote offline fixtures to %s\n", cfg.OutDir)
	return nil
}

func parseArgs(args []string) (fixturesConfig, error) {
	fs := flag.NewFlagSet("mkfixtures", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	outDir := fs.String("out-dir", "proof/testdata", "output directory for offline fixtures")

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return fixturesConfig{}, err
		}
		return fixturesConfig{}, usageError{message: err.Error()}
	}
	if fs.NArg() != 0 {
		return fixturesConfig{}, usageError{message: fmt.Sprintf("mkfixtures does not accept positional arguments: %s", strings.Join(fs.Args(), " "))}
	}

	return fixturesConfig{
		OutDir: *outDir,
	}, nil
}

type usageError struct {
	message string
}

func (e usageError) Error() string {
	return e.message
}

func isUsageError(err error) bool {
	var usageErr usageError
	return errors.As(err, &usageErr)
}
