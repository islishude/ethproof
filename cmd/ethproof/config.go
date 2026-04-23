package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
)

type cliConfig struct {
	Generate generateConfigSections `json:"generate"`
	Verify   verifyConfigSections   `json:"verify"`
}

type generateConfigSections struct {
	State   *generateStateConfigFile       `json:"state"`
	Receipt *generateReceiptConfigFile     `json:"receipt"`
	Tx      *generateTransactionConfigFile `json:"tx"`
}

type verifyConfigSections struct {
	State   *verifyStateConfigFile       `json:"state"`
	Receipt *verifyReceiptConfigFile     `json:"receipt"`
	Tx      *verifyTransactionConfigFile `json:"tx"`
}

type generateStateConfigFile struct {
	RPCs    []string `json:"rpcs"`
	MinRPCs *int     `json:"minRpcs"`
	Block   *uint64  `json:"block"`
	Account string   `json:"account"`
	Slots   []string `json:"slots"`
	Out     string   `json:"out"`
}

type generateReceiptConfigFile struct {
	RPCs     []string `json:"rpcs"`
	MinRPCs  *int     `json:"minRpcs"`
	Tx       string   `json:"tx"`
	LogIndex *uint    `json:"logIndex"`
	Out      string   `json:"out"`
}

type generateTransactionConfigFile struct {
	RPCs    []string `json:"rpcs"`
	MinRPCs *int     `json:"minRpcs"`
	Tx      string   `json:"tx"`
	Out     string   `json:"out"`
}

type verifyStateConfigFile struct {
	RPCs    []string `json:"rpcs"`
	MinRPCs *int     `json:"minRpcs"`
	Proof   string   `json:"proof"`
}

type verifyReceiptConfigFile struct {
	RPCs          []string `json:"rpcs"`
	MinRPCs       *int     `json:"minRpcs"`
	Proof         string   `json:"proof"`
	ExpectEmitter string   `json:"expectEmitter"`
	ExpectTopics  []string `json:"expectTopics"`
	ExpectData    string   `json:"expectData"`
}

type verifyTransactionConfigFile struct {
	RPCs    []string `json:"rpcs"`
	MinRPCs *int     `json:"minRpcs"`
	Proof   string   `json:"proof"`
}

func loadCLIConfig(path string) (*cliConfig, error) {
	if path == "" {
		return nil, nil
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config %s: %w", path, err)
	}
	var cfg cliConfig
	decoder := json.NewDecoder(bytes.NewReader(b))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&cfg); err != nil {
		return nil, fmt.Errorf("decode config %s: %w", path, err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		return nil, fmt.Errorf("decode config %s: unexpected trailing data", path)
	}
	return &cfg, nil
}
