package proof

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
)

func TestNormalizeSourceNames(t *testing.T) {
	tests := []struct {
		name    string
		sources []stubNamedSource
		min     int
		want    string
	}{
		{
			name:    "not enough sources",
			sources: []stubNamedSource{"only-one"},
			min:     2,
			want:    "need at least 2 distinct rpc sources",
		},
		{
			name:    "duplicate names",
			sources: []stubNamedSource{"one", "one"},
			min:     1,
			want:    "duplicates source",
		},
		{
			name:    "empty name",
			sources: []stubNamedSource{" "},
			min:     1,
			want:    "empty name",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := normalizeSourceNames(tt.sources, tt.min)
			if err == nil {
				t.Fatal("expected normalizeSourceNames to fail")
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestNormalizeSourceNamesRejectsTypedNilSource(t *testing.T) {
	var source *fakeHeaderSource
	_, err := normalizeSourceNames([]HeaderSource{source}, 1)
	if err == nil || !strings.Contains(err.Error(), "source 0 is nil") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestNormalizeSourceNamesDoesNotExposeNames(t *testing.T) {
	secret := "https://user:password@example.com/v3/path-key?api_key=query-secret"
	ids, err := normalizeSourceNames([]stubNamedSource{stubNamedSource(secret), "safe-second-source"}, 2)
	if err != nil {
		t.Fatalf("normalizeSourceNames: %v", err)
	}
	if !slices.Equal(ids, []string{"source[0]", "source[1]"}) {
		t.Fatalf("unexpected source ids: %v", ids)
	}
	_, err = normalizeSourceNames([]stubNamedSource{stubNamedSource(secret), stubNamedSource(secret)}, 1)
	if err == nil {
		t.Fatal("expected duplicate source names to fail")
	}
	for _, forbidden := range []string{secret, "password", "path-key", "query-secret"} {
		if strings.Contains(err.Error(), forbidden) {
			t.Fatalf("error leaked %q: %v", forbidden, err)
		}
	}
}

func TestValidateSourceConsensusMetadata(t *testing.T) {
	valid := SourceConsensus{
		Mode:    "live-rpc",
		RPCs:    []string{"source[0]", "source[1]"},
		Digests: []ConsensusDigest{{Name: "header"}},
		Fields:  []ConsensusField{{Name: "blockHash", Consistent: true}},
	}
	if err := validateSourceConsensusMetadata(SourceConsensus{}); err != nil {
		t.Fatalf("zero-value metadata should remain compatible: %v", err)
	}
	if err := validateSourceConsensusMetadata(valid); err != nil {
		t.Fatalf("valid metadata rejected: %v", err)
	}

	tests := []struct {
		name string
		edit func(*SourceConsensus)
	}{
		{name: "missing mode", edit: func(value *SourceConsensus) { value.Mode = "" }},
		{name: "live mode without sources", edit: func(value *SourceConsensus) { value.RPCs = nil }},
		{name: "blank source id", edit: func(value *SourceConsensus) { value.RPCs[0] = " " }},
		{name: "duplicate source id", edit: func(value *SourceConsensus) { value.RPCs[1] = value.RPCs[0] }},
		{name: "blank digest name", edit: func(value *SourceConsensus) { value.Digests[0].Name = "" }},
		{name: "inconsistent field", edit: func(value *SourceConsensus) { value.Fields[0].Consistent = false }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			value := sourceConsensus(valid.Mode, valid.RPCs, valid.Digests, valid.Fields)
			tt.edit(&value)
			if err := validateSourceConsensusMetadata(value); err == nil {
				t.Fatal("expected invalid metadata to fail")
			}
		})
	}
}

func TestVerifierRejectsMalformedSourceConsensusMetadata(t *testing.T) {
	pkg := mustLoadTransactionFixture(t)
	pkg.Block.SourceConsensus.Fields[0].Consistent = false
	if err := VerifyTransactionProofPackageAgainstEmbeddedRoots(&pkg); err == nil {
		t.Fatal("expected malformed source consensus metadata to fail")
	}
}

func TestRequireMatchingSnapshotsRejectsMismatch(t *testing.T) {
	base := testTransactionSnapshot()
	other := cloneTransactionSnapshot(base)
	other.TxHash = common.HexToHash("0x7777")

	_, err := requireMatchingSnapshots([]string{"rpc-a", "rpc-b"}, []*transactionSnapshot{base, other}, compareTransactionSnapshot)
	if err == nil {
		t.Fatal("expected mismatch to fail")
	}
	if !strings.Contains(err.Error(), "normalized data mismatch") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCollectFromSourcesWrapsSourceErrors(t *testing.T) {
	_, err := collectFromSources(context.Background(), []stubNamedSource{"source-a"}, func(_ context.Context, _ stubNamedSource) (int, error) {
		return 0, errors.New("boom")
	})
	if err == nil {
		t.Fatal("expected source error")
	}
	if !strings.Contains(err.Error(), "source[0]: boom") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCollectFromSourcesPreservesInputOrder(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	bDone := make(chan struct{})
	cDone := make(chan struct{})
	got, err := collectFromSources(ctx, []stubNamedSource{"source-a", "source-b", "source-c"}, func(ctx context.Context, source stubNamedSource) (int, error) {
		switch source {
		case "source-a":
			select {
			case <-bDone:
				return 1, nil
			case <-ctx.Done():
				return 0, ctx.Err()
			}
		case "source-b":
			select {
			case <-cDone:
				close(bDone)
				return 2, nil
			case <-ctx.Done():
				return 0, ctx.Err()
			}
		case "source-c":
			close(cDone)
			return 3, nil
		default:
			return 0, errors.New("unexpected source")
		}
	})
	if err != nil {
		t.Fatalf("collectFromSources: %v", err)
	}
	if !slices.Equal(got, []int{1, 2, 3}) {
		t.Fatalf("unexpected result order: got %v want %v", got, []int{1, 2, 3})
	}
}

func TestCollectFromSourcesCancelsOnFirstError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	canceled := make(chan struct{})
	_, err := collectFromSources(ctx, []stubNamedSource{"source-a", "source-b"}, func(ctx context.Context, source stubNamedSource) (int, error) {
		switch source {
		case "source-a":
			return 0, errors.New("boom")
		case "source-b":
			<-ctx.Done()
			close(canceled)
			return 0, ctx.Err()
		default:
			return 0, errors.New("unexpected source")
		}
	})
	if err == nil {
		t.Fatal("expected source error")
	}
	if !strings.Contains(err.Error(), "source[0]: boom") {
		t.Fatalf("unexpected error: %v", err)
	}

	select {
	case <-canceled:
	case <-time.After(time.Second):
		t.Fatal("expected sibling fetch to observe cancellation")
	}
}

func TestCollectFromSourcesReturnsParentContextError(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := collectFromSources(ctx, []stubNamedSource{"source-a"}, func(ctx context.Context, _ stubNamedSource) (int, error) {
		<-ctx.Done()
		return 0, ctx.Err()
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context cancellation, got %v", err)
	}
	if strings.Contains(err.Error(), "source[0]:") {
		t.Fatalf("expected parent context error without source prefix, got %v", err)
	}
}
