package errors

// Attribution: portions of the below code and documentation are modeled
// directly on the github.com/dominikh/go-tools/blob/master/printf
// package, used with the permission available under the software
// license (MIT):
// https://github.com/dominikh/go-tools/blob/master/LICENSE

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// Errorf is a shorthand for:
//
//	fmt.Errorf("some msg: %w", errors.WithFrame(err))
//
// It is made available to support the best practice of adding a call
// stack frame to the error context alongside a message when building a
// chain. When possible, prefer using the full syntax instead of this
// shorthand for clarity.
//
// Similar to fmt.Errorf, this function supports multiple `%w` verbs to
// generate a multierror: each wrapped error will have a frame attached
// to it.
func Errorf(format string, values ...interface{}) error {
	verbs, err := parseFormatString(format, len(values))
	if err != nil {
		return errors.New(`%!e(errors.Errorf=failed: ` + err.Error() + `)`)
	}

	// Interpose and wrap errors with framer if the associated verb is `%w`.
	for _, v := range verbs {
		if v.letter != 'w' {
			continue
		}
		if wrappedErr, ok := values[v.idx].(error); ok {
			if wrappedErr != nil {
				values[v.idx] = &withFrames{
					error:  wrappedErr,
					frames: frames{getFrame(3)},
				}
			}
		}
	}

	return fmt.Errorf(format, values...)
}

type fmtVerb struct {
	letter rune
	flags  string

	// Which value in the argument list the verb uses:
	//   * -1 denotes the next argument,
	//   * >0 denote explicit arguments,
	//   * 0 denotes that no argument is consumed, ie: %%. This will not be returned.
	value int

	// Similar to above: take into account argument indices used in either
	// place. When a literal will be 0.
	width, prec int

	// The 0-indexed argument this verb is associated with.
	idx int

	raw string
}

// parseFormatString parses f and returns a list of actions.
// An action may either be a literal string, or a Verb.
//
// This may break down when doing some more abstract things, like using
// argument indices with star precisions. If there is a problem, please
// don't use Errorf.
func parseFormatString(f string, numValues int) (verbs []fmtVerb, err error) {
	var nextValueIndex int
	for len(f) > 0 {
		if f[0] == '%' {
			v, n, err := parseVerb(f)
			if err != nil {
				return nil, err
			}
			f = f[n:]
			if v.value != 0 {
				if v.width > numValues {
					return nil, errors.New("invalid format string: not enough arguments")
				}
				if v.prec > numValues {
					return nil, errors.New("invalid format string: not enough arguments")
				}
				if v.value == -1 {
					v.idx = nextValueIndex
					nextValueIndex++
				} else {
					// printf argument index is one-indexed, so we can always subtract 1 here.
					v.idx = v.value - 1
					nextValueIndex = v.value
				}
				if v.idx >= numValues {
					return nil, errors.New("invalid format string: not enough arguments")
				}
				verbs = append(verbs, v)
			}
		} else {
			n := strings.IndexByte(f, '%')
			if n > -1 {
				f = f[n:]
			} else {
				f = ""
			}
		}
	}
	return verbs, nil
}

func atoi(s string) int {
	n, _ := strconv.Atoi(s)
	return n
}

// parseVerb parses the verb at the beginning of f. It returns the verb,
// how much of the input was consumed, and an error, if any.
func parseVerb(f string) (fmtVerb, int, error) {
	if len(f) < 2 {
		return fmtVerb{}, 0, errors.New("invalid format string")
	}
	const (
		flags      = 1
		widthStar  = 3
		widthIndex = 5
		dot        = 6
		precStar   = 8
		precIndex  = 10
		verbIndex  = 11
		verb       = 12
	)

	m := re.FindStringSubmatch(f)
	if m == nil {
		return fmtVerb{}, 0, errors.New("invalid format string")
	}

	v := fmtVerb{
		letter: []rune(m[verb])[0],
		flags:  m[flags],
		raw:    m[0],
	}

	if m[widthStar] != "" {
		if m[widthIndex] != "" {
			v.width = atoi(m[widthIndex])
		} else {
			v.width = -1
		}
	}

	if m[dot] != "" && m[precStar] != "" {
		if m[precIndex] != "" {
			v.prec = atoi(m[precIndex])
		} else {
			v.prec = -1
		}
	}

	if m[verb] == "%" {
		v.value = 0
	} else if m[verbIndex] != "" {
		idx := atoi(m[verbIndex])
		if idx <= 0 || idx > 128 {
			return fmtVerb{}, 0, errors.New("invalid format string: bad argument index")
		}
		v.value = idx
	} else {
		v.value = -1
	}

	return v, len(m[0]), nil
}

const (
	flags             = `([+#0 -]*)`
	verb              = `([a-zA-Z%])`
	index             = `(?:\[([0-9]+)\])`
	star              = `((` + index + `)?\*)`
	width1            = `([0-9]+)`
	width2            = star
	width             = `(?:` + width1 + `|` + width2 + `)`
	precision         = width
	widthAndPrecision = `(?:(?:` + width + `)?(?:(\.)(?:` + precision + `)?)?)`
)

var re = regexp.MustCompile(`^%` + flags + widthAndPrecision + `?` + index + `?` + verb)
