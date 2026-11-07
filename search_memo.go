package regexp2

import "github.com/eugenioenko/regexp2/v2/syntax"

// SearchMemo carries what searches of one input have proved, so later
// searches of the same input with FindRunesCaptureIndicesWithMemo do not
// repeat that work:
//
//   - Loop states that cannot lead to a match are recorded and skipped. A
//     pattern whose loop rescans the same suffix from every candidate start
//     (for example a lookahead that runs to the end of the line) then costs
//     time linear in the input instead of quadratic. Patterns with
//     backreferences, conditionals on captures, balancing groups or \G record
//     nothing.
//   - NoMatchBefore reports how far the last search proved there is no match,
//     which can exceed a maxStartExclusive bound.
//
// A memo belongs to the Regexp that created it and to one input. It resets
// itself when used with a different rune slice; call Reset after modifying
// the contents of the same slice. A SearchMemo is not safe for concurrent use.
type SearchMemo struct {
	code          *syntax.Code
	text          *rune
	n             int
	slots         [][]uint64
	noMatchBefore int
	// requiredLimit is requiredStartLimit from 0, or -2 when not computed.
	// The limit from any start is the same position when it lies after that
	// start, and -1 otherwise.
	requiredLimit int
}

// NewSearchMemo returns a memo for re, or nil for right-to-left patterns.
func (re *Regexp) NewSearchMemo() *SearchMemo {
	if re.code == nil || re.code.RightToLeft {
		return nil
	}
	return &SearchMemo{code: re.code, slots: make([][]uint64, re.code.BranchmarkMemoCount), requiredLimit: -2}
}

// Reset forgets everything recorded for the current input.
func (m *SearchMemo) Reset() {
	if m == nil {
		return
	}
	m.text = nil
	m.n = 0
	m.noMatchBefore = 0
	m.requiredLimit = -2
	for i := range m.slots {
		m.slots[i] = m.slots[i][:0]
	}
}

// NoMatchBefore reports that no match starts at or after the startAt of the
// last search and before the returned position. After a match it is the
// match's start. len(input)+1 means no match starts at or after startAt.
func (m *SearchMemo) NoMatchBefore() int {
	return m.noMatchBefore
}

func (m *SearchMemo) bind(text []rune) {
	var first *rune
	if len(text) > 0 {
		first = &text[0]
	}
	if m.text == first && m.n == len(text) {
		return
	}
	m.Reset()
	m.text = first
	m.n = len(text)
}

func (m *SearchMemo) requiredStartLimit(start int, text []rune) int {
	if m.requiredLimit == -2 {
		m.requiredLimit = requiredStartLimit(m.code, text, 0)
	}
	if m.requiredLimit > start {
		return m.requiredLimit
	}
	return -1
}

func (m *SearchMemo) failed(slot int32, pos int) bool {
	bits := m.slots[slot]
	word := pos >> 6
	return word < len(bits) && bits[word]&(1<<(pos&63)) != 0
}

func (m *SearchMemo) markFailed(slot int32, pos int) {
	bits := m.slots[slot]
	if len(bits) == 0 {
		words := m.n>>6 + 1
		if cap(bits) >= words {
			bits = bits[:words]
			clear(bits)
		} else {
			bits = make([]uint64, words)
		}
		m.slots[slot] = bits
	}
	bits[pos>>6] |= 1 << (pos & 63)
}

// FindRunesCaptureIndicesWithMemo is FindRunesCaptureIndicesStartingAt, or
// FindRunesCaptureIndicesStartingAtBefore when maxStartExclusive is not
// negative, reusing and extending memo. Results are identical to the same
// search without a memo. A nil memo, or one created by another Regexp, is
// ignored.
func (re *Regexp) FindRunesCaptureIndicesWithMemo(r []rune, startAt, maxStartExclusive int, memo *SearchMemo, dst []CaptureIndex) ([]CaptureIndex, error) {
	if maxStartExclusive >= 0 && re.RightToLeft() {
		return dst[:0], errRightToLeftStartBound
	}
	return re.findRunesCaptureIndicesWithMemo(r, startAt, maxStartExclusive, memo, dst)
}
