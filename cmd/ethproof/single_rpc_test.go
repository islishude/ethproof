package main

import (
	"strings"
	"testing"
)

func TestSingleRPCInputs(t *testing.T) {
	parsers := []struct {
		name  string
		parse func([]string) error
		args  []string
	}{
		{"generate state", func(args []string) error { _, err := parseGenerateStateArgs(args); return err }, []string{"--account", "0x1111111111111111111111111111111111111111", "--slot", "0x01"}},
		{"generate receipt", func(args []string) error { _, err := parseGenerateReceiptArgs(args); return err }, []string{"--tx", testHash02}},
		{"generate tx", func(args []string) error { _, err := parseGenerateTransactionArgs(args); return err }, []string{"--tx", testHash02}},
		{"verify state", func(args []string) error { _, err := parseVerifyStateArgs(args); return err }, nil},
		{"verify receipt", func(args []string) error { _, err := parseVerifyReceiptArgs(args); return err }, nil},
		{"verify tx", func(args []string) error { _, err := parseVerifyTransactionArgs(args); return err }, nil},
	}
	for _, parser := range parsers {
		t.Run(parser.name, func(t *testing.T) {
			for _, tt := range []struct {
				name string
				args []string
				want string
			}{
				{"single", []string{"--rpc", "http://localhost:8545"}, ""},
				{"missing", nil, "requires"},
				{"blank", []string{"--rpc", " \t"}, "requires"},
				{"empty", []string{"--rpc="}, "requires"},
				{"repeated", []string{"--rpc", "http://one", "--rpc=http://two"}, "only be supplied once"},
				{"repeated identical", []string{"--rpc=http://one", "--rpc=http://one"}, "only be supplied once"},
				{"removed threshold", []string{"--rpc", "http://one", "--min-rpcs", "1"}, "flag provided but not defined"},
			} {
				t.Run(tt.name, func(t *testing.T) {
					err := parser.parse(append(append([]string{}, parser.args...), tt.args...))
					if tt.want == "" {
						if err != nil {
							t.Fatal(err)
						}
						return
					}
					if err == nil || !strings.Contains(err.Error(), tt.want) {
						t.Fatalf("want %q, got %v", tt.want, err)
					}
				})
			}
			parts := strings.Split(parser.name, " ")
			for _, field := range []string{`"rpcs":["http://old"]`, `"rpc":"http://one","minRpcs":1`} {
				path := writeTestConfig(t, `{"`+parts[0]+`":{"`+parts[1]+`":{`+field+`}}}`)
				if err := parser.parse(append(append([]string{}, parser.args...), "--config", path)); err == nil || !strings.Contains(err.Error(), "unknown field") {
					t.Fatalf("legacy config accepted: %v", err)
				}
			}
			path := writeTestConfig(t, `{"`+parts[0]+`":{"`+parts[1]+`":{"rpc":"http://configured"}}}`)
			args := append(append([]string{}, parser.args...), "--config", path)
			if err := parser.parse(args); err != nil {
				t.Fatal(err)
			}
			if err := parser.parse(append(args, "--rpc", " ")); err == nil || !strings.Contains(err.Error(), "requires") {
				t.Fatalf("blank flag must override config and fail: %v", err)
			}
		})
	}
}

func TestVerifyReceiptDoesNotReuseGenerateRPC(t *testing.T) {
	path := writeTestConfig(t, `{"generate":{"receipt":{"rpc":"http://generate"}}}`)
	if _, err := parseVerifyReceiptArgs([]string{"--config", path}); err == nil || !strings.Contains(err.Error(), "requires a separately configured RPC") {
		t.Fatalf("reused generation endpoint: %v", err)
	}
	path = writeTestConfig(t, `{"generate":{"receipt":{"rpc":"http://same"}},"verify":{"receipt":{"rpc":"http://same"}}}`)
	cfg, err := parseVerifyReceiptArgs([]string{"--config", path})
	if err != nil || cfg.RPCURL != "http://same" {
		t.Fatalf("explicitly shared endpoint rejected: %+v, %v", cfg, err)
	}
}

func TestRepeatedRPCDoesNotLeakCredentials(t *testing.T) {
	secretURL := "https://user:password@example.com/path-key?api_key=query-secret"
	_, err := parseVerifyTransactionArgs([]string{"--rpc", "http://first", "--rpc", secretURL})
	if err == nil || !strings.Contains(err.Error(), "only be supplied once") {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, secret := range []string{secretURL, "user", "password", "path-key", "query-secret"} {
		if strings.Contains(err.Error(), secret) {
			t.Fatalf("duplicate RPC error leaked %q", secret)
		}
	}
}
