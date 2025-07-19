package generics

import (
	"reflect"
	"strconv"
	"testing"
)

func TestMap(t *testing.T) {
	// Define the test cases using a table-driven approach
	testCases := []struct {
		name     string
		input    []int
		f        func(int) string
		expected []string
	}{
		{
			name:     "Test with integers to strings",
			input:    []int{1, 2, 3},
			f:        func(i int) string { return strconv.Itoa(i) },
			expected: []string{"1", "2", "3"},
		},
		{
			name:     "Test with an empty slice",
			input:    []int{},
			f:        func(i int) string { return strconv.Itoa(i) },
			expected: []string{},
		},
		{
			name:     "Test with a nil slice",
			input:    nil,
			f:        func(i int) string { return strconv.Itoa(i) },
			expected: []string{},
		},
	}

	// Iterate over the test cases
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Call the function to be tested
			result := Map(tc.input, tc.f)

			// Check if the result is as expected
			// For nil vs empty slice, reflect.DeepEqual is important
			if !reflect.DeepEqual(result, tc.expected) {
				t.Errorf("Map() = %v, want %v", result, tc.expected)
			}

			// Special check for nil input, which should produce a non-nil empty slice
			if tc.input == nil && result == nil {
				t.Errorf("Map(nil) should return an empty slice, not a nil slice")
			}
		})
	}
}

// Example with a different type to ensure generic capabilities
func TestMap_StringLength(t *testing.T) {
	input := []string{"a", "bb", "ccc"}
	expected := []int{1, 2, 3}

	result := Map(input, func(s string) int {
		return len(s)
	})

	if !reflect.DeepEqual(result, expected) {
		t.Errorf("Map() with strings = %v, want %v", result, expected)
	}
}
