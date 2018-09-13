package quotable

import (
	"testing"
)

type quoteTest struct {
	name   string   // descriptive name of test
	input  string   // input string to parse
	opts   *Options // options to use when parsing
	output []string // expected output
	err    string   // expected error
}

func doTest(q quoteTest, t *testing.T) {
	r, e := Split(q.input, q.opts)
	if len(r) != len(q.output) {
		t.Errorf("%s: length of results wrong (got %d, expected %d)", q.name, len(r), len(q.output))
	}
	for i, o := range q.output {
		if i >= len(r) {
			t.Errorf("%s: missing result %d (expected %q)", q.name, i, o)
			continue
		}
		if r[i] != o {
			t.Errorf("%s: unexpected result %d (got %q, expected %q)", q.name, i, r[i], o)
		}
	}
	for i := len(q.output); i < len(r); i++ {
		t.Errorf("%s: extra result %d (%q)", q.name, i, r[i])
	}
	// errors match: okay
	if q.err == "" && e == nil {
		return
	}
	if q.err == "" {
		t.Errorf("%s: unexpected error %s", q.name, e.Error())
		return
	}
	if e == nil {
		t.Errorf("%s: missing expected error %s", q.name, q.err)
		return
	}
	got := e.Error()
	if got != q.err {
		t.Errorf("%s: unexpected error, got %s, expecting %s", q.name, got, q.err)
	}
}

// Various special cases and edge cases I've thought of.
//
// There's no test for quote or space occurring inside UTF-8 strings because
// they can't; all characters after the first byte are 10xxxxxx, so they
// can't match ASCII characters.
var tests = []quoteTest{
	{name: "degenerate", input: ``, output: []string{}},
	{name: "one_word", input: `foo`, output: []string{"foo"}},
	{name: "trivial", input: `a b`, output: []string{"a", "b"}},
	{name: "tab", input: "a\tb", output: []string{"a", "b"}},
	{name: "tab_is_not_space", input: "a\tb", opts: &Options{OnlySpaceIsSpace: true}, output: []string{"a\tb"}},
	{name: "backslash_space", input: "a\\ b", output: []string{"a b"}},
	// double-backslash is just a backslash, so the space still works
	{name: "double_backslash_space", input: `a\\ b`, output: []string{"a\\", "b"}},
	{name: "quoted", input: `a" "b`, output: []string{"a b"}},
	{name: "quoted_backslashed_quote", input: `a" \"" b`, output: []string{"a \"", "b"}},
	{name: "quoted_double_backslashed_quote", input: `a" \\"" b`, output: []string{"a \\ b"}, err: "mismatched quote"},
	{name: "backslash_quote", input: `a\" b\"`, output: []string{`a"`, `b"`}},
	{name: "backslash_space_in_quote", input: `a "b\ c" d`, output: []string{"a", "b c", "d"}},
	{name: "backslash_x", opts: &Options{FancyBackslash: true}, input: `\x69`, output: []string{`i`}},
	// This behavior may not be good, but it's what I specified.
	{name: "backslash_x_invalid", opts: &Options{FancyBackslash: true}, input: `\xza`, output: []string{"\x00za"}},
	// In an early implementation, after completing a hex value the state machine reverted
	// to the backslash state, whoops.
	{name: "backslash_x_x", opts: &Options{FancyBackslash: true}, input: `\x69x23`, output: []string{`ix23`}},
	{name: "mismatched_quote", input: `"foo`, output: []string{`foo`}, err: "mismatched quote"},
	{name: "long_data", input: `a b c d e f g h i j k l m n o p q r s t u v w x y z`, output: []string{"a", "b", "c", "d", "e", "f", "g", "h", "i", "j", "k", "l", "m", "n", "o", "p", "q", "r", "s", "t", "u", "v", "w", "x", "y", "z"}},
}

func TestBasicQuoting(t *testing.T) {
	for _, q := range tests {
		doTest(q, t)
	}
}

func BenchmarkQuotable(b *testing.B) {
	b.ReportAllocs()
	t := 0
	for n := 0; n < b.N; n++ {
		for _, q := range tests {
			r, _ := Split(q.input, &Options{FancyBackslash: true})
			// this avoids "unused value" warnings
			t += len(r)
		}
	}
}
