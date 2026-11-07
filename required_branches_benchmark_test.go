package regexp2

import (
	"fmt"
	"strings"
	"testing"
)

func BenchmarkWideTypeScriptPropertyLookahead(b *testing.B) {
	const pattern = `\w+(?=:\s*(?:Function|\([^)]*\)|[^=]*=\s*function))`
	for _, fields := range []int{10, 30, 60} {
		b.Run(fmt.Sprintf("fields_%d", fields), func(b *testing.B) {
			var line strings.Builder
			for i := range fields {
				fmt.Fprintf(&line, "'f%d': { name: 'f%d'; type: { kind: 'NON_NULL'; name: never; ofType: { kind: 'SCALAR'; name: 'String'; ofType: null; } }; }; ", i, i)
			}
			input := []rune(line.String())
			re := MustCompile(pattern)
			b.ReportAllocs()
			b.SetBytes(int64(line.Len()))
			b.ResetTimer()
			for b.Loop() {
				captures, err := re.FindRunesCaptureIndicesStartingAt(input, 0, nil)
				if err != nil {
					b.Fatal(err)
				}
				if len(captures) != 0 {
					b.Fatal("declaration fields unexpectedly matched a function property")
				}
			}
		})
	}
}
