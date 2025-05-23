package os

import "testing"

func TestHasVFSFileMaxLimitReached(t *testing.T) {
	tests := []struct {
		name string
		line string
		want bool
	}{
		{
			name: "exact match",
			line: "VFS: file-max limit 1000000 reached",
			want: true,
		},
		{
			name: "with timestamp prefix",
			line: "[Sun Dec  1 14:54:40 2024] VFS: file-max limit 1000000 reached",
			want: true,
		},
		{
			name: "with different number",
			line: "VFS: file-max limit 500000 reached",
			want: true,
		},
		{
			name: "with facility and level",
			line: "kern  :warn  : VFS: file-max limit 1000000 reached",
			want: true,
		},
		{
			name: "with ISO timestamp",
			line: "kern  :warn  : 2025-01-21T04:41:44,285060+00:00 VFS: file-max limit 1000000 reached",
			want: true,
		},
		{
			name: "some other log message",
			line: "some other log message",
			want: false,
		},
		{
			name: "empty string",
			line: "",
			want: false,
		},
		{
			name: "partial match - missing reached",
			line: "VFS: file-max limit 1000000",
			want: false,
		},
		{
			name: "partial match - missing number",
			line: "VFS: file-max limit reached",
			want: false,
		},
		{
			name: "case mismatch",
			line: "vfs: File-max Limit 1000000 Reached",
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := HasVFSFileMaxLimitReached(tt.line); got != tt.want {
				t.Errorf("HasVFSFileMaxLimitReached(%q) = %v, want %v", tt.line, got, tt.want)
			}
		})
	}
}

func TestMatch(t *testing.T) {
	tests := []struct {
		name        string
		line        string
		wantName    string
		wantMessage string
	}{
		{
			name:        "VFS file-max limit reached",
			line:        "VFS: file-max limit 1000000 reached",
			wantName:    eventVFSFileMaxLimitReached,
			wantMessage: messageVFSFileMaxLimitReached,
		},
		{
			name:        "VFS file-max with timestamp",
			line:        "[Sun Dec  1 14:54:40 2024] VFS: file-max limit 1000000 reached",
			wantName:    eventVFSFileMaxLimitReached,
			wantMessage: messageVFSFileMaxLimitReached,
		},
		{
			name:        "VFS file-max with ISO timestamp and facility",
			line:        "kern  :warn  : 2025-01-21T04:41:44,285060+00:00 VFS: file-max limit 1000000 reached",
			wantName:    eventVFSFileMaxLimitReached,
			wantMessage: messageVFSFileMaxLimitReached,
		},
		{
			name:        "no match",
			line:        "some random log message",
			wantName:    "",
			wantMessage: "",
		},
		{
			name:        "empty string",
			line:        "",
			wantName:    "",
			wantMessage: "",
		},
		{
			name:        "partial match",
			line:        "VFS: file-max limit 1000000",
			wantName:    "",
			wantMessage: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotName, gotMessage := Match(tt.line)
			if gotName != tt.wantName {
				t.Errorf("Match() name = %v, want %v", gotName, tt.wantName)
			}
			if gotMessage != tt.wantMessage {
				t.Errorf("Match() message = %v, want %v", gotMessage, tt.wantMessage)
			}
		})
	}
}

func TestGetMatches(t *testing.T) {
	matches := getMatches()

	// Verify we have the expected number of matchers
	if len(matches) != 1 {
		t.Errorf("getMatches() returned %d matches, want 1", len(matches))
	}

	// Verify the VFS file-max matcher
	vfsMatch := matches[0]
	if vfsMatch.eventName != eventVFSFileMaxLimitReached {
		t.Errorf("first match name = %v, want %v", vfsMatch.eventName, eventVFSFileMaxLimitReached)
	}
	if vfsMatch.regex != regexVFSFileMaxLimitReached {
		t.Errorf("first match regex = %v, want %v", vfsMatch.regex, regexVFSFileMaxLimitReached)
	}
	if vfsMatch.message != messageVFSFileMaxLimitReached {
		t.Errorf("first match message = %v, want %v", vfsMatch.message, messageVFSFileMaxLimitReached)
	}

	// Test the check function
	if !vfsMatch.check("VFS: file-max limit 1000000 reached") {
		t.Error("check function failed to match valid input")
	}
	if vfsMatch.check("invalid input") {
		t.Error("check function matched invalid input")
	}
}

// TestMatch_ComprehensiveInvalidInputs tests the Match function with various invalid inputs
func TestMatch_ComprehensiveInvalidInputs(t *testing.T) {
	invalidInputs := []string{
		"",
		" ",
		"VFS: file-max limit",
		"file-max limit reached",
		"VFS file-max limit reached",          // Missing colon
		"vfs: file-max limit 1000000 reached", // Lowercase VFS
	}

	for _, input := range invalidInputs {
		t.Run("Invalid: "+input, func(t *testing.T) {
			name, message := Match(input)
			if name != "" || message != "" {
				t.Errorf("Match(%q) = (%q, %q), want empty strings", input, name, message)
			}
		})
	}
}

// TestMatch_WithMultipleMatchers tests the behavior if we were to have multiple matchers
func TestMatch_WithMultipleMatchers(t *testing.T) {
	// This test verifies that the first match is returned
	matches := getMatches()
	if len(matches) == 0 {
		t.Skip("No matches defined, skipping test")
	}

	// Test that the first match takes precedence
	match := matches[0]
	testInput := "VFS: file-max limit 1000000 reached"

	name, message := Match(testInput)
	if name != match.eventName {
		t.Errorf("Match(%q) name = %q, want %q", testInput, name, match.eventName)
	}
	if message != match.message {
		t.Errorf("Match(%q) message = %q, want %q", testInput, message, match.message)
	}
}
