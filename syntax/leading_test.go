package syntax

import "testing"

func TestAnalyzeLeadingFilter(t *testing.T) {
	scenarios := []struct {
		pattern string
		allowed string
		denied  string
		none    bool
	}{
		{pattern: `(?<!\w)this`, allowed: "t", denied: "hs _"},
		{pattern: `(?<=[(,])\s*(?=<)`, allowed: " \t<", denied: "a(,>"},
		{pattern: `\b(?:async|await)\b`, allowed: "a", denied: "bw "},
		{pattern: `(?!x)[a-c]+`, allowed: "abc", denied: "dx"},
		{pattern: `(?=ab)\w+`, allowed: "a", denied: "b_1"},
		{pattern: `x?y`, allowed: "xy", denied: "z"},
		{pattern: `(?:a|b*)c`, allowed: "abc", denied: "d"},
		{pattern: `(?=a?)b`, allowed: "b", denied: "a"},
		{pattern: `a*`, none: true},
		{pattern: `(?=x)`, allowed: "x", denied: "y"},
		{pattern: `.x`, none: true},
		{pattern: `(?i)ab`, allowed: "aA", denied: "b"},
		{pattern: `(a)\1`, allowed: "a", denied: "b"},
		{pattern: `\1(a)`, none: true},
	}
	for _, s := range scenarios {
		t.Run(s.pattern, func(t *testing.T) {
			tree, err := Parse(s.pattern, ParseOptions{})
			if err != nil {
				t.Fatal(err)
			}
			filter := AnalyzeLeadingFilter(tree)
			if s.none {
				if filter != nil {
					t.Fatalf("filter = %+v, want nil", filter)
				}
				return
			}
			if filter == nil {
				t.Fatal("filter = nil")
			}
			for _, ch := range s.allowed {
				if !filter.Allows(ch) {
					t.Errorf("Allows(%q) = false, want true", ch)
				}
			}
			for _, ch := range s.denied {
				if filter.Allows(ch) {
					t.Errorf("Allows(%q) = true, want false", ch)
				}
			}
		})
	}
}

func TestLeadingFilterIndexFrom(t *testing.T) {
	tree, err := Parse(`(?<!\w)é|<`, ParseOptions{})
	if err != nil {
		t.Fatal(err)
	}
	filter := AnalyzeLeadingFilter(tree)
	text := []rune("ab<cé")
	if got := filter.IndexFrom(text, 0, len(text)); got != 2 {
		t.Fatalf("IndexFrom = %d, want 2", got)
	}
	if got := filter.IndexFrom(text, 3, len(text)); got != 4 {
		t.Fatalf("IndexFrom from 3 = %d, want 4", got)
	}
	if got := filter.IndexFrom(text, 3, 4); got != -1 {
		t.Fatalf("IndexFrom bounded = %d, want -1", got)
	}
}
