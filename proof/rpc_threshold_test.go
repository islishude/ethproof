package proof

import (
	"context"
	"slices"
	"testing"
)

type stubNamedSource string

func (s stubNamedSource) SourceName() string {
	return string(s)
}

func TestNormalizeRPCURLsDefaultMinimum(t *testing.T) {
	if _, err := normalizeRPCURLs([]string{"http://one"}, 0); err == nil {
		t.Fatal("expected default minimum of 3 rpc sources to reject a single url")
	}
}

func TestNormalizeRPCURLsCustomMinimum(t *testing.T) {
	got, err := normalizeRPCURLs([]string{
		"http://one",
		"http://one",
		" http://two ",
	}, 1)
	if err != nil {
		t.Fatalf("normalizeRPCURLs returned error: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected duplicate urls to be deduplicated, got %d entries", len(got))
	}
	if got[0] != "http://one" || got[1] != "http://two" {
		t.Fatalf("unexpected normalized urls: %#v", got)
	}
}

func TestNormalizeRPCURLsRejectsInvalidMinimum(t *testing.T) {
	if _, err := normalizeRPCURLs([]string{"http://one"}, -1); err == nil {
		t.Fatal("expected invalid minimum to fail")
	}
}

func TestOpenNormalizedRPCSourcesUsingNormalizesURLsAndClosesSources(t *testing.T) {
	openCalls := 0
	closeCalls := 0

	sourceSet, err := openNormalizedRPCSourcesUsing(
		context.Background(),
		[]string{" http://one ", "http://one", "http://two "},
		1,
		func(_ context.Context, urls []string) ([]*rpcSource, error) {
			openCalls++
			if !slices.Equal(urls, []string{"http://one", "http://two"}) {
				t.Fatalf("unexpected normalized urls: %v", urls)
			}
			return []*rpcSource{
				{url: "http://one"},
				{url: "http://two"},
			}, nil
		},
		func(sources []*rpcSource) {
			closeCalls++
			if len(sources) != 2 {
				t.Fatalf("unexpected source count passed to closer: %d", len(sources))
			}
		},
	)
	if err != nil {
		t.Fatalf("openNormalizedRPCSourcesUsing returned error: %v", err)
	}

	headerSources := sourceSet.HeaderSources()
	got := make([]string, len(headerSources))
	for i, source := range headerSources {
		got[i] = source.SourceName()
	}
	if !slices.Equal(got, []string{"http://one", "http://two"}) {
		t.Fatalf("unexpected source names: %v", got)
	}

	sourceSet.Close()
	sourceSet.Close()

	if openCalls != 1 {
		t.Fatalf("expected opener to be called once, got %d", openCalls)
	}
	if closeCalls != 1 {
		t.Fatalf("expected closer to be called once, got %d", closeCalls)
	}
}

func TestOpenNormalizedRPCSourcesUsingSkipsOpenOnNormalizationError(t *testing.T) {
	openCalls := 0

	_, err := openNormalizedRPCSourcesUsing(
		context.Background(),
		[]string{"http://one"},
		-1,
		func(_ context.Context, urls []string) ([]*rpcSource, error) {
			openCalls++
			return nil, nil
		},
		func([]*rpcSource) {},
	)
	if err == nil {
		t.Fatal("expected normalization error")
	}
	if openCalls != 0 {
		t.Fatalf("expected opener to be skipped, got %d calls", openCalls)
	}
}
