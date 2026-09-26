package regexp2

import "testing"

func TestEscapedHyphenRangeEndpoints(t *testing.T) {
	tests := []struct {
		pattern string
		match   string
		reject  string
	}{
		{`^[*-\-/]$`, "*+,-/", "."},
		{`^[\--9]$`, "-./09", ",:"},
		{`^[a\-z]$`, "a-z", "bmy"},
		{`^[ !%\&*-\-/<-@\\|~]$`, " !%&*+,-/<=>?@\\|~", ".#a"},
	}
	for _, test := range tests {
		re := MustCompile(test.pattern)
		for _, ch := range test.match {
			if ok, _ := re.MatchString(string(ch)); !ok {
				t.Errorf("%s should match %q", test.pattern, ch)
			}
		}
		for _, ch := range test.reject {
			if ok, _ := re.MatchString(string(ch)); ok {
				t.Errorf("%s should not match %q", test.pattern, ch)
			}
		}
	}
}
