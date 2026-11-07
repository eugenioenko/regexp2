package syntax

import "math"

// A greedy unbounded loop compiles to a Branchmark. After a nonempty iteration
// (or on entry to a {0,} loop) the only state its continuation reads, up to the
// end of the enclosing lookaround or atomic group, is the text position, as
// long as:
//
//   - no opcode reads captures (backreferences, (?(n)...), balancing groups),
//   - no opcode reads the search origin (\G),
//   - no enclosing loop inside the same region reads its iteration mark or
//     count when this loop exits, and
//   - the loop does not run right to left.
//
// When both continuations of such a state fail, every later arrival at the
// same position fails too, so the runner can record and skip it. A region that
// succeeds discards its backtracking entries (Forejump/Backjump), so a state is
// only recorded when its continuation could not reach the region's end.

func treeAllowsBranchmarkMemo(tree *RegexTree) bool {
	if tree == nil || tree.Root == nil || tree.Options&RightToLeft != 0 {
		return false
	}
	return !readsCapturesOrOrigin(tree.Root)
}

func readsCapturesOrOrigin(n *RegexNode) bool {
	switch n.T {
	case NtRef, NtBackRefCond, NtStart:
		return true
	case NtCapture:
		if n.N != -1 {
			return true
		}
	}
	for _, child := range n.Children {
		if readsCapturesOrOrigin(child) {
			return true
		}
	}
	return false
}

func branchmarkMemoEligible(loop *RegexNode) bool {
	if loop.T != NtLoop || loop.M > 1 || loop.N != math.MaxInt32 || loop.Options&RightToLeft != 0 {
		return false
	}
	for n := loop.Parent; n != nil; n = n.Parent {
		if n.Options&RightToLeft != 0 {
			return false
		}
		switch n.T {
		case NtPosLook, NtNegLook, NtAtomic:
			return true
		case NtLoop, NtLazyloop:
			return false
		}
	}
	return true
}
