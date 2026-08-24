package main

import (
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

func TestGenerateParsersRejectMalformedEthereumValues(t *testing.T) {
	tests := []struct {
		name string
		run  func() error
		want string
	}{
		{
			name: "invalid account",
			run: func() error {
				_, err := parseGenerateStateArgs([]string{"--rpc", "http://rpc", "--min-rpcs", "1", "--account", "not-hex", "--slot", "0x0"})
				return err
			},
			want: "20-byte",
		},
		{
			name: "invalid storage slot",
			run: func() error {
				_, err := parseGenerateStateArgs([]string{"--rpc", "http://rpc", "--min-rpcs", "1", "--account", "0x1111111111111111111111111111111111111111", "--slot", "0xzz"})
				return err
			},
			want: "not valid hex",
		},
		{
			name: "oversized storage slot",
			run: func() error {
				_, err := parseGenerateStateArgs([]string{"--rpc", "http://rpc", "--min-rpcs", "1", "--account", "0x1111111111111111111111111111111111111111", "--slot", "0x" + strings.Repeat("1", 65)})
				return err
			},
			want: "between 1 and 64",
		},
		{
			name: "invalid receipt tx hash",
			run: func() error {
				_, err := parseGenerateReceiptArgs([]string{"--rpc", "http://rpc", "--min-rpcs", "1", "--tx", "not-hex"})
				return err
			},
			want: "32-byte",
		},
		{
			name: "short transaction tx hash",
			run: func() error {
				_, err := parseGenerateTransactionArgs([]string{"--rpc", "http://rpc", "--min-rpcs", "1", "--tx", "0x01"})
				return err
			},
			want: "32-byte",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.run()
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestStorageSlotParserPreservesShortValidKeys(t *testing.T) {
	got, err := parseStorageSlotStrict("0xabc")
	if err != nil {
		t.Fatalf("parseStorageSlotStrict: %v", err)
	}
	if got != common.HexToHash("0x0abc") {
		t.Fatalf("unexpected slot: %s", got)
	}
}

func TestReceiptExpectationsRejectMalformedValues(t *testing.T) {
	tests := []struct {
		name    string
		emitter string
		data    string
		topics  []string
	}{
		{name: "emitter", emitter: "0x1234"},
		{name: "data", data: "0xabc"},
		{name: "topic", topics: []string{"0x01"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := buildReceiptExpectations(tt.emitter, tt.data, tt.topics); err == nil {
				t.Fatal("expected malformed expectation to fail")
			}
		})
	}
}
