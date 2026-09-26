package syntax

import "slices"

const (
	maxLeadingSets  = 16
	maxLeadingRunes = 64
)

// LeadingFilter tests whether a rune can begin a match. It is used when the
// prefix analyzer finds no search strategy, typically because the pattern opens
// with lookarounds: zero-width assertions are skipped, and a positive
// lookahead with nothing consumed before it contributes its own first
// characters.
type LeadingFilter struct {
	ascii [2]uint64
	runes []rune
	sets  []*CharSet
}

type leadingInfo struct {
	any      bool
	nullable bool
	runes    []rune
	sets     []*CharSet
}

func leadingFilterFor(tree *RegexTree) *LeadingFilter {
	if tree.FindOptimizations != nil && tree.FindOptimizations.FindMode != NoSearch {
		return nil
	}
	return AnalyzeLeadingFilter(tree)
}

// AnalyzeLeadingFilter returns nil when a match may begin with any rune or may
// be empty.
func AnalyzeLeadingFilter(tree *RegexTree) *LeadingFilter {
	if tree == nil || tree.Root == nil || tree.Options&RightToLeft != 0 {
		return nil
	}
	info := leading(tree.Root)
	if info.any || info.nullable {
		return nil
	}
	filter := &LeadingFilter{sets: info.sets}
	for _, ch := range info.runes {
		if ch >= 0 && ch < 128 {
			filter.ascii[ch>>6] |= 1 << (ch & 63)
		} else if !slices.Contains(filter.runes, ch) {
			filter.runes = append(filter.runes, ch)
		}
	}
	for ch := rune(0); ch < 128; ch++ {
		for _, set := range info.sets {
			if set.Contains(ch) {
				filter.ascii[ch>>6] |= 1 << (ch & 63)
				break
			}
		}
	}
	return filter
}

// IndexFrom returns the first index in [start, end) whose rune can begin a
// match, or -1.
func (f *LeadingFilter) IndexFrom(text []rune, start, end int) int {
	for i := start; i < end; i++ {
		if f.Allows(text[i]) {
			return i
		}
	}
	return -1
}

func (f *LeadingFilter) Allows(ch rune) bool {
	if ch >= 0 && ch < 128 {
		return f.ascii[ch>>6]&(1<<(ch&63)) != 0
	}
	if slices.Contains(f.runes, ch) {
		return true
	}
	for _, set := range f.sets {
		if set.Contains(ch) {
			return true
		}
	}
	return false
}

var (
	leadingAny      = leadingInfo{any: true}
	leadingZeroWide = leadingInfo{nullable: true}
)

func leading(n *RegexNode) leadingInfo {
	if n.T == NtNegLook || n.T == NtPosLook && n.Options&RightToLeft != 0 {
		return leadingZeroWide
	}
	if n.Options&RightToLeft != 0 {
		return leadingAny
	}
	switch n.T {
	case NtOne:
		if n.Options&IgnoreCase != 0 {
			return leadingAny
		}
		return leadingInfo{runes: []rune{n.Ch}}
	case NtMulti:
		if n.Options&IgnoreCase != 0 || len(n.Str) == 0 {
			return leadingAny
		}
		return leadingInfo{runes: []rune{n.Str[0]}}
	case NtOneloop, NtOnelazy, NtOneloopatomic:
		if n.Options&IgnoreCase != 0 {
			return leadingAny
		}
		return leadingInfo{runes: []rune{n.Ch}, nullable: n.M == 0}
	case NtSet:
		return leadingInfo{sets: []*CharSet{n.Set}}
	case NtSetloop, NtSetlazy, NtSetloopatomic:
		return leadingInfo{sets: []*CharSet{n.Set}, nullable: n.M == 0}
	case NtCapture, NtGroup, NtAtomic:
		if len(n.Children) != 1 {
			return leadingAny
		}
		return leading(n.Children[0])
	case NtLoop, NtLazyloop:
		if len(n.Children) != 1 {
			return leadingAny
		}
		info := leading(n.Children[0])
		info.nullable = info.nullable || n.M == 0
		return info
	case NtConcatenate:
		var acc leadingInfo
		for _, child := range n.Children {
			info := leading(child)
			if info.any {
				return leadingAny
			}
			acc = mergeLeading(acc, info)
			if acc.any {
				return leadingAny
			}
			if !info.nullable {
				return acc
			}
		}
		acc.nullable = true
		return acc
	case NtAlternate:
		var acc leadingInfo
		for _, child := range n.Children {
			info := leading(child)
			if info.any {
				return leadingAny
			}
			nullable := acc.nullable || info.nullable
			acc = mergeLeading(acc, info)
			if acc.any {
				return leadingAny
			}
			acc.nullable = nullable
		}
		return acc
	case NtPosLook:
		// A lookahead that must consume constrains the rune at the current
		// position.
		if len(n.Children) != 1 {
			return leadingZeroWide
		}
		info := leading(n.Children[0])
		if info.any || info.nullable {
			return leadingZeroWide
		}
		return info
	case NtEmpty, NtBol, NtEol, NtBoundary, NtNonboundary, NtECMABoundary,
		NtNonECMABoundary, NtBeginning, NtStart, NtEndZ, NtEnd, NtUpdateBumpalong:
		return leadingZeroWide
	case NtNothing:
		return leadingInfo{}
	}
	return leadingAny
}

func mergeLeading(a, b leadingInfo) leadingInfo {
	out := leadingInfo{
		runes: append(slices.Clone(a.runes), b.runes...),
		sets:  append(slices.Clone(a.sets), b.sets...),
	}
	if len(out.runes) > maxLeadingRunes || len(out.sets) > maxLeadingSets {
		return leadingAny
	}
	return out
}
