package main

import (
	"fmt"
	"testing"
)

func TestExpandCharRanges(t *testing.T) {

	tests := []struct {
		input, output string
	}{
		{"", ""},
		{"abc", "abc"},
		{"[a-c]", "abc"},
		{"x[a-c]y", "xabcy"},
		{"[a-c", "[a-c"},
		{"[a+c]", "[a+c]"},
		{"_-[a-f]", "_-abcdef"},
		{"_-[f-a]", "_-abcdef"},
		{"_-[a-a]", "_-a"},
		{"][", "]["},
		{"[a-", "[a-"},
		{"[a-(", "[a-("},
		{"[a-c][d-f]", "abcdef"},
	}
	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {

			r := expandCharRanges(tc.input)
			if r != tc.output {
				t.Fatalf("expected '%s' but got '%s'", tc.output, r)
			}
		})
	}
}

func TestMatching(t *testing.T) {
	s := " /path/to/file/file.c "

	if matching(s, 5, "/[a-z].") != "/path/to/file/file.c" {
		t.Fatalf("arrr")
	}
	tests := []struct {
		pos                  int
		input, chars, output string
	}{
		{0, "file.c", "/[a-z].", "file.c"},
		{5, "file.c", "/[a-z].", "file.c"},
		{6, "file.c", "/[a-z].", ""},
		{0, " /path/to/file/file.c ", "/[a-z].", ""},
		{5, " /path/to/file/file.c ", "/[a-z].", "/path/to/file/file.c"},
		{5, ":/path/to/file/file.c,", "/[a-z].", "/path/to/file/file.c"},
		{5, "asd /path/to/file/file.c:20:50", "/[a-z][A-Z].:[0-9]", "/path/to/file/file.c:20:50"},
		{5, "asd /path/to/file/file.c:20:50,", "/[a-z][A-Z].:[0-9]", "/path/to/file/file.c:20:50"},
	}
	for _, tc := range tests {
		t.Run(fmt.Sprintf("%s[%d],%s", tc.input, tc.pos, tc.chars), func(t *testing.T) {

			r := matching(tc.input, tc.pos, tc.chars)
			if r != tc.output {
				t.Fatalf("expected '%s' but got '%s'", tc.output, r)
			}
		})
	}
}
