package syntax

import "testing"

func TestAnalyzeRequiredRunes(t *testing.T) {
	scenarios := []struct {
		pattern string
		want    string // "" means no required set
	}{
		{`abc`, "a"},
		{`a|b`, "ab"},
		{`a?b`, "b"},
		{`a+b`, "a"},
		{`[a-c]x`, "x"},
		{`(?<=x)y`, "y"},
		{`(?<!x)y`, "y"},
		{`(?!x)y`, "y"},
		{`(?=<)\w+`, "<"},
		{`\bfoo`, "f"},
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
