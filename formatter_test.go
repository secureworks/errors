package errors

import (
	"errors"
	"fmt"
	"testing"

	"github.com/secureworks/errors/internal/testutils"
)

func TestErrorf_singleError(t *testing.T) {
	t.Run("wraps with message context", func(t *testing.T) {
		err := Errorf("wraps: %w", newErrorCaller())
		testutils.AssertEqual(t, `wraps: new err`, fmt.Sprint(err))
	})

	t.Run("includes frame data", func(t *testing.T) {
		err := Errorf("wraps: %w", newErrorCaller())
		ff := FramesFrom(err)
		testutils.AssertLinesMatch(t, ff, "%+v", []string{
			"",
			"^github.com/secureworks/errors\\.TestErrorf_singleError.func2$",
			"^\t.+/formatter_test\\.go:\\d+$",
		})
	})

	t.Run("handles variant params", func(t *testing.T) {
		err := Errorf("wraps: %[2]s (%[3]d): %[1]w", newErrorCaller(), "inner", 1)
		testutils.AssertErrorMessage(t, "wraps: inner (1): new err", err)
		ff := FramesFrom(err)
		testutils.AssertLinesMatch(t, ff, "%+v", []string{
			"",
			"^github.com/secureworks/errors\\.TestErrorf_singleError.func3$",
			"^\t.+/formatter_test\\.go:\\d+$",
		})
	})
}

func TestErrorf_multiError(t *testing.T) {
	errSignal := errors.New("signal")
	ctxErr := errors.New("just context")
	err := Errorf("outer: %w: %v: %w", errSignal, ctxErr, newErrorCaller())

	testutils.AssertEqual(t, `outer: signal: just context: new err`, fmt.Sprint(err))
	testutils.AssertTrue(t, Is(err, errSignal))

	errs := ErrorsFrom(err)
	testutils.AssertEqual(t, 2, len(errs))

	// Err 1
	testutils.AssertTrue(t, Is(errs[0], errSignal))
	ff := FramesFrom(errs[0])
	testutils.AssertLinesMatch(t, ff, "%+v", []string{
		"",
		"^github.com/secureworks/errors\\.TestErrorf_multiError$",
		"^\t.+/formatter_test\\.go:\\d+$",
	})

	// Err 2
	testutils.AssertEqual(t, "new err", errs[1].Error())
	ff = FramesFrom(errs[1])
	testutils.AssertLinesMatch(t, ff, "%+v", []string{
		"",
		"^github.com/secureworks/errors\\.TestErrorf_multiError$",
		"^\t.+/formatter_test\\.go:\\d+$",
	})
}

func Benchmark_parseVerb(b *testing.B) {
	for i := 0; i < b.N; i++ {
		parseVerb("%[3]*.[2]*[1]f") //nolint:errcheck
	}
}

func Test_parseVerb(t *testing.T) {
	var cases = []struct {
		in  string
		out fmtVerb
	}{
		{
			`%d`,
			fmtVerb{
				letter: 'd',
				value:  -1,
			},
		},
		{
			`%#d`,
			fmtVerb{
				letter: 'd',
				flags:  "#",
				value:  -1,
			},
		},
		{
			`%+#d`,
			fmtVerb{
				letter: 'd',
				flags:  "+#",
				value:  -1,
			},
		},
		{
			`%[2]d`,
			fmtVerb{
				letter: 'd',
				value:  2,
			},
		},
		{
			`%[3]*.[2]*[1]f`,
			fmtVerb{
				letter: 'f',
				value:  1,
				prec:   2,
				width:  3,
			},
		},
		{
			`%6.2f`,
			fmtVerb{
				letter: 'f',
				value:  -1,
			},
		},
		{
			`%#[1]x`,
			fmtVerb{
				letter: 'x',
				flags:  "#",
				value:  1,
			},
		},
		{
			"%%",
			fmtVerb{
				letter: '%',
				value:  0,
			},
		},
		{
			"%*%",
			fmtVerb{
				letter: '%',
				value:  0,
				width:  -1,
			},
		},
		{
			"%[1]%",
			fmtVerb{
				letter: '%',
				value:  0,
			},
		},
	}

	for _, tc := range cases {
		t.Run(fmt.Sprintf("parseVerb(%q)", tc.in), func(t *testing.T) {
			tc.out.raw = tc.in
			v, n, err := parseVerb(tc.in)
			if err != nil {
				t.Errorf("unexpected error %s while parsing %s", err, tc.in)
			}
			if n != len(tc.in) {
				t.Errorf("parseVerb only consumed %d of %d bytes", n, len(tc.in))
			}
			if v != tc.out {
				t.Errorf("%s parsed to %#v, want %#v", tc.in, v, tc.out)
			}
		})
	}
}

func Test_parseFormatString(t *testing.T) {
	d := func(raw string, value, idx int) fmtVerb {
		return fmtVerb{
			letter: 'd',
			value:  value,
			idx:    idx,
			raw:    raw,
		}
	}

	var cases = []struct {
		str string
		num int
		out []fmtVerb
		err error
	}{
		{
			str: `%d`,
			num: 1,
			out: []fmtVerb{d("%d", -1, 0)},
		},
		{
			str: `%d`,
			num: 5,
			out: []fmtVerb{d("%d", -1, 0)},
		},
		{
			str: `%[3]d`,
			num: 5,
			out: []fmtVerb{d("%[3]d", 3, 2)},
		},
		{
			str: `%[0]d`,
			num: 1,
			err: errors.New("invalid format string: bad argument index"),
		},
		{
			str: `%[6]d`,
			num: 5,
			err: errors.New("invalid format string: not enough arguments"),
		},
		{
			str: `%d`,
			num: 0,
			err: errors.New("invalid format string: not enough arguments"),
		},
		{
			str: `%d %d %d %d`,
			num: 4,
			out: []fmtVerb{
				d("%d", -1, 0),
				d("%d", -1, 1),
				d("%d", -1, 2),
				d("%d", -1, 3),
			},
		},
		{
			str: `%[2]d %d %d`,
			num: 4,
			out: []fmtVerb{
				d("%[2]d", 2, 1),
				d("%d", -1, 2),
				d("%d", -1, 3),
			},
		},
		{
			str: `%[2]d %d %d`,
			num: 3,
			err: errors.New("invalid format string: not enough arguments"),
		},
		{
			str: `%d %d %d %[1]d %d %d`,
			num: 3,
			out: []fmtVerb{
				d("%d", -1, 0),
				d("%d", -1, 1),
				d("%d", -1, 2),
				d("%[1]d", 1, 0),
				d("%d", -1, 1),
				d("%d", -1, 2),
			},
		},
		{
			str: `%[3]*.[2]*[1]d %d`,
			num: 3,
			out: []fmtVerb{
				{
					letter: 'd',
					width:  3,
					prec:   2,
					value:  1,
					idx:    0,
					raw:    "%[3]*.[2]*[1]d",
				},
				d("%d", -1, 1),
			},
		},
		{
			str: `%[3]*.[2]*[1]d %d`,
			num: 2,
			err: errors.New("invalid format string: not enough arguments"),
		},
	}
	for _, tc := range cases {
		t.Run(fmt.Sprintf("parseFormatString(%q,%d)", tc.str, tc.num), func(t *testing.T) {
			verbs, err := parseFormatString(tc.str, tc.num)
			if tc.err != nil {
				if err == nil {
					t.Fatalf("did not return an error")
				} else if err.Error() != tc.err.Error() {
					t.Fatalf("returned an unexpected error: %v", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("returned an unexpected error: %v", err)
				return
			}
			if len(verbs) != len(tc.out) {
				t.Fatalf("returned %d verbs, want %d", len(verbs), len(tc.out))
				return
			}
			for i, v := range verbs {
				if v != tc.out[i] {
					t.Fatalf("returned %#v, want %#v", v, tc.out[i])
				}
			}
		})
	}
}
