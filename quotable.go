// Package quotable provides consistent splitting of strings into words,
// allowing quoting so returned words can contain spaces. It also supports
// backslashes, and optionally "fancy" backslashes, allowing C/Go-style
// backslash escapes to be interpolated. It does not currently support
// single-quotes, unlike Bourne shell.
package quotable

import (
	"fmt"
	"strings"
	"unicode"
)

// Options specifies optional behaviors for quoting, such as
// extended backslash handling. The zero value is the default; all options
// are framed such that "false" is the default behavior.
type Options struct {
	// Also support backslash escape sequences for unicode and special characters.
	FancyBackslash   bool
	// Only accept spaces, not arbitrary things for which unicode.IsSpace() returns true.
	OnlySpaceIsSpace bool
}

// The Error type represents an error specific to failure in the dequoting/splitting
// logic, such as mismatched quotes.
type Error string

// Error implements the error interface.
func (q Error) Error() string {
	return string(q)
}

var (
	// MismatchedQuote means that the last quoted text encountered did not have a terminating quote.
	MismatchedQuote = Error("mismatched quote")
	// IncompleteBackslash means that a backslash happened at the end of input.
	IncompleteBackslash = Error("incomplete backslash sequence")
)

type state int

const (
	normal = state(iota)
	quoted
	backslash
	hex
)

var stateFuncs []stateFunc

func init() {
	stateFuncs = []stateFunc{
		normal:    stateNormal,
		quoted:    stateQuoted,
		backslash: stateBackslash,
		hex:       stateHex,
	}
}

// isExactSpace lets us use function-pointers instead of additional ifs to
// handle space recognition.
func isExactSpace(r rune) bool {
	return r == ' '
}

// A stateFunc executes the current state, and possibly modifies the state
// func stack.
type stateFunc func(q *quoter, c rune)

type quoter struct {
	buf         strings.Builder // a buffer of suitably-dequoted characters that are actually in strings
	states      []state
	currentFunc stateFunc
	partial     bool // do we have a partial word
	isspace     func(c rune) bool
	backslash   stateFunc
	parseHex    int   // number of hex characters we want
	hexValue    rune  // used to hold the values of \x and so on by fancyBackslash
	indexes     []int // indexes of the words
	err         error
}

func (q *quoter) push(s state) {
	if len(q.states) == 0 {
		q.states = append(q.states, normal)
	}
	q.states = append(q.states, s)
	q.currentFunc = stateFuncs[s]
}

func (q *quoter) pop() {
	if len(q.states) > 1 {
		q.states = q.states[:len(q.states)-1]
		// it is intentional that this uses the now-lower length of q.states
		q.currentFunc = stateFuncs[q.states[len(q.states)-1]]
	}
}

func (q *quoter) newWord() {
	if !q.partial {
		return
	}
	q.indexes = append(q.indexes, q.buf.Len())
	q.partial = false
}

func (q *quoter) next(c rune) {
	q.currentFunc(q, c)
}

// simpleBackslash disregards any specialness of the next character, which
// means that spaces just get written into a string without breaking a word,
// quotes don't start (or end) quoted strings, and backslashes don't do
// anything special at all. it silently does nothing for anything else;
// sometimes \cigar is just cigar.
func simpleBackslash(q *quoter, c rune) {
	q.buf.WriteRune(c)
	q.pop()
}

// fancyBackslash handles C-style backslash escapes for common characters and
// allows hex encoding of characters.
func fancyBackslash(q *quoter, c rune) {
	// No matter what, the backslash processing is done. Things
	// which need further digits will then push themselves, but
	// when they're done, we go to the parent state.
	q.pop()
	switch c {
	case 'x':
		q.hexValue = 0
		q.parseHex = 2
		q.push(hex)
	case 'u':
		q.hexValue = 0
		q.parseHex = 4
		q.push(hex)
	case 'U':
		q.hexValue = 0
		q.parseHex = 8
		q.push(hex)
	// the following code was written roughly five minutes before someone mentioned
	// strconv.UnquoteChar to me.
	case 'a':
		q.buf.WriteRune('\a')
	case 'b':
		q.buf.WriteRune('\b')
	case 'f':
		q.buf.WriteRune('\f')
	case 'n':
		q.buf.WriteRune('\n')
	case 'r':
		q.buf.WriteRune('\r')
	case 't':
		q.buf.WriteRune('\t')
	case 'v':
		q.buf.WriteRune('\v')
	case '\\':
		q.buf.WriteRune('\\')
	case '"':
		q.buf.WriteRune('"')
	case '\'':
		q.buf.WriteRune('\'')
	default:
		q.err = Error(fmt.Sprintf("invalid backslash escape character '%c'", c))
		// but write it anyway
		q.buf.WriteRune(c)
	}
}

var hexDigits = [128]int8{
	-1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, /* 0x00-0x10 */
	-1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, /* 0x10-0x1f */
	-1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, /* 0x20-0x2f */
	0, 1, 2, 3, 4, 5, 6, 7, 8, 9, -1, -1, -1, -1, -1, -1, /* 0x30-0x3f */
	-1, 10, 11, 12, 13, 14, 15, -1, -1, -1, -1, -1, -1, -1, -1, -1, /* 0x40-0x4f */
	-1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, /* 0x50-0x5f */
	-1, 10, 11, 12, 13, 14, 15, -1, -1, -1, -1, -1, -1, -1, -1, -1, /* 0x60-0x6f */
	-1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, /* 0x70-0x7f */
}

// stateHex handles hexadecimal inputs, accepting up to q.parseHex
// digits (used to handle \x, \u, and \U with 2/4/8).
func stateHex(q *quoter, c rune) {
	if c < 128 {
		val := hexDigits[c]
		if val != -1 {
			q.hexValue = q.hexValue*16 + rune(val)
			q.parseHex--
			if q.parseHex == 0 {
				q.buf.WriteRune(q.hexValue)
				q.pop()
				return
			}
			// got a hex digit, still expecting more, continue
			return
		}
	}
	// write whatever hex value we got, even if we didn't get one, in which
	// case it's zero
	q.buf.WriteRune(q.hexValue)
	q.pop()
	// and hand the character we couldn't handle back to the previous state func
	q.currentFunc(q, c)
}

func stateNormal(q *quoter, c rune) {
	switch {
	case q.isspace(c):
		q.newWord()
		return
	case c == '\\':
		q.push(backslash)
	case c == '"':
		q.push(quoted)
	default:
		q.buf.WriteRune(c)
	}
	// anything other than a space will cause us to count as starting
	// a word.
	q.partial = true
}

func stateBackslash(q *quoter, c rune) {
	q.backslash(q, c)
}

func stateQuoted(q *quoter, c rune) {
	switch {
	case c == '\\':
		q.push(backslash)
	case c == '"':
		q.pop()
	default:
		q.buf.WriteRune(c)
	}
}

// Split splits the given string into words, with behavior controlled
// by the provided Options. If `q` is nil, it's treated like a zero
// valued Options.
//
// Split attempts to return meaningful values even if an error is
// encountered. For instance, if a line has an unmatched quote, the
// returned slice of strings still has the end of the line as the
// last word, but the returned error is non-nil.
func Split(s string, qopt *Options) (results []string, err error) {
	var opt Options
	if qopt != nil {
		opt = *qopt
	}
	var q quoter
	q.states = append(q.states, normal)
	q.currentFunc = stateFuncs[normal]
	if opt.OnlySpaceIsSpace {
		q.isspace = isExactSpace
	} else {
		q.isspace = unicode.IsSpace
	}
	if opt.FancyBackslash {
		q.backslash = fancyBackslash
	} else {
		q.backslash = simpleBackslash
	}

	for _, c := range s {
		q.next(c)
	}
	switch q.states[len(q.states)-1] {
	case quoted:
		q.err = MismatchedQuote
	case backslash:
		q.buf.WriteRune('\\')
		q.err = IncompleteBackslash
	case hex:
	}

	q.newWord()
	bufStr := q.buf.String()
	words := make([]string, len(q.indexes))
	prev := 0
	for idx, next := range q.indexes {
		words[idx] = bufStr[prev:next]
		prev = next
	}
	return words, q.err
}
