package regexp2

import "testing"

func TestStartPrefilters(t *testing.T) {
	tests := []struct {
		name    string
		pattern string
		options RegexOptions
		input   string
		startAt int
		want    int
	}{
		{"bol skips to next line", `^ab`, Multiline, "x\nab", 0, 2},
		{"bol without line start", `^ab`, Multiline, "xab", 0, -1},
		{"bol from inside a line", `^ab`, Multiline, "xab\nab", 1, 4},
		{"bol empty last line", `^$`, Multiline, "a\n", 1, 2},
		{"bol in every branch", `^b|^c`, Multiline, "a\nc", 0, 2},
		{"bol after lookahead", `(?=a)^a`, Multiline, "ba\na", 0, 3},
		{"bol in one branch only", `^a|b`, Multiline, "xb", 0, 1},
		{"beginning is not bol", `^a`, None, "b\na", 0, -1},
		{"bol right to left", `^a`, Multiline | RightToLeft, "b\na", 3, 2},
		{"literal present", `^\s*([\w\s]*)(enum)\s+(\w+)`, Multiline, "  public enum Color", 0, 0},
		{"literal absent", `^\s*([\w\s]*)(enum)\s+(\w+)`, Multiline, "  public class Color", 0, -1},
		{"literal after boundary", `\bfoo`, None, "xfoo foo", 0, 5},
		{"literal after optional", `x?yz`, None, "ayz", 0, 1},
		{"literal split", `x?yz`, None, "xy z", 0, -1},
		{"literal before start", `foo`, None, "foo bar", 1, -1},
		{"literal after start", `foo`, None, "foofoo", 1, 3},
		{"literal in lookahead", `\w+(?=bar)`, None, "foobar", 0, 0},
		{"literal in lookahead far", `a(?=.*end)`, None, "a b end", 0, 0},
		{"literal ignore case", `(?i)ENUM`, None, "enum", 0, 0},
		{"literal non-ascii", `é界`, None, "aé界", 0, 1},
		{"literal right to left", `ab`, RightToLeft, "abab", 4, 2},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			re := MustCompile(test.pattern, test.options)
			input := []rune(test.input)
			m, err := re.FindRunesMatchStartingAt(input, test.startAt)
			if err != nil {
				t.Fatal(err)
			}
			got := -1
			if m != nil {
				got = m.RuneIndex
			}
			if got != test.want {
				t.Fatalf("match index = %d, want %d", got, test.want)
			}
			if test.options&RightToLeft != 0 {
				return
			}
			indices, err := re.FindRunesCaptureIndicesStartingAt(input, test.startAt, nil)
			if err != nil {
				t.Fatal(err)
			}
			got = -1
			if len(indices) != 0 {
				got = indices[0].RuneIndex
			}
			if got != test.want {
				t.Fatalf("capture index = %d, want %d", got, test.want)
			}
		})
	}
}

func TestStartPrefilterRespectsBound(t *testing.T) {
	re := MustCompile(`foo`, None)
	input := []rune("xfoo")
	for _, test := range []struct {
		bound int
		want  int
	}{{1, -1}, {2, 1}} {
		indices, err := re.FindRunesCaptureIndicesStartingAtBefore(input, 0, test.bound, nil)
		if err != nil {
			t.Fatal(err)
		}
		got := -1
		if len(indices) != 0 {
			got = indices[0].RuneIndex
		}
		if got != test.want {
			t.Fatalf("bound %d: index = %d, want %d", test.bound, got, test.want)
		}
	}
}

func TestLastPossibleStart(t *testing.T) {
	tests := []struct {
		pattern string
		options RegexOptions
		input   string
		want    int
	}{
		{`foo`, None, "foo foo", 4},
		{`foo`, None, "bar", -1},
		{`[<>]`, None, "a<b>c", 3},
		{`a*`, None, "bbb", 3},
		{`\w+(?=bar)`, None, "foobar", 3},
		{`foo`, RightToLeft, "foo", 3},
	}
	for _, test := range tests {
		t.Run(test.pattern, func(t *testing.T) {
			re := MustCompile(test.pattern, test.options)
			if got := re.LastPossibleStart([]rune(test.input)); got != test.want {
				t.Fatalf("LastPossibleStart = %d, want %d", got, test.want)
			}
		})
	}
}
