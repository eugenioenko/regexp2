package syntax

import "testing"

func TestRequiredBranches(t *testing.T) {
	const pattern = `\w+(?=:\s*(?:Function|\([^)]*\)|[^;=]*=\s*function))`
	tests := []struct {
		pattern string
		text    string
		start   int
		want    int
	}{
		{pattern, `name: { kind: 'SCALAR'; name: 'String' }`, 0, -1},
		{pattern, `name: Function`, 0, 4},
		{pattern, `name: (x)`, 0, 4},
		{pattern, `name: Type = function`, 0, 4},
		{pattern, `name: Function`, 7, -1},
		{`x(?=(?:ab|cd)z)`, `x abz`, 0, 0},
		{`x(?=(?:ab|cd)z)`, `x ab`, 0, -1},
		{`x(?=(?:ab|cd)z)`, `x cdz`, 0, 0},
		{`x(?=(?:ab|)z)`, `x z`, 0, 0},
		{`x(?=(?:ab|(?i:CD))z)`, `x CDz`, 0, 0},
		{`x(?=(?:éa|界b)z)`, `x 界bz`, 0, 0},
		{`x(?=(?:ab|cd)?)`, `x`, 0, 0},
		{`x(?=(?:ab|cd)+)`, `x cd`, 0, 0},
		{`x(?=(?:ab|cd)+)`, `x ef`, 0, -1},
	}
	for _, tc := range tests {
		t.Run(tc.pattern+"/"+tc.text, func(t *testing.T) {
			tree, err := Parse(tc.pattern, ParseOptions{})
			if err != nil {
				t.Fatal(err)
			}
			required := AnalyzeRequiredBranches(tree)
			if required == nil {
				t.Fatal("missing branch requirements")
			}
			if got := required.LastIndex([]rune(tc.text), tc.start); got != tc.want {
				t.Fatalf("last start = %d, want %d", got, tc.want)
			}
		})
	}
}

func TestRequiredBranchesSkipsUnconstrainedAssertions(t *testing.T) {
	for _, pattern := range []string{
		`(?<=ab|cd)x`, `(?<!ab|cd)x`, `x(?!ab|cd)`, `(?=(?:ab|))`,
	} {
		tree, err := Parse(pattern, ParseOptions{})
		if err != nil {
			t.Fatal(err)
		}
		if got := AnalyzeRequiredBranches(tree); got != nil {
			t.Fatalf("%s: unexpected requirements %+v", pattern, got)
		}
	}
}
