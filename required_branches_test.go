package regexp2

import (
	"strings"
	"testing"
)

func TestLastPossibleStartRejectsMissingFunctionMarkers(t *testing.T) {
	re := MustCompile(`\w+(?=:\s*(?:Function|\([^)]*\)|[^;=]*=\s*function))`)
	text := []rune(strings.Repeat(`name: { kind: 'SCALAR'; name: 'String' }; `, 60))
	if got := re.LastPossibleStart(text); got != -1 {
		t.Fatalf("last possible start = %d, want no candidate", got)
	}
	for _, suffix := range []string{`handler: Function`, `handler: (value)`, `handler: Type = function`} {
		withFunction := append(append([]rune(nil), text...), []rune(suffix)...)
		got, err := re.FindRunesCaptureIndicesStartingAt(withFunction, 0, nil)
		if err != nil {
			t.Fatal(err)
		}
		if len(got) == 0 || got[0].RuneIndex != len(text) || got[0].RuneLength != len("handler") {
			t.Fatalf("%s: captures = %+v, want handler after fields", suffix, got)
		}
	}
}

func TestRequiredBranchesPreservesMatchSemantics(t *testing.T) {
	tests := []struct {
		pattern string
		text    string
		start   int
		options RegexOptions
		index   int
		length  int
	}{
		{`\G\w+(?=:(?:foo|bar))`, `é name:foo`, 2, 0, 2, 4},
		{`\G\w+(?=:(?:foo|bar))`, `é name:foo`, 1, 0, -1, 0},
		{`name(?=:(?:foo|bar))`, `NAME:BAR`, 0, IgnoreCase, 0, 4},
		{`(?<=foo|bar)x(?=a|bc)`, `barxbc`, 0, 0, 3, 1},
		{`x(?<!ab|cd)(?=a|bc)`, `xbc`, 0, 0, 0, 1},
		{`^x(?=(?:a|bc))`, "z\nxbc", 0, Multiline, 2, 1},
		{`é(?=(?:界a|🙂b))`, `é🙂b`, 0, 0, 0, 1},
		{`x(?=(?:ab|cd)?)`, `x`, 0, 0, 0, 1},
		{`x(?=(?:ab|))`, `x`, 0, 0, 0, 1},
		{`x(?=(?:ab|cd))`, `xab xcd`, -1, RightToLeft, 4, 1},
	}
	for _, tc := range tests {
		t.Run(tc.pattern+"/"+tc.text, func(t *testing.T) {
			re := MustCompile(tc.pattern, tc.options)
			got, err := re.FindRunesCaptureIndicesStartingAt([]rune(tc.text), tc.start, nil)
			if err != nil {
				t.Fatal(err)
			}
			if tc.index < 0 {
				if len(got) != 0 {
					t.Fatalf("unexpected captures %+v", got)
				}
			} else if len(got) == 0 || got[0].RuneIndex != tc.index || got[0].RuneLength != tc.length {
				t.Fatalf("captures = %+v, want [%d,%d)", got, tc.index, tc.index+tc.length)
			}
		})
	}
}

func TestRequiredBranchesAnalysisLimitPreservesMatches(t *testing.T) {
	pattern := `x(?=(?:` + strings.Repeat(`(a?b)`, 300) + `z|cd))`
	text := "x" + strings.Repeat("b", 300) + "z"
	re := MustCompile(pattern)
	got, err := re.FindRunesCaptureIndicesStartingAt([]rune(text), 0, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) == 0 || got[0].RuneIndex != 0 || got[0].RuneLength != 1 {
		t.Fatalf("captures = %+v, want x at [0,1)", got)
	}
}
