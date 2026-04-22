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
			want:    "duplicate source name",
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
	if !strings.Contains(err.Error(), "source-a: boom") {
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
	if !strings.Contains(err.Error(), "source-a: boom") {
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
	if strings.Contains(err.Error(), "source-a:") {
		t.Fatalf("expected parent context error without source prefix, got %v", err)
	}
}
