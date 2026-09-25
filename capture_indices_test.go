package regexp2

import (
	"errors"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestFindRunesCaptureIndicesStartingAtMatchesPublicGroups(t *testing.T) {
	tests := []struct {
		name    string
		pattern string
		input   string
		startAt int
		options []CompileOption
	}{
		{
			name:    "unmatched and empty",
			pattern: `^(a)?()b$`,
			input:   "b",
			startAt: 0,
		},
		{
			name:    "repeated capture uses last",
			pattern: `(?<letter>[a-z])+`,
			input:   "abc",
			startAt: 0,
		},
		{
			name:    "mixed named unnamed dotnet order",
			pattern: `(?<first>this).+?(testing).+?(?<last>stuff)`,
			input:   "this is a testing stuff",
			startAt: 0,
		},
		{
			name:    "mixed named unnamed pattern order",
			pattern: `(?<first>this).+?(testing).+?(?<last>stuff)`,
			input:   "this is a testing stuff",
			startAt: 0,
			options: []CompileOption{OptionMaintainCaptureOrder()},
		},
		{
			name:    "sparse explicit numbers",
			pattern: `((?<256>abc)\d+)?(?<16>xyz)(.*)`,
			input:   "abc8978xyz][]12_+-",
			startAt: 0,
		},
		{
			name:    "balanced capture removed",
			pattern: `^(?<open>a)+(?<-open>b)+(?(open)(?!))$`,
			input:   "aaabbb",
			startAt: 0,
		},
		{
			name:    "unicode start offset",
			pattern: `(needle)(?:-(\p{L}+))?`,
			input:   "é界 needle-élan",
			startAt: 3,
		},
		{
			name:    "right to left",
			pattern: `(\w+)`,
			input:   "one two three",
			startAt: -1,
			options: []CompileOption{RightToLeft},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			re := MustCompile(tc.pattern, tc.options...)
			input := []rune(tc.input)

			match, err := re.FindRunesMatchStartingAt(input, tc.startAt)
			if err != nil {
				t.Fatalf("FindRunesMatchStartingAt failed: %v", err)
			}
			want := publicCaptureIndices(re, match)

			got, err := re.FindRunesCaptureIndicesStartingAt(input, tc.startAt, nil)
			if err != nil {
				t.Fatalf("FindRunesCaptureIndicesStartingAt failed: %v", err)
			}
			if !reflect.DeepEqual(got, want) {
				t.Fatalf("capture indexes = %#v, want %#v", got, want)
			}
		})
	}
}

func TestFindRunesCaptureIndicesStartingAtGroupOrder(t *testing.T) {
	tests := []struct {
		name    string
		options []CompileOption
		want    []int
	}{
		{
			name: "dotnet order",
			want: []int{0, 1, 2, 3},
		},
		{
			name:    "pattern order",
			options: []CompileOption{OptionMaintainCaptureOrder()},
			want:    []int{0, 1, 2, 3},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			re := MustCompile(`(?<first>this).+?(testing).+?(?<last>stuff)`, tc.options...)
			got, err := re.FindRunesCaptureIndicesStartingAt([]rune("this is a testing stuff"), 0, nil)
			if err != nil {
				t.Fatal(err)
			}
			gotNumbers := make([]int, len(got))
			for i := range got {
				gotNumbers[i] = got[i].GroupNumber
			}
			if !reflect.DeepEqual(gotNumbers, tc.want) {
				t.Fatalf("group numbers = %v, want %v", gotNumbers, tc.want)
			}
			if !reflect.DeepEqual(gotNumbers, re.GetGroupNumbers()) {
				t.Fatalf("group numbers = %v, GetGroupNumbers = %v", gotNumbers, re.GetGroupNumbers())
			}
		})
	}

	re := MustCompile(`((?<256>abc)\d+)?(?<16>xyz)(.*)`)
	got, err := re.FindRunesCaptureIndicesStartingAt([]rune("abc8978xyz][]12_+-"), 0, nil)
	if err != nil {
		t.Fatal(err)
	}
	gotNumbers := make([]int, len(got))
	for i := range got {
		gotNumbers[i] = got[i].GroupNumber
	}
	wantNumbers := []int{0, 1, 2, 16, 256}
	if !reflect.DeepEqual(gotNumbers, wantNumbers) {
		t.Fatalf("sparse group numbers = %v, want %v", gotNumbers, wantNumbers)
	}
}

func TestFindRunesCaptureIndicesStartingAtAnchors(t *testing.T) {
	tests := []struct {
		name    string
		pattern string
		startAt int
		matched bool
	}{
		{name: "G at startAt", pattern: `\G(foo)`, startAt: 2, matched: true},
		{name: "G rejects later candidate", pattern: `\G(foo)`, startAt: 1, matched: false},
		{name: "beginning rejects startAt", pattern: `^(foo)`, startAt: 2, matched: false},
	}
	input := []rune("xxfoo")
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			re := MustCompile(tc.pattern)
			got, err := re.FindRunesCaptureIndicesStartingAt(input, tc.startAt, nil)
			if err != nil {
				t.Fatal(err)
			}
			if matched := len(got) != 0; matched != tc.matched {
				t.Fatalf("matched = %v, want %v; captures %#v", matched, tc.matched, got)
			}
		})
	}
}

func TestFindRunesCaptureIndicesStartingAtNoMatchAndDstReuse(t *testing.T) {
	re := MustCompile(`(foo)(bar)?`)

	got, err := re.FindRunesCaptureIndicesStartingAt([]rune("none"), 0, nil)
	if err != nil {
		t.Fatal(err)
	}
	if got != nil {
		t.Fatalf("nil dst on no match returned %#v, want nil", got)
	}

	dst := make([]CaptureIndex, 0, 3)
	first := &dst[:cap(dst)][0]
	got, err = re.FindRunesCaptureIndicesStartingAt([]rune("foobar"), 0, dst)
	if err != nil {
		t.Fatal(err)
	}
	if &got[0] != first {
		t.Fatal("provided destination storage was not reused")
	}
	saved := append([]CaptureIndex(nil), got...)

	// A later scan reuses the runner but must not alter the previously returned
	// caller-owned result.
	if _, err := re.FindRunesCaptureIndicesStartingAt([]rune("foo"), 0, nil); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, saved) {
		t.Fatalf("returned captures changed after runner reuse: got %#v, want %#v", got, saved)
	}
}

func TestFindRunesCaptureIndicesStartingAtTimeout(t *testing.T) {
	re := MustCompile(`(.+)*\?`)
	re.MatchTimeout = -time.Millisecond
	got, err := re.FindRunesCaptureIndicesStartingAt([]rune("Do you think you found the problem string!"), 0, nil)
	if err == nil || !strings.Contains(err.Error(), "match timeout") {
		t.Fatalf("error = %v, want match timeout", err)
	}
	if got != nil {
		t.Fatalf("captures = %#v, want nil on error", got)
	}
}

func TestFindRunesCaptureIndicesStartingAtConcurrent(t *testing.T) {
	re := MustCompile(`(?<word>[a-z]+)-(\d+)?`)
	inputs := []string{"alpha-1", "beta-22", "gamma-", "delta-4444"}
	wants := make([][]CaptureIndex, len(inputs))
	for i, input := range inputs {
		match, err := re.FindRunesMatchStartingAt([]rune(input), 0)
		if err != nil {
			t.Fatal(err)
		}
		wants[i] = publicCaptureIndices(re, match)
	}

	const iterations = 100
	var wg sync.WaitGroup
	errs := make(chan error, len(inputs))
	results := make([][]CaptureIndex, len(inputs))
	for i, input := range inputs {
		wg.Add(1)
		go func(i int, input string) {
			defer wg.Done()
			var dst []CaptureIndex
			for range iterations {
				var err error
				dst, err = re.FindRunesCaptureIndicesStartingAt([]rune(input), 0, dst)
				if err != nil {
					errs <- err
					return
				}
				if !reflect.DeepEqual(dst, wants[i]) {
					errs <- errors.New("concurrent capture mismatch")
					return
				}
			}
			results[i] = dst
		}(i, input)
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Fatal(err)
	}

	// Each result remains stable and independent after every runner has returned
	// to the pool and other goroutines have completed scans.
	for i := range results {
		if !reflect.DeepEqual(results[i], wants[i]) {
			t.Fatalf("result %d changed: got %#v, want %#v", i, results[i], wants[i])
		}
	}
}

func TestFindRunesCaptureIndicesStartingAtRegisteredEngineUsesFullCaptures(t *testing.T) {
	const pattern = "capture-index-registered-engine"
	fullCalls := 0
	quickCalls := 0
	RegisterEngine(pattern, RuntimeEngineData{
		CapSize: 3,
		FindFirstChar: func(r *Runner) bool {
			return r.Runtextpos == 0
		},
		Execute: func(r *Runner) error {
			fullCalls++
			r.Capture(1, 0, 1)
			r.Capture(2, 1, 2)
			r.Capture(0, 0, 2)
			return nil
		},
		ExecuteQuick: func(r *Runner) error {
			quickCalls++
			r.Capture(0, 0, 2)
			return nil
		},
	}, OptionMaintainCaptureOrder())

	re := MustCompile(pattern, OptionMaintainCaptureOrder())
	got, err := re.FindRunesCaptureIndicesStartingAt([]rune("ab"), 0, nil)
	if err != nil {
		t.Fatal(err)
	}
	want := []CaptureIndex{
		{GroupNumber: 0, RuneIndex: 0, RuneLength: 2},
		{GroupNumber: 1, RuneIndex: 0, RuneLength: 1},
		{GroupNumber: 2, RuneIndex: 1, RuneLength: 1},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("capture indexes = %#v, want %#v", got, want)
	}
	if fullCalls != 1 || quickCalls != 0 {
		t.Fatalf("full/quick calls = %d/%d, want 1/0", fullCalls, quickCalls)
	}
}

func publicCaptureIndices(re *Regexp, match *Match) []CaptureIndex {
	if match == nil {
		return nil
	}
	groups := match.Groups()
	numbers := re.GetGroupNumbers()
	result := make([]CaptureIndex, len(groups))
	for i, group := range groups {
		result[i] = CaptureIndex{GroupNumber: numbers[i], RuneIndex: -1}
		if len(group.Captures) != 0 {
			result[i].RuneIndex = group.RuneIndex
			result[i].RuneLength = group.RuneLength
		}
	}
	return result
}

var benchmarkCaptureIndex int

func BenchmarkFindRunesCaptureIndicesStartingAt(b *testing.B) {
	re := MustCompile(`(?<one>foo)(bar)?(?<three>baz)?(quux)?`, OptionMaintainCaptureOrder())
	input := []rune("prefix foobarbazquux suffix")

	b.Run("MatchGroups", func(b *testing.B) {
		for b.Loop() {
			match, err := re.FindRunesMatchStartingAt(input, 0)
			if err != nil {
				b.Fatal(err)
			}
			groups := match.Groups()
			for i := range groups {
				benchmarkCaptureIndex += groups[i].RuneIndex + groups[i].RuneLength
			}
		}
	})

	b.Run("CaptureIndicesReused", func(b *testing.B) {
		dst := make([]CaptureIndex, 0, 5)
		for b.Loop() {
			var err error
			dst, err = re.FindRunesCaptureIndicesStartingAt(input, 0, dst)
			if err != nil {
				b.Fatal(err)
			}
			for i := range dst {
				benchmarkCaptureIndex += dst[i].RuneIndex + dst[i].RuneLength
			}
		}
	})
}
