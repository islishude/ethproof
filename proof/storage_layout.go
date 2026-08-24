package proof

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// StorageLayout mirrors Solidity's storageLayout compiler output.
type StorageLayout struct {
	Storage []StorageLayoutEntry         `json:"storage"`
	Types   map[string]StorageLayoutType `json:"types"`
}

// StorageLayoutEntry describes one storage variable or struct member.
type StorageLayoutEntry struct {
	ASTID    int    `json:"astId"`
	Contract string `json:"contract"`
	Label    string `json:"label"`
	Offset   uint64 `json:"offset"`
	Slot     string `json:"slot"`
	Type     string `json:"type"`
}

// StorageLayoutType describes one Solidity storage type entry.
type StorageLayoutType struct {
	Encoding      string               `json:"encoding"`
	Label         string               `json:"label"`
	NumberOfBytes string               `json:"numberOfBytes"`
	Key           string               `json:"key,omitempty"`
	Value         string               `json:"value,omitempty"`
	Base          string               `json:"base,omitempty"`
	Members       []StorageLayoutEntry `json:"members,omitempty"`
}

// StorageLayoutFormat controls how compiler output is interpreted.
type StorageLayoutFormat string

const (
	StorageLayoutFormatAuto      StorageLayoutFormat = "auto"
	StorageLayoutFormatLayout    StorageLayoutFormat = "layout"
	StorageLayoutFormatArtifact  StorageLayoutFormat = "artifact"
	StorageLayoutFormatBuildInfo StorageLayoutFormat = "build-info"
)

// ParseStorageLayoutFormat validates a user-facing compiler-output format flag.
func ParseStorageLayoutFormat(raw string) (StorageLayoutFormat, error) {
	switch StorageLayoutFormat(strings.TrimSpace(raw)) {
	case "", StorageLayoutFormatAuto:
		return StorageLayoutFormatAuto, nil
	case StorageLayoutFormatLayout:
		return StorageLayoutFormatLayout, nil
	case StorageLayoutFormatArtifact:
		return StorageLayoutFormatArtifact, nil
	case StorageLayoutFormatBuildInfo:
		return StorageLayoutFormatBuildInfo, nil
	default:
		return "", fmt.Errorf("unsupported storage layout format %q", raw)
	}
}

// LoadStorageLayout reads compiler output from path and extracts Solidity storageLayout data.
func LoadStorageLayout(path string, contract string, format StorageLayoutFormat) (*StorageLayout, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read compiler output %s: %w", path, err)
	}
	layout, err := ParseStorageLayoutJSON(b, contract, format)
	if err != nil {
		return nil, err
	}
	return layout, nil
}

// ParseStorageLayoutJSON extracts Solidity storageLayout data from raw layout JSON,
// a Foundry artifact, or a Hardhat build-info file.
func ParseStorageLayoutJSON(b []byte, contract string, format StorageLayoutFormat) (*StorageLayout, error) {
	if format == "" {
		format = StorageLayoutFormatAuto
	}
	switch format {
	case StorageLayoutFormatAuto:
		detected, err := detectStorageLayoutFormat(b)
		if err != nil {
			return nil, err
		}
		return ParseStorageLayoutJSON(b, contract, detected)
	case StorageLayoutFormatLayout:
		if strings.TrimSpace(contract) != "" {
			return nil, fmt.Errorf("contract selector is not supported for raw storage layout input")
		}
		return parseRawStorageLayoutJSON(b)
	case StorageLayoutFormatArtifact:
		return parseArtifactStorageLayoutJSON(b, contract)
	case StorageLayoutFormatBuildInfo:
		return parseBuildInfoStorageLayoutJSON(b, contract)
	default:
		return nil, fmt.Errorf("unsupported storage layout format %q", format)
	}
}

func detectStorageLayoutFormat(b []byte) (StorageLayoutFormat, error) {
	var root map[string]json.RawMessage
	if err := json.Unmarshal(b, &root); err != nil {
		return "", fmt.Errorf("decode compiler output: %w", err)
	}
	if _, ok := root["storage"]; ok {
		if _, ok := root["types"]; ok {
			return StorageLayoutFormatLayout, nil
		}
	}
	if _, ok := root["storageLayout"]; ok {
		return StorageLayoutFormatArtifact, nil
	}
	if _, ok := root["output"]; ok {
		return StorageLayoutFormatBuildInfo, nil
	}
	return "", fmt.Errorf("unsupported compiler output shape")
}

func parseRawStorageLayoutJSON(b []byte) (*StorageLayout, error) {
	var layout StorageLayout
	if err := json.Unmarshal(b, &layout); err != nil {
		return nil, fmt.Errorf("decode raw storage layout: %w", err)
	}
	return validateStorageLayout(&layout, "raw storage layout")
}

func parseArtifactStorageLayoutJSON(b []byte, contract string) (*StorageLayout, error) {
	var artifact struct {
		StorageLayout *StorageLayout  `json:"storageLayout"`
		Metadata      json.RawMessage `json:"metadata"`
	}
	if err := json.Unmarshal(b, &artifact); err != nil {
		return nil, fmt.Errorf("decode artifact: %w", err)
	}
	if artifact.StorageLayout == nil {
		return nil, fmt.Errorf("storageLayout not found in artifact")
	}
	if strings.TrimSpace(contract) == "" {
		return nil, fmt.Errorf("contract is required for artifact storage layout lookup")
	}
	targets, err := artifactCompilationTargets(artifact.Metadata)
	if err != nil {
		return nil, err
	}
	if err := validateCompilationTargetSelector(targets, contract, "artifact"); err != nil {
		return nil, err
	}
	return validateStorageLayout(artifact.StorageLayout, "artifact storage layout")
}

func artifactCompilationTargets(raw json.RawMessage) (map[string]string, error) {
	metadataJSON := bytes.TrimSpace(raw)
	if len(metadataJSON) == 0 || bytes.Equal(metadataJSON, []byte("null")) {
		return nil, fmt.Errorf("artifact metadata.settings.compilationTarget is required to validate contract selector")
	}
	if len(metadataJSON) > 0 && metadataJSON[0] == '"' {
		var encoded string
		if err := json.Unmarshal(metadataJSON, &encoded); err != nil {
			return nil, fmt.Errorf("decode artifact metadata string: %w", err)
		}
		metadataJSON = []byte(encoded)
	}
	var metadata struct {
		Settings struct {
			CompilationTarget map[string]string `json:"compilationTarget"`
		} `json:"settings"`
	}
	if err := json.Unmarshal(metadataJSON, &metadata); err != nil {
		return nil, fmt.Errorf("decode artifact metadata: %w", err)
	}
	if len(metadata.Settings.CompilationTarget) == 0 {
		return nil, fmt.Errorf("artifact metadata.settings.compilationTarget is required to validate contract selector")
	}
	return metadata.Settings.CompilationTarget, nil
}

func validateCompilationTargetSelector(targets map[string]string, selector string, source string) error {
	selector = strings.TrimSpace(selector)
	if sourceName, contractName, ok := splitBuildInfoSelector(selector); ok {
		if targetContract, found := targets[sourceName]; !found || targetContract != contractName {
			return fmt.Errorf("contract %q does not match %s compilation target", selector, source)
		}
		return nil
	}

	matches := 0
	for _, contractName := range targets {
		if contractName == selector {
			matches++
		}
	}
	switch matches {
	case 0:
		return fmt.Errorf("contract %q does not match %s compilation target", selector, source)
	case 1:
		return nil
	default:
		return fmt.Errorf("contract %q is ambiguous in %s compilation targets; use source:contract", selector, source)
	}
}

func parseBuildInfoStorageLayoutJSON(b []byte, contract string) (*StorageLayout, error) {
	if strings.TrimSpace(contract) == "" {
		return nil, fmt.Errorf("contract is required for build-info storage layout lookup")
	}

	var buildInfo struct {
		Output struct {
			Contracts map[string]map[string]struct {
				StorageLayout *StorageLayout `json:"storageLayout"`
			} `json:"contracts"`
		} `json:"output"`
	}
	if err := json.Unmarshal(b, &buildInfo); err != nil {
		return nil, fmt.Errorf("decode build-info: %w", err)
	}
	layout, err := findBuildInfoStorageLayout(buildInfo.Output.Contracts, contract)
	if err != nil {
		return nil, err
	}
	return validateStorageLayout(layout, "build-info storage layout")
}

func findBuildInfoStorageLayout(contracts map[string]map[string]struct {
	StorageLayout *StorageLayout `json:"storageLayout"`
}, selector string) (*StorageLayout, error) {
	if sourceName, contractName, ok := splitBuildInfoSelector(selector); ok {
		sourceContracts, ok := contracts[sourceName]
		if !ok {
			return nil, fmt.Errorf("contract source %q not found in build-info", sourceName)
		}
		contractOutput, ok := sourceContracts[contractName]
		if !ok {
			return nil, fmt.Errorf("contract %q not found in build-info source %q", contractName, sourceName)
		}
		if contractOutput.StorageLayout == nil {
			return nil, fmt.Errorf("storageLayout not found for contract %s", selector)
		}
		return contractOutput.StorageLayout, nil
	}

	var matches []*StorageLayout
	for _, sourceContracts := range contracts {
		if contractOutput, ok := sourceContracts[selector]; ok {
			if contractOutput.StorageLayout == nil {
				return nil, fmt.Errorf("storageLayout not found for contract %s", selector)
			}
			matches = append(matches, contractOutput.StorageLayout)
		}
	}
	switch len(matches) {
	case 0:
		return nil, fmt.Errorf("contract %q not found in build-info", selector)
	case 1:
		return matches[0], nil
	default:
		return nil, fmt.Errorf("contract %q is ambiguous in build-info; use source:contract", selector)
	}
}

func splitBuildInfoSelector(selector string) (string, string, bool) {
	idx := strings.LastIndex(selector, ":")
	if idx <= 0 || idx == len(selector)-1 {
		return "", "", false
	}
	return selector[:idx], selector[idx+1:], true
}

func validateStorageLayout(layout *StorageLayout, source string) (*StorageLayout, error) {
	if layout == nil {
		return nil, fmt.Errorf("%s is empty", source)
	}
	if layout.Types == nil {
		return nil, fmt.Errorf("%s is missing types", source)
	}
	if layout.Storage == nil {
		layout.Storage = []StorageLayoutEntry{}
	}
	return layout, nil
}
