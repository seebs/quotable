// +build shlexcompare
// Take away the build tag, or build with that tag, to get the comparison.
//
// I'm aware that this is totally unfair because shlex does a ton of things
// quotable doesn't. That's fine. If I want a shell lexer, I know where to
// find it.

package quotable

import (
	"fmt"
	"testing"

	"github.com/google/shlex"
)

// BenchmarkShlex is a comparison benchmark against shlex.
func BenchmarkShlex(b *testing.B) {
	b.ReportAllocs()
	t := 0
	for n := 0; n < b.N; n++ {
		for _, q := range tests {
			r, _ := shlex.Split(q.input)
			// this avoids "unused value" warnings
			t += len(r)
		}
	}
}

func TestShlex(t *testing.T) {
	for _, q := range tests {
		var notes []interface{}
		note := func(format string, args ...interface{}) {
			if notes == nil {
				notes = append(notes, fmt.Sprintf("test %s:", q.name))
			}
			notes = append(notes, fmt.Sprintf(format, args...))
		}
		shlexR, _ := shlex.Split(q.input)
		quoteR, qE := Split(q.input, nil)
		l := len(shlexR)
		if len(quoteR) < l {
			l = len(quoteR)
		}
		t.Logf("test %s:", q.name)
		for i := 0; i < l; i++ {
			if shlexR[i] != quoteR[i] {
				note("  [%d]: '%s' != '%s'", i, shlexR[i], quoteR[i])
			}
		}
		for i := l; i < len(shlexR); i++ {
			note("  [%d]: shlex got '%s', quotable got nothing.", i, shlexR[i])
		}
		// don't report quote getting extra things if we reported a mismatched quote,
		// that's expected.
		if qE == nil || qE.Error() != "mismatched quote" {
			for i := l; i < len(quoteR); i++ {
				note("  [%d]: quote got '%s', shlex got nothing.", i, quoteR[i])
			}
		}
		for _, n := range notes {
			fmt.Println(n)
		}
	}
}
