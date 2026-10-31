package regexp2

import (
	"reflect"
	"testing"
)

// Prefilters only skip start positions, so a pattern must find the same
// captures as a copy whose never-matching extra branch defeats the
// required-rune and leading-character analyses.
func TestPrefiltersAgreeWithUnfilteredPattern(t *testing.T) {
	patterns := []string{
		`foo`, `(?=bar)\w+`, `(?<=foo)bar`, `(?<!x)abc`, `a|b|c`, `(?:ab|)c`, `a*b`, `a{0}b`,
		`(?i:abc)def`, `(?i)k`, `\bk\w*`, `a++b`, `(a)?b\1`, `(a)?(?(1)b|c)`, `^\s*#\s*include`,
		`(?:^|(?<=\W))x`, `[a-z]{2,}\d`, `\G\s+`, `\d+(?=px)`, `日本`, `🙂+`, `[αβγ]x`,
		`abcdefghijklm`, `(?:\s+|(?<=\W)|(?=\W)|^|\n?$)(?<!\w)(?:const|static)(?!\w)`,
		`(?:(?<![\s\S])|^(?=[\s\S]))\s*\}`, `[[:alpha:]]+:`, `"(?:[^"\\]|\\.)*"`,
	}
	texts := []string{
		"foo bar foobar xabc abc\n",
		"  #  include <x>\nk K static int const\n",
		"aab b c ab abc ababc\n",
		"12px 3em 日本語 🙂🙂 αx βy\n",
		"x = \"a\\\"b\" } abcdefghijklm\n",
		"",
		"\n",
	}
	for _, pattern := range patterns {
		filtered := MustCompile(pattern, Multiline, OptionMaintainCaptureOrder())
		unfiltered := MustCompile(`(?:`+pattern+`)|\w(?<=\x00)`, Multiline, OptionMaintainCaptureOrder())
		if unfiltered.code.RequiredRunes != nil {
			t.Fatalf("%s: wrapper kept required runes", pattern)
		}
		for _, text := range texts {
			runes := []rune(text)
			for start := 0; start <= len(runes); start++ {
				got, err := filtered.FindRunesCaptureIndicesStartingAt(runes, start, nil)
				if err != nil {
					t.Fatal(err)
				}
				want, err := unfiltered.FindRunesCaptureIndicesStartingAt(runes, start, nil)
				if err != nil {
					t.Fatal(err)
				}
				if !reflect.DeepEqual(got, want) {
					t.Fatalf("%s on %q from %d: %v, unfiltered %v", pattern, text, start, got, want)
				}
				for bound := start; bound <= len(runes); bound += 3 {
					got, err := filtered.FindRunesCaptureIndicesStartingAtBefore(runes, start, bound, nil)
					if err != nil {
						t.Fatal(err)
					}
					var wantBounded []CaptureIndex
					if len(want) != 0 && want[0].RuneIndex < bound {
						wantBounded = want
					}
					if len(got) != len(wantBounded) || (len(got) != 0 && !reflect.DeepEqual(got, wantBounded)) {
						t.Fatalf("%s on %q from %d before %d: %v, want %v", pattern, text, start, bound, got, wantBounded)
					}
				}
			}
		}
	}
}
