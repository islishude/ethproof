package proof

import "testing"

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
