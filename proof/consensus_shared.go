package proof

import "fmt"

func requireMatchingSnapshots[T any](rpcs []string, snapshots []T, compare func(base, other T) []string) (T, error) {
	var zero T
	if len(snapshots) == 0 {
		return zero, fmt.Errorf("no snapshots returned")
	}
	if len(rpcs) != len(snapshots) {
		return zero, fmt.Errorf("expected %d snapshots, got %d", len(rpcs), len(snapshots))
	}

	// Compare every snapshot to the first source's normalized view so mismatch errors can point
	// at a concrete pair of RPC endpoints.
	base := snapshots[0]
	for i := 1; i < len(snapshots); i++ {
		if err := combineMismatch(rpcs[0], rpcs[i], compare(base, snapshots[i])); err != nil {
			return zero, err
		}
	}
	return base, nil
}
