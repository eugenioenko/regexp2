package regexp2

import (
	"strings"
	"testing"

	"github.com/eugenioenko/regexp2/v2/syntax"
)

func TestZeroWidthAlternationGroupingPreservesMatches(t *testing.T) {
	tests := []struct {
		pattern, input string
		start          int
		want           string
	}{
		{`(?:\s+|(?<=\W)|(?=\W)|^|\n?$)(?<!\w)(?:const|static)(?!\w)`, "x = static int", 0, " static"},
		{`(?:\s+|(?<=\W)|(?=\W)|^|\n?$)(?<!\w)(?:const|static)(?!\w)`, "  const", 0, "  const"},
		{`(?:(?<=\W)|(?=\W)|\b)(x)y`, "a.xy", 0, "xy"},
		{`(?:(?=a)|(?=b)|c)\w`, "zzbz", 0, "b"},
		{`(?:(?=a)|(?=b)|c)\w`, "zzcz", 0, "cz"},
		{`(?:(?=(a))|(?=a\w))\1`, "ab", 0, "a"},
		{`(?:(?=(a))|(?=(ab)))\2`, "ab", 0, "ab"},
		{`^(?:(?=[\s\S])|(?<![\s\S]))foo`, "x\nfoo\n", 1, "foo"},
		{`(?:\b|$)(?:(?=x)|(?<=y))\w*`, "y.xz", 1, ""},
	}
	for _, test := range tests {
		re := MustCompile(test.pattern, Multiline)
		m, err := re.FindStringMatchStartingAt(test.input, test.start)
		if err != nil {
			t.Fatalf("%s: %v", test.pattern, err)
		}
		got := ""
		if m != nil {
			got = m.String()
		}
		if got != test.want {
			t.Errorf("%s on %q = %q, want %q", test.pattern, test.input, got, test.want)
		}
	}
}

func TestZeroWidthAlternationGroupingShape(t *testing.T) {
	tests := []struct {
		pattern string
		atomic  bool
	}{
		{`(?:a|(?<=\W)|(?=\W)|b)c`, true},
		{`(?:(?<=x)|(?=y))c`, true},
		{`(?:a|(?<=\W)|b|(?=\W))c`, false},
		{`(?:(?=(x))|(?=y))c`, false},
	}
	for _, test := range tests {
		tree, err := syntax.Parse(test.pattern, syntax.ParseOptions{})
		if err != nil {
			t.Fatal(err)
		}
		if got := strings.Contains(tree.Dump(), "Atomic"); got != test.atomic {
			t.Errorf("%s: atomic grouping = %v, want %v\n%s", test.pattern, got, test.atomic, tree.Dump())
		}
	}
}
