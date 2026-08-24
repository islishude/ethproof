package proof

import (
	"errors"
	"strings"
	"testing"
)

func TestRedactRPCErrorRemovesURLCredentials(t *testing.T) {
	rawURL := "https://user:password@example.com/v3/path-key?api_key=query-secret#fragment-secret"
	cause := errors.New("request failed for " + rawURL + " password /v3/path-key api_key=query-secret fragment-secret")
	err := redactRPCError(cause, rawURL, "source[0]")
	if !errors.Is(err, cause) {
		t.Fatal("redacted error must preserve unwrapping")
	}
	if !strings.Contains(err.Error(), "source[0]") {
		t.Fatalf("redacted error lost safe source id: %v", err)
	}
	for _, forbidden := range []string{rawURL, "user", "password", "path-key", "query-secret", "fragment-secret"} {
		if strings.Contains(err.Error(), forbidden) {
			t.Fatalf("redacted error leaked %q: %v", forbidden, err)
		}
	}
}
