package commands

import (
	"testing"

	"leetcli/internal/api"
	"leetcli/internal/config"
)

func TestAutoDetectCategory(t *testing.T) {
	cfg := config.Default()
	ds := cfg.GetDataStructures()

	tests := []struct {
		name     string
		tags     []api.TopicTag
		expected string
	}{
		{
			name:     "Array tag",
			tags:     []api.TopicTag{{Name: "Array"}, {Name: "Hash Table"}},
			expected: "array",
		},
		{
			name:     "Dynamic Programming tag",
			tags:     []api.TopicTag{{Name: "Dynamic Programming"}},
			expected: "dp",
		},
		{
			name:     "Two Pointers tag",
			tags:     []api.TopicTag{{Name: "Two Pointers"}},
			expected: "two-pointer",
		},
		{
			name:     "Sliding Window tag",
			tags:     []api.TopicTag{{Name: "Sliding Window"}},
			expected: "sliding",
		},
		{
			name:     "Heap tag",
			tags:     []api.TopicTag{{Name: "Heap (Priority Queue)"}},
			expected: "heap",
		},
		{
			name:     "Binary Search tag",
			tags:     []api.TopicTag{{Name: "Binary Search"}},
			expected: "binary",
		},
		{
			name:     "Unknown tag",
			tags:     []api.TopicTag{{Name: "Brainteaser"}},
			expected: "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := autoDetectCategory(tc.tags, ds)
			if got != tc.expected {
				t.Errorf("autoDetectCategory() = %q, want %q", got, tc.expected)
			}
		})
	}
}
