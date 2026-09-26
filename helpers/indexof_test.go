package helpers

import (
	"testing"

	"github.com/eugenioenko/regexp2/v2/syntax"
)

func TestIndexOf_Miss(t *testing.T) {
	i := IndexOf([]rune("GHMJ"), []rune("HIJ"))
	if want, got := -1, i; want != got {
		t.Fatalf("Expected %v got %v", want, got)
	}
}

func TestIndexOf_Hit0(t *testing.T) {
	i := IndexOf([]rune("the quick brown fox"), []rune("the quick brown fox"))
	if want, got := 0, i; want != got {
		t.Fatalf("Expected %v got %v", want, got)
	}
}

func TestIndexOf_IgnoreCaseHit(t *testing.T) {
	i := IndexOfIgnoreCase([]rune("what do you know of the quick brown fox?"), []rune("the quick brown fox"))
	if want, got := 20, i; want != got {
		t.Fatalf("Expected %v got %v", want, got)
	}
}

func TestIndexOfIgnoreCaseAscii_Hit(t *testing.T) {
	i := IndexOfIgnoreCaseAscii([]rune("what do you know of the quick brown fox?"), []rune("THE QUICK BROWN FOX"))
	if want, got := 20, i; want != got {
		t.Fatalf("Expected %v got %v", want, got)
	}
}

func TestIndexStringIgnoreCaseASCII(t *testing.T) {
	i := IndexStringIgnoreCaseASCII("0123456789TeStToken", "testtoken")
	if want, got := 10, i; want != got {
		t.Fatalf("Expected %v got %v", want, got)
	}

	i = IndexStringIgnoreCaseASCII("0123456789", "testtoken")
	if want, got := -1, i; want != got {
		t.Fatalf("Expected %v got %v", want, got)
	}
}

func TestStartsWith_Miss(t *testing.T) {
	if want, got := false, StartsWith([]rune("GHMJ")[1:], []rune("HIJ")); want != got {
		t.Fatalf("Expected %v got %v", want, got)
	}
}

func TestIndexOf_UnalignedFalsePositive(t *testing.T) {
	// U+0100 is 0x00,0x01,0x00,0x00 little-endian. Searching for NUL
	// (0x00,0x00,0x00,0x00) must not match inside that rune.
	in := []rune{0x0100, 0x0000}
	if got := IndexOfAny1(in, 0); got != 1 {
		t.Fatalf("IndexOfAny1 NUL = %d, want 1", got)
	}
	if got := IndexOf(in, []rune{0}); got != 1 {
		t.Fatalf("IndexOf NUL = %d, want 1", got)
	}
}

func TestIndexOfAny_ASCIISet(t *testing.T) {
	in := []rune("zzzzzzzzzzzzzzqzzzz")
	if got, want := IndexOfAny(in, []rune("abcq")), 14; got != want {
		t.Fatalf("IndexOfAny = %d, want %d", got, want)
	}
	if got := IndexOfAny(in, []rune("ABC")); got != -1 {
		t.Fatalf("IndexOfAny miss = %d", got)
	}
}

func TestLastIndexOf(t *testing.T) {
	in := []rune("the cat sat on the mat")
	if got, want := LastIndexOf(in, []rune("the")), 15; got != want {
		t.Fatalf("LastIndexOf = %d, want %d", got, want)
	}
	if got := LastIndexOf(in, []rune("dog")); got != -1 {
		t.Fatalf("LastIndexOf miss = %d", got)
	}
}

func TestIndexOfAnyExceptInSet(t *testing.T) {
	if got := IndexOfAnyExceptInSet(nil, *syntax.AnyClass()); got != -1 {
		t.Fatalf("empty input = %d", got)
	}
	if got := IndexOfAnyExceptInSet([]rune("abc"), *syntax.AnyClass()); got != -1 {
		t.Fatalf("any class = %d, want -1", got)
	}
	if got := IndexOfAnyExceptInSet([]rune("abc"), *syntax.NoneClass()); got != 0 {
		t.Fatalf("none class = %d, want 0", got)
	}
}
