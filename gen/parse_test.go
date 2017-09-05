package gen

import "testing"

func TestTitleCases(t *testing.T) {
	titleCases := []struct {
		in       string
		expected string
	}{
		// simple cases
		{"t", "T"},
		{"baseCase", "BaseCase"},
		// check snaking coverage
		{"snake_case", "SnakeCase"},
		// check initialisms
		{"snake_json", "SnakeJSON"},
		{"json_snake", "JSONSnake"},
		{"s_json_snake", "SJSONSnake"},
		// initialisms strict on casing
		{"jSon_snake", "JSonSnake"},
		// key word edge cases
		{"Args", "Args_"},
		{"AResult", "AResult_"},
		// combo snake, initialism, key word
		{"json_args", "JSONArgs_"},
	}

	for _, tc := range titleCases {
		actual := titleCase(tc.in)
		if actual != tc.expected {
			t.Errorf("titleCase(%q) => %q, want %q", tc.in, actual, tc.expected)
		}
	}
}
