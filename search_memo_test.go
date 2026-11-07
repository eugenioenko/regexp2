package regexp2

import (
	"math/rand"
	"reflect"
	"strings"
	"testing"
	"time"
)

var searchMemoPatterns = []string{
	`(a|b)+c`, `(?:a|b)*c`, `(?:a|ab)+c`, `(a|ab)(c|bcd)(d*)`, `(?:x+x+)+y`,
	`(a?)*b`, `(a|)*b`, `(?:a*)*b`, `(?:(a)|b)+\1`, `(a+)+\1c`, `\G(?:a|b)+c`,
	`(?:a|b)*?c`, `((a+)b)+c`, `(?:ab|ba)+c`, `(?:ab|a)*b`, `x(?=(?:ab|ba)+c)`, `(?:ab|ba)*$`,
	`(?:(?:ab|ba)+c|a)+`, `(?>(?:ab|a)+)b`, `x(?!(?:ab|ba)+c)`, `(?:ab|(?=ba)b)+a`, `(?:(?:a|b)+c)+d`, `(?:a(?:b|c)+)*d`,
	`x(?=(?:a|b)+c)`, `x(?=(a|b)+c)(a|b)+`, `x(?!(?:a|b)+c)\w`, `(?<=(?:a|b)+)c`,
	`(?>(?:a|b)+)c`, `(?>(?:a|b)+)b`, `(?:(?=(a|b)+c)a)+`, `(?:a|(?=b)(?:b|c)+)+d`,
	`(?<a>x)(?<-a>y)?(?:a|b)+c`, `(?(x)(?:a|b)+c|d)`, `(?:a|b){2,}c`, `(?:a|b){0,5}c`,
	`\w+(?=\s*:\s*(?:=>|\([^()]*\)|[^(),<=>])+=\s*(?:function|\w+\s*=>))`,
	`(\w+)(?:(\?)|(!))?(?=\s*(?::\s*(?:=>|\([^()]*\)|<[^<>]*>|[^(),<=>])+=\s*(?:\w+\s*=>|function)|=\s*function))`,
	`(?=(?:[^"]|"[^"]*")*$)x`, `^(?:(?:a|b)+,)*c$`, `(?m)^(?:a|b)+$`,
}

var searchMemoTexts = []string{
	"", "c", "aaab", "ababababc", "abcbcdd", "xxxxxxxxxxxxx", "aaaaaaaaaab",
	"xaabc xab xbbbc xabd", "abababababababababababababab", "a\nb\nab\nc",
	"aaa aab abab ab ac ad abcd abacd xy xxy", "xyab xy",
	"name: { kind: 'SCALAR'; name: 'String' }; cb: (x: number) => void; f: T = x => x;",
	"a: b; c: d; e: (x) = function g; h = function",
	`x "a" x "b x`,
}

func TestSearchMemoAgreesWithPlainSearch(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	for _, pattern := range searchMemoPatterns {
		checkSearchMemoAgrees(t, rng, pattern, searchMemoTexts)
	}
}

func TestSearchMemoAgreesOnRandomPatterns(t *testing.T) {
	rng := rand.New(rand.NewSource(7))
	alphabet := []string{"aab", "abcabc", "abababc", "cccab", "abc abc ab", "aaaaaaaaab", "b", ""}
	iterations := 3000
	if testing.Short() {
		iterations = 300
	}
	memoized := 0
	for range iterations {
		pattern := randomMemoPattern(rng, 4)
		re, err := Compile(pattern, None)
		if err != nil {
			continue
		}
		if re.code.BranchmarkMemoCount > 0 {
			memoized++
		}
		checkSearchMemoAgrees(t, rng, pattern, alphabet)
	}
	if memoized < iterations/10 {
		t.Fatalf("only %d of %d random patterns had memoized loops", memoized, iterations)
	}
}

func randomMemoPattern(rng *rand.Rand, depth int) string {
	if depth == 0 || rng.Intn(4) == 0 {
		return []string{"a", "b", "c", ".", "[ab]", "a?", "b*", "", `\b`, "$"}[rng.Intn(10)]
	}
	x := randomMemoPattern(rng, depth-1)
	y := randomMemoPattern(rng, depth-1)
	switch rng.Intn(16) {
	case 0:
		return "(" + x + ")+"
	case 1:
		return "(?:" + x + "|" + y + ")*"
	case 2:
		return "(?:" + x + "|" + y + ")+"
	case 3:
		return "(" + x + "|" + y + ")*?"
	case 4:
		return "(?=" + x + ")"
	case 5:
		return "(?!" + x + ")"
	case 6:
		return "(?>" + x + ")"
	case 7:
		return "(?:" + x + "){1,3}"
	case 8:
		return "(?<=" + x + ")"
	case 9:
		return "(" + x + ")" + y + `\1`
	case 10:
		return "(?:" + x + ")?" + y
	default:
		return x + y
	}
}

func checkSearchMemoAgrees(t *testing.T, rng *rand.Rand, pattern string, texts []string) {
	t.Helper()
	re := MustCompile(pattern, Multiline, OptionMaintainCaptureOrder())
	re.MatchTimeout = 100 * time.Millisecond
	for _, text := range texts {
		if !checkSearchMemoText(t, rng, re, pattern, text) {
			return
		}
	}
}

// checkSearchMemoText reports false when the plain search times out, which
// random patterns with nested quantifiers can do.
func checkSearchMemoText(t *testing.T, rng *rand.Rand, re *Regexp, pattern, text string) bool {
	t.Helper()
	{
		runes := []rune(text)
		memo := re.NewSearchMemo()
		type call struct{ start, bound int }
		var calls []call
		for start := 0; start <= len(runes); start++ {
			calls = append(calls, call{start, -1})
			for bound := start; bound <= len(runes)+1; bound += 2 {
				calls = append(calls, call{start, bound})
			}
		}
		rng.Shuffle(len(calls), func(i, j int) { calls[i], calls[j] = calls[j], calls[i] })
		for _, c := range calls {
			want, err := re.FindRunesCaptureIndicesStartingAt(runes, c.start, nil)
			if err != nil && strings.HasPrefix(err.Error(), "match timeout") {
				return false
			}
			if err != nil {
				t.Fatal(err)
			}
			if c.bound >= 0 && len(want) != 0 && want[0].RuneIndex >= c.bound {
				want = nil
			}
			got, err := re.FindRunesCaptureIndicesWithMemo(runes, c.start, c.bound, memo, nil)
			if err != nil {
				t.Fatal(err)
			}
			if len(got) != len(want) || (len(got) != 0 && !reflect.DeepEqual(got, want)) {
				t.Fatalf("%s on %q from %d before %d: memo %v, plain %v", pattern, text, c.start, c.bound, got, want)
			}
			if memo == nil {
				continue
			}
			unbounded, _ := re.FindRunesCaptureIndicesStartingAt(runes, c.start, nil)
			hint := memo.NoMatchBefore()
			if len(got) != 0 && hint != got[0].RuneIndex {
				t.Fatalf("%s on %q from %d before %d: hint %d after match at %d", pattern, text, c.start, c.bound, hint, got[0].RuneIndex)
			}
			if len(unbounded) != 0 && hint > unbounded[0].RuneIndex {
				t.Fatalf("%s on %q from %d before %d: hint %d skips match at %d", pattern, text, c.start, c.bound, hint, unbounded[0].RuneIndex)
			}
		}
	}
	return true
}

func TestSearchMemoEligibleLoops(t *testing.T) {
	tests := []struct {
		pattern string
		want    int
	}{
		{`(a|b)+c`, 1},
		{`(?:ab|cd)*e`, 1},
		{`x(?=(?:ab|cd)+e)`, 1},
		{`(?:(?:ab|cd)+e)+f`, 1},
		{`(?:x(?=(?:ab|cd)+e))+`, 2},
		{`(?>(?:ab|cd)+)e`, 1},
		{`(?:ab|cd)*?e`, 0},
		{`(?:ab|cd){2,}e`, 0},
		{`(e)(?:ab|cd)+\1`, 0},
		{`\G(?:ab|cd)+`, 0},
		{`(?<=(?:ab|cd)+)e`, 0},
		{`(?<x>a)(?<-x>b)(?:ab|cd)+`, 0},
		{`(a)(?(1)(?:ab|cd)+)`, 0},
		{`[ab]+c`, 0},
	}
	for _, tt := range tests {
		re := MustCompile(tt.pattern, None)
		if got := re.code.BranchmarkMemoCount; got != tt.want {
			t.Errorf("%s: %d memoized loops, want %d", tt.pattern, got, tt.want)
		}
	}
	if MustCompile(`(a|b)+c`, RightToLeft).NewSearchMemo() != nil {
		t.Error("right-to-left pattern returned a memo")
	}
}

func TestSearchMemoResetsForNewInput(t *testing.T) {
	re := MustCompile(`x(?=(?:a|b)+c)`, None)
	memo := re.NewSearchMemo()
	first := []rune("xabab xab")
	if got, _ := re.FindRunesCaptureIndicesWithMemo(first, 0, -1, memo, nil); len(got) != 0 {
		t.Fatalf("unexpected match %v", got)
	}
	second := []rune("xabab xabc")
	got, _ := re.FindRunesCaptureIndicesWithMemo(second, 0, -1, memo, nil)
	if len(got) == 0 || got[0].RuneIndex != 6 {
		t.Fatalf("got %v, want a match at 6", got)
	}
	copy(first, []rune("xababcxab"))
	memo.Reset()
	got, _ = re.FindRunesCaptureIndicesWithMemo(first, 0, -1, memo, nil)
	if len(got) == 0 || got[0].RuneIndex != 0 {
		t.Fatalf("after Reset got %v, want a match at 0", got)
	}
}

func TestSearchMemoNoMatchBeforeCrossesBound(t *testing.T) {
	re := MustCompile(`\(`, None)
	memo := re.NewSearchMemo()
	text := []rune(strings.Repeat("a", 100) + "(")
	if got, _ := re.FindRunesCaptureIndicesWithMemo(text, 0, 10, memo, nil); len(got) != 0 {
		t.Fatalf("unexpected match %v", got)
	}
	if hint := memo.NoMatchBefore(); hint != 100 {
		t.Fatalf("NoMatchBefore = %d, want 100", hint)
	}
	if got, _ := re.FindRunesCaptureIndicesWithMemo(text[:100], 0, 10, memo, nil); len(got) != 0 {
		t.Fatalf("unexpected match %v", got)
	}
	if hint := memo.NoMatchBefore(); hint != 101 {
		t.Fatalf("NoMatchBefore = %d, want 101 when the text has no match", hint)
	}
}

// Every identifier followed by ':' starts a lookahead whose loop runs to the
// end of the line. Without the memo each search repeats that scan.
func TestSearchMemoKeepsRepeatedLookaheadLinear(t *testing.T) {
	re := MustCompile(`\w+(?=\s*:\s*(?:=>|\([^()]*\)|[^(),<=>])+=\s*(?:function|\w+\s*=>))`, OptionMaxBacktrackingStackSize(-1))
	re.MatchTimeout = 10 * time.Second
	line := []rune(strings.Repeat("name: 'String'; kind: 'SCALAR'; ", 1500) + "x = 1; y: g => h;")
	memo := re.NewSearchMemo()
	started := time.Now()
	for start := 0; start < len(line); {
		bound := min(len(line), start+16)
		got, err := re.FindRunesCaptureIndicesWithMemo(line, start, bound, memo, nil)
		if err != nil {
			t.Fatal(err)
		}
		if len(got) != 0 {
			t.Fatalf("unexpected match %v", got)
		}
		start = bound
	}
	if elapsed := time.Since(started); elapsed > 3*time.Second {
		t.Fatalf("memoized scan took %s", elapsed)
	}
}
