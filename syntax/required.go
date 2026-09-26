package syntax

import "slices"

const maxRequiredRunes = 24

// RequiredRunes is a set of runes such that every left-to-right match
// contains at least one of them at or after its start position, so a match
// cannot start after the last one.
type RequiredRunes struct {
	ascii [2]uint64
	other []rune
}

// AnalyzeRequiredRunes returns nil when no small required set is known.
// Lookbehinds are skipped because the text they inspect may precede the search
// start.
func AnalyzeRequiredRunes(tree *RegexTree) *RequiredRunes {
	if tree == nil || tree.Root == nil || tree.Options&RightToLeft != 0 {
		return nil
	}
	set := requiredRunes(tree.Root)
	if len(set) == 0 {
		return nil
	}
	required := &RequiredRunes{}
	for _, ch := range set {
		if ch >= 0 && ch < 128 {
			required.ascii[ch>>6] |= 1 << (ch & 63)
		} else if !slices.Contains(required.other, ch) {
			required.other = append(required.other, ch)
		}
	}
	return required
}

// LastIndex returns the last index at or after start holding a required rune,
// or -1.
func (q *RequiredRunes) LastIndex(text []rune, start int) int {
	for i := len(text) - 1; i >= start; i-- {
		ch := text[i]
		if ch < 128 {
			if ch >= 0 && q.ascii[ch>>6]&(1<<(ch&63)) != 0 {
				return i
			}
		} else if len(q.other) != 0 && slices.Contains(q.other, ch) {
			return i
		}
	}
	return -1
}

func requiredRunes(n *RegexNode) []rune {
	if n.Options&(RightToLeft|IgnoreCase) != 0 {
		return nil
	}
	switch n.T {
	case NtOne:
		return []rune{n.Ch}
	case NtMulti:
		if len(n.Str) == 0 {
			return nil
		}
		return []rune{n.Str[0]}
	case NtOneloop, NtOnelazy, NtOneloopatomic:
		if n.M < 1 {
			return nil
		}
		return []rune{n.Ch}
	case NtSet:
		return enumerateSet(n.Set)
	case NtSetloop, NtSetlazy, NtSetloopatomic:
		if n.M < 1 {
			return nil
		}
		return enumerateSet(n.Set)
	case NtCapture, NtGroup, NtAtomic, NtPosLook:
		if len(n.Children) != 1 {
			return nil
		}
		return requiredRunes(n.Children[0])
	case NtLoop, NtLazyloop:
		if n.M < 1 || len(n.Children) != 1 {
			return nil
		}
		return requiredRunes(n.Children[0])
	case NtConcatenate:
		var best []rune
		for _, child := range n.Children {
			if set := requiredRunes(child); set != nil && (best == nil || len(set) < len(best)) {
				best = set
			}
		}
		return best
	case NtAlternate:
		var union []rune
		for _, child := range n.Children {
			set := requiredRunes(child)
			if set == nil {
				return nil
			}
			union = append(union, set...)
			if len(union) > maxRequiredRunes {
				return nil
			}
		}
		return union
	}
	return nil
}

func enumerateSet(set *CharSet) []rune {
	if set == nil || set.negate || set.anything || set.sub != nil || len(set.categories) != 0 {
		return nil
	}
	var out []rune
	for _, r := range set.ranges {
		if int(r.Last-r.First)+1+len(out) > maxRequiredRunes {
			return nil
		}
		for ch := r.First; ch <= r.Last; ch++ {
			out = append(out, ch)
		}
	}
	return out
}
