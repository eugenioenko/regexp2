package syntax

import "slices"

const maxRequiredBranchNodes = 256

// RequiredBranches retains the conjunctions inside alternations instead of
// reducing each branch to one required rune. A missing function marker can
// then reject an entire lookahead even when its leading ':' is present.
type RequiredBranches struct {
	content  *RequiredRunes
	children []*RequiredBranches
	any      bool
}

func AnalyzeRequiredBranches(tree *RegexTree) *RequiredBranches {
	if tree == nil || tree.Root == nil || tree.Options&RightToLeft != 0 || !hasBranchingLookahead(tree.Root, false) {
		return nil
	}
	remaining := maxRequiredBranchNodes
	return requiredBranches(tree.Root, &remaining)
}

func hasBranchingLookahead(n *RegexNode, lookahead bool) bool {
	if n.Options&RightToLeft != 0 || n.T == NtNegLook {
		return false
	}
	lookahead = lookahead || n.T == NtPosLook
	if lookahead && n.T == NtAlternate {
		return true
	}
	for _, child := range n.Children {
		if hasBranchingLookahead(child, lookahead) {
			return true
		}
	}
	return false
}

func requiredBranches(n *RegexNode, remaining *int) *RequiredBranches {
	if *remaining <= 0 || n.Options&(RightToLeft|IgnoreCase) != 0 {
		return nil
	}
	*remaining -= 1
	switch n.T {
	case NtCapture, NtGroup, NtAtomic, NtPosLook:
		if len(n.Children) == 1 {
			return requiredBranches(n.Children[0], remaining)
		}
	case NtLoop, NtLazyloop:
		if n.M > 0 && len(n.Children) == 1 {
			return requiredBranches(n.Children[0], remaining)
		}
	case NtConcatenate, NtAlternate:
		var children []*RequiredBranches
		for _, child := range n.Children {
			branch := requiredBranches(child, remaining)
			if branch == nil {
				if n.T == NtAlternate {
					return nil
				}
				continue
			}
			children = append(children, branch)
		}
		if len(children) == 1 {
			return children[0]
		}
		if len(children) > 1 {
			return &RequiredBranches{children: children, any: n.T == NtAlternate}
		}
	default:
		if lit := requiredLiteral(n); len(lit) > 0 {
			return &RequiredBranches{content: &RequiredRunes{literal: slices.Clone(lit[:min(len(lit), maxRequiredLiteral)])}}
		}
		if set := requiredRunes(n); len(set) > 0 {
			content := &RequiredRunes{}
			for _, ch := range set {
				if ch >= 0 && ch < 128 {
					content.ascii[ch>>6] |= 1 << (ch & 63)
				} else {
					content.other = append(content.other, ch)
				}
			}
			return &RequiredBranches{content: content}
		}
	}
	return nil
}

// LastIndex returns a conservative last possible match start, or -1 when
// required content is absent. Conjunctions take the earliest bound; alternatives
// take the latest, since any branch may succeed.
func (q *RequiredBranches) LastIndex(text []rune, start int) int {
	if q.content != nil {
		return q.content.LastIndex(text, start)
	}
	bound := len(text)
	if q.any {
		bound = -1
	}
	for _, child := range q.children {
		last := child.LastIndex(text, start)
		if q.any {
			bound = max(bound, last)
		} else {
			bound = min(bound, last)
			if bound < 0 {
				break
			}
		}
	}
	return bound
}
