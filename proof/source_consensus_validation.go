package proof

import (
	"fmt"
	"strings"
)

func validateSourceConsensusMetadata(consensus SourceConsensus) error {
	if isEmptySourceConsensus(consensus) {
		return nil
	}
	if strings.TrimSpace(consensus.Mode) == "" {
		return fmt.Errorf("source consensus mode is required when metadata is present")
	}
	if consensus.Mode == "live-rpc" && len(consensus.RPCs) == 0 {
		return fmt.Errorf("live source consensus requires at least one source id")
	}
	if err := validateUniqueMetadataNames("source consensus rpcs", consensus.RPCs); err != nil {
		return err
	}

	digestNames := make([]string, len(consensus.Digests))
	for i, digest := range consensus.Digests {
		digestNames[i] = digest.Name
	}
	if err := validateUniqueMetadataNames("source consensus digests", digestNames); err != nil {
		return err
	}

	fieldNames := make([]string, len(consensus.Fields))
	for i, field := range consensus.Fields {
		fieldNames[i] = field.Name
		if !field.Consistent {
			return fmt.Errorf("source consensus field %d is not marked consistent", i)
		}
	}
	return validateUniqueMetadataNames("source consensus fields", fieldNames)
}

func isEmptySourceConsensus(consensus SourceConsensus) bool {
	return consensus.Mode == "" && len(consensus.RPCs) == 0 && len(consensus.Digests) == 0 && len(consensus.Fields) == 0
}

func validateUniqueMetadataNames(group string, names []string) error {
	seen := make(map[string]int, len(names))
	for i, rawName := range names {
		name := strings.TrimSpace(rawName)
		if name == "" {
			return fmt.Errorf("%s entry %d is empty", group, i)
		}
		if previous, ok := seen[name]; ok {
			return fmt.Errorf("%s entries %d and %d are duplicates", group, previous, i)
		}
		seen[name] = i
	}
	return nil
}
