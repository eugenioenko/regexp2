package regexp2

import "testing"

// Grouping literal alternation branches by first character must not change
// which branch wins, including when later text forces backtracking into the
// alternation and when case-insensitive branches share a position.
func TestLiteralAlternationGroupingPreservesMatches(t *testing.T) {
	tests := []struct {
		pattern, input, want string
	}{
		{`(?:hi|there|hello|he)(?:llo!)`, "hello!", "hello!"},
		{`(?:hi|there|hello|h)x`, "hx", "hx"},
		{`(?:ab|c|a|abc)c`, "abc", "abc"},
		{`(?:ab|c|a|abc)$`, "abc", "abc"},
		{`(?:co|const_cast|c|consteval)[a-z_]*`, "consteval", "consteval"},
		{`(?:x|(?i:HELLO)|hel|h)lo`, "hello", "hello"},
		{`(?:x|(?i:H)|hel)lo`, "hello", "hello"},
		{`(\w+)(?:foo|bar|f)(oo)`, "xfoo", "xfoo"},
	}
	for _, test := range tests {
		re := MustCompile(test.pattern)
		m, err := re.FindStringMatch(test.input)
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

func TestLiteralAlternationGroupingKeepsBranchPriority(t *testing.T) {
	re := MustCompile(`(ab|x|a|y|abc)`)
	m, err := re.FindStringMatch("abc")
	if err != nil || m == nil || m.GroupByNumber(1).String() != "ab" {
		t.Fatalf("match = %v, %v; want first branch ab", m, err)
	}
}
