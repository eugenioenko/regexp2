package syntax

import "testing"

func TestAnalyzeRequiredRunes(t *testing.T) {
	scenarios := []struct {
		pattern string
		want    string // "" means no required set
	}{
		{`a|b`, "ab"},
		{`a?b`, "b"},
		{`a+b`, "a"},
		{`[a-c]x`, "x"},
		{`(?<=x)y`, "y"},
		{`(?<!x)y`, "y"},
		{`(?!x)y`, "y"},
		{`(?=<)\w+`, "<"},
		{`(foo|bar)+`, "bf"},
		{`a*`, ""},
		{`a{0,3}`, ""},
		{`(?i)a`, "Aa"},
		{`x|\w`, ""},
		{`[^a]`, ""},
		{`(?<=a)`, ""},
		{`\1(a)`, "a"},
		{`^$`, ""},
	}
	for _, s := range scenarios {
		t.Run(s.pattern, func(t *testing.T) {
			tree, err := Parse(s.pattern, ParseOptions{})
			if err != nil {
				t.Fatal(err)
			}
			got := AnalyzeRequiredRunes(tree)
			if s.want == "" {
				if got != nil {
					t.Fatalf("required = %+v, want nil", got)
				}
				return
			}
			if got == nil {
				t.Fatalf("required = nil, want %q", s.want)
			}
			for ch := rune(0); ch < 128; ch++ {
				text := []rune{ch}
				want := containsRune(s.want, ch)
				if found := got.LastIndex(text, 0) == 0; found != want {
					t.Fatalf("contains %q = %v, want %v", ch, found, want)
				}
			}
		})
	}
}

func TestAnalyzeRequiredLiteral(t *testing.T) {
	scenarios := []struct {
		pattern string
		want    string // "" means no literal of two or more runes
	}{
		{`abc`, "abc"},
		{`\bfoo`, "foo"},
		{`a\bb`, "ab"},
		{`x?yz`, "yz"},
		{`(ab)+c`, "ab"},
		{`^\s*([\w\s]*)(enum)\s+(\w+)`, "enum"},
		{`(?=\s*extends)x`, "extends"},
		{`(?:foo)bar`, "foobar"},
		{`foo|bar`, ""},
		{`(?i)abc`, ""},
		{`(?<=abc)d`, ""},
		{`(?!abc)d`, ""},
		{`(ab)?c`, ""},
		{`a*bc*`, ""},
	}
	for _, s := range scenarios {
		t.Run(s.pattern, func(t *testing.T) {
			tree, err := Parse(s.pattern, ParseOptions{})
			if err != nil {
				t.Fatal(err)
			}
			var got string
			if required := AnalyzeRequiredRunes(tree); required != nil {
				got = string(required.literal)
			}
			if got != s.want {
				t.Fatalf("literal = %q, want %q", got, s.want)
			}
		})
	}
}

func TestRequiredLiteralLastIndex(t *testing.T) {
	tree, err := Parse(`enum`, ParseOptions{})
	if err != nil {
		t.Fatal(err)
	}
	required := AnalyzeRequiredRunes(tree)
	text := []rune("enum a enum b enu")
	if got := required.LastIndex(text, 0); got != 7 {
		t.Fatalf("LastIndex = %d, want 7", got)
	}
	if got := required.LastIndex(text, 8); got != -1 {
		t.Fatalf("LastIndex from 8 = %d, want -1", got)
	}
	if got := required.LastIndex([]rune("en"), 0); got != -1 {
		t.Fatalf("LastIndex on short text = %d, want -1", got)
	}
}

func TestRequiredRunesLastIndex(t *testing.T) {
	tree, err := Parse(`é|<`, ParseOptions{})
	if err != nil {
		t.Fatal(err)
	}
	required := AnalyzeRequiredRunes(tree)
	text := []rune("a<bé c")
	if got := required.LastIndex(text, 0); got != 3 {
		t.Fatalf("LastIndex = %d, want 3", got)
	}
	if got := required.LastIndex(text, 4); got != -1 {
		t.Fatalf("LastIndex from 4 = %d, want -1", got)
	}
}

func containsRune(s string, ch rune) bool {
	for _, r := range s {
		if r == ch {
			return true
		}
	}
	return false
}
