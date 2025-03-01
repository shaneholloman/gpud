package peermem

import (
	"regexp"
	"testing"
)

func TestRegexInvalidContext(t *testing.T) {
	t.Parallel()

	tests := []struct {
		log     string
		matches bool
	}{
		{"[Thu Sep 19 02:29:46 2024] nvidia-peermem nv_get_p2p_free_callback:127 ERROR detected invalid context, skipping further processing", true},
		{"ERROR detected invalid context, skipping further processing", true},
		{"[123213123123] ERROR detected invalid context, skipping further processing", true},
	}

	for _, test := range tests {
		matched := hasInvalidContext(test.log)
		if matched != test.matches {
			t.Errorf("Expected match: %v, got: %v for log: %s", test.matches, matched, test.log)
		}
	}
}

var (
	compiledInvalidContext = regexp.MustCompile(RegexInvalidContext)
)

func hasInvalidContext(line string) bool {
	if match := compiledInvalidContext.FindStringSubmatch(line); match != nil {
		return true
	}
	return false
}
