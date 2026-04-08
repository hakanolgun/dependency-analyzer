package engine

import "testing"

func TestNormalizeRepoURL(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"https://github.com/facebook/react-native.git", "https://github.com/facebook/react-native"},
		{"git+https://github.com/facebook/react-native.git", "https://github.com/facebook/react-native"},
		{"git@github.com:facebook/react-native.git", "https://github.com/facebook/react-native"},
		{"git@github.com:facebook/react-native", "https://github.com/facebook/react-native"},
		{"ssh://git@github.com/facebook/react-native.git", "https://github.com/facebook/react-native"},
		{"git://github.com/facebook/react-native.git", "https://github.com/facebook/react-native"},
		{"", ""},
	}

	for _, tt := range tests {
		got := NormalizeRepoURL(tt.input)
		if got != tt.expected {
			t.Errorf("NormalizeRepoURL(%q) = %q; want %q", tt.input, got, tt.expected)
		}
	}
}
