package errors

import (
	"fmt"
	"io"
	"reflect"
	"testing"

	"github.com/secureworks/errors/internal/testutils"
)

var (
	errBasic        = New("new err")
	errSentinel     = New("sentinel err")
	errWrapSentinel = fmt.Errorf("wrap: %w", errSentinel)
	errMultiWrap    = fmt.Errorf("wrap 2: %w", fmt.Errorf("wrap 1: %w", New("err")))
	errWrappedMulti = fmt.Errorf("wrap: %w", &multierrorType{msg: "err", errs: []error{errBasic, errBasic}})

	errWithFrames error
)

func init() {
	errWithFrames = NewWithStackTrace("stack trace err")
}

type multierrorType struct {
	msg  string
	errs []error
}

func (m *multierrorType) Error() string {
	if m == nil {
		return ""
	}
	return m.msg
}

func (m *multierrorType) Unwrap() []error {
	if m == nil {
		return nil
	}
	return m.errs
}

func TestMultiError(t *testing.T) {
	// Tests below for NewMultiError, Append and Unwrap/Errors.

	t.Run("combines errors together, retaining order", func(t *testing.T) {
		err1 := New("err 1")
		err2 := New("err 2")
		err3 := New("err 3")

		merr := NewMultiError(err1, err2, err3)

		errs := merr.Unwrap()
		testutils.AssertEqual(t, 3, len(errs))
		testutils.AssertEqual(t, err1, errs[0])
		testutils.AssertEqual(t, err2, errs[1])
		testutils.AssertEqual(t, err3, errs[2])
	})

	t.Run("removes nil errors", func(t *testing.T) {
		err1 := New("err 1")
		err2 := New("err 2")
		err3 := New("err 3")

		merr := NewMultiError(nilError(), err1, nilError(), err2, err3)

		errs := merr.Unwrap()
		testutils.AssertEqual(t, 3, len(errs))
		testutils.AssertEqual(t, err1, errs[0])
		testutils.AssertEqual(t, err2, errs[1])
		testutils.AssertEqual(t, err3, errs[2])
	})

	t.Run("unwraps and flattens MultiErrors", func(t *testing.T) {
		err1 := New("err 1")
		err2 := New("err 2")
		err3 := New("err 3")
		err4 := New("err 4")

		merr1 := NewMultiError(err1, err2, err3)
		merr := NewMultiError(merr1, nilError(), err4)

		errs := merr.Unwrap()
		testutils.AssertEqual(t, 4, len(errs))
		testutils.AssertEqual(t, err1, errs[0])
		testutils.AssertEqual(t, err2, errs[1])
		testutils.AssertEqual(t, err3, errs[2])
		testutils.AssertEqual(t, err4, errs[3])
	})

	t.Run("unwraps and flattens multierrors", func(t *testing.T) {
		err1 := New("err 1")
		err2 := New("err 2")
		err3 := New("err 3")
		err4 := New("err 4")
		err5 := New("err 5")
		err6 := New("err 6")

		merr1 := &multierrorType{msg: "err", errs: []error{nilError(), err1, err2, err3}}
		merr2 := &multierrorType{msg: "err", errs: []error{err4, err5}}
		merr3 := &multierrorType{msg: "err", errs: []error{nilError(), err6}}
		merr := NewMultiError(merr1, nilError(), merr2, merr3)

		errs := merr.Unwrap()
		testutils.AssertEqual(t, 6, len(errs))
		testutils.AssertEqual(t, err1, errs[0])
		testutils.AssertEqual(t, err2, errs[1])
		testutils.AssertEqual(t, err3, errs[2])
		testutils.AssertEqual(t, err4, errs[3])
		testutils.AssertEqual(t, err5, errs[4])
		testutils.AssertEqual(t, err6, errs[5])
	})
}

func TestMultiErrorErrorOrNil(t *testing.T) {
	t.Run("returns nil when empty errors list", func(t *testing.T) {
		testutils.AssertNil(t, NewMultiError().ErrorOrNil())
	})
	t.Run("returns nil when no errors", func(t *testing.T) {
		testutils.AssertNil(t, NewMultiError(nil, nil).ErrorOrNil())
	})
	t.Run("returns an error when error", func(t *testing.T) {
		err := NewMultiError(errBasic, nil).ErrorOrNil()
		testutils.AssertNotNil(t, err)
		testutils.AssertTrue(t, reflect.TypeOf(err).Implements(reflect.TypeOf((*error)(nil)).Elem()))
	})
	t.Run("returns an error when errors", func(t *testing.T) {
		err := NewMultiError(errBasic, errBasic).ErrorOrNil()
		testutils.AssertNotNil(t, err)
		testutils.AssertTrue(t, reflect.TypeOf(err).Implements(reflect.TypeOf((*error)(nil)).Elem()))
	})
}

func TestMultiError_errors_Unwrap(t *testing.T) {
	t.Run("returns nil", func(t *testing.T) {
		merr := NewMultiError(
			errWithFrames,
			errMultiWrap,
		)
		testutils.AssertNil(t, Unwrap(merr))
	})
}

func TestMultiError_errors_As(t *testing.T) {
	err1 := customErr{msg: "err 1"}
	err2 := customErr{msg: "err 2"}
	merr := NewMultiError(
		errBasic,
		fmt.Errorf("wrap: %w", err1),
		fmt.Errorf("wrap: %w", New(newMsg)),
		err2,
	)

	t.Run("includes any wrapped error in any error item", func(t *testing.T) {
		var testErr customErr // Value type.
		testutils.AssertTrue(t, As(merr, &testErr))

		var errErr error // Interface type.
		testutils.AssertTrue(t, As(merr, &errErr))
		testutils.AssertEqual(t, "[new err; wrap: err 1; wrap: new err; err 2]", errErr.Error())
	})

	t.Run("matches the error in order", func(t *testing.T) {
		var testErr customErr
		As(merr, &testErr)
		testutils.AssertEqual(t, err1, testErr)
	})
}

func TestMultiError_errors_Is(t *testing.T) {
	errNotFound := New("err not found")
	merr := NewMultiError(
		errBasic,
		errBasic,
		errWrapSentinel,
	)

	t.Run("includes any wrapped error in any error item", func(t *testing.T) {
		cases := []struct {
			name  string
			error error
			found bool
		}{
			{"errSentinel", errSentinel, true},
			{"errNotFound", errNotFound, false},
		}
		for _, tt := range cases {
			t.Run(tt.name, func(t *testing.T) {
				testutils.AssertEqual(t, tt.found, Is(merr, tt.error))
			})
		}
	})
}

func TestMultiErrorFormat(t *testing.T) {
	t.Run("message context", func(t *testing.T) {
		merr := NewMultiError(
			errWithFrames,
			errMultiWrap,
		)
		testutils.AssertEqual(t, "[stack trace err; wrap 2: wrap 1: err]", merr.Error())

		// Order matters.
		merr = NewMultiError(
			errMultiWrap,
			errWithFrames,
		)
		testutils.AssertEqual(t, "[wrap 2: wrap 1: err; stack trace err]", merr.Error())
	})

	t.Run("formatted output", func(t *testing.T) {
		merr := NewMultiError(
			errWithFrames,
			errMultiWrap,
		)

		cases := []struct {
			format string
			error  error
			expect interface{}
		}{
			{
				format: "%s",
				error:  merr,
				expect: `^\[stack trace err; wrap 2: wrap 1: err\]$`,
			},
			{
				format: "%q",
				error:  merr,
				expect: `^"\[stack trace err; wrap 2: wrap 1: err\]"$`,
			},
			{
				format: "%v",
				error:  merr,
				expect: `^\[stack trace err; wrap 2: wrap 1: err\]$`,
			},
			{
				format: "%#v",
				error:  merr,
				expect: `^\*errors.MultiError\{stack trace err; wrap 2: wrap 1: err\}$`,
			},
			{
				format: "%d",
				error:  merr,
				expect: ``, // Empty.
			},
			{
				format: "%+v",
				error:  merr,
				expect: `multiple errors:

\* error 1 of 2: stack trace err
github\.com/secureworks/errors\.init
	.+/multierror_test.go:\d+
runtime\.doInit
	.+/runtime/proc\.go:\d+
runtime\.doInit
	.+/runtime/proc\.go:\d+
runtime\.main
	.+/runtime/proc\.go:\d+

\* error 2 of 2: wrap 2: wrap 1: err
`,
			},
		}
		for _, tt := range cases {
			t.Run(tt.format, func(t *testing.T) {
				testutils.AssertLinesMatch(t, tt.error, tt.format, tt.expect)
			})
		}
	})

	t.Run("formatted output handles empty", func(t *testing.T) {
		merr := NewMultiError()

		cases := []struct {
			format string
			expect string
		}{
			{"%s", `^\[\]$`},
			{"%q", `^"\[\]"$`},
			{"%v", `^\[\]$`},
			{"%#v", `^\*errors.MultiError\{\}$`},
			{"%d", ``},
			{"%+v", `^empty errors: \[\]$`},
		}
		for _, tt := range cases {
			t.Run(tt.format, func(t *testing.T) {
				testutils.AssertLinesMatch(t, merr, tt.format, tt.expect)
			})
		}
	})
}

// func TestErrorsFrom(t *testing.T) {
// 	cases := []struct {
// 		name   string
// 		error  error
// 		result []error
// 	}{
// 		{"nil", nil, nil},
// 		{"single error", errBasic, []error{errBasic}},
// 		{"multierror", errWrappedMulti, []error{errBasic, errBasic}},
// 		{"empty multierror", &multierrorType{}, nil},
// 		{"MultiError", NewMultiError(errBasic, errBasic), []error{errBasic, errBasic}},
// 		{"empty MultiError", NewMultiError(), nil},
// 	}
// 	for _, tt := range cases {
// 		t.Run(tt.name, func(t *testing.T) {
// 			testutils.AssertEqual(t, tt.result, ErrorsFrom(tt.error))
// 		})
// 	}
// }

func TestAppend(t *testing.T) {
	t.Run("handles nil", func(t *testing.T) {
		err1 := New("err 1")
		err3 := Append(err1, nil)

		errs := ErrorsFrom(err3)
		testutils.AssertEqual(t, 1, len(errs))
		testutils.AssertEqual(t, err1, errs[0])

		err4 := Append(nil, err1)
		errs = ErrorsFrom(err4)
		testutils.AssertEqual(t, 1, len(errs))
		testutils.AssertEqual(t, err1, errs[0])

		err5 := Append(nil, nil)
		testutils.AssertNil(t, err5)

		terrs := []error{}
		err6 := Append(terrs...)
		testutils.AssertNil(t, err6)
	})

	t.Run("merges errors", func(t *testing.T) {
		err1 := New("err 1")
		err2 := New("err 2")

		err3 := Append(err1, nil, err2)

		errs := ErrorsFrom(err3)
		testutils.AssertEqual(t, 2, len(errs))
		testutils.AssertEqual(t, err1, errs[0])
		testutils.AssertEqual(t, err2, errs[1])
	})

	t.Run("handles multierror params", func(t *testing.T) {
		err1 := New("err 1")
		err2 := New("err 2")
		merr := NewMultiError(err1, err2)
		err3 := New("err 3")

		rerr := Append(merr, err3)
		errs := ErrorsFrom(rerr)
		testutils.AssertEqual(t, 3, len(errs))
		testutils.AssertEqual(t, err1, errs[0])
		testutils.AssertEqual(t, err2, errs[1])
		testutils.AssertEqual(t, err3, errs[2])

		rerr = Append(err3, merr)
		errs = ErrorsFrom(rerr)
		testutils.AssertEqual(t, 3, len(errs))
		testutils.AssertEqual(t, err3, errs[0])
		testutils.AssertEqual(t, err1, errs[1])
		testutils.AssertEqual(t, err2, errs[2])

		merrT1 := &multierrorType{msg: "err", errs: []error{nil, err1, err2}}

		rerr = Append(merrT1, err3, merr)
		errs = ErrorsFrom(rerr)
		testutils.AssertEqual(t, 5, len(errs))
		testutils.AssertEqual(t, err1, errs[0])
		testutils.AssertEqual(t, err2, errs[1])
		testutils.AssertEqual(t, err3, errs[2])
		testutils.AssertEqual(t, err1, errs[3])
		testutils.AssertEqual(t, err2, errs[4])

	})
}

func TestAppendInto(t *testing.T) {
	t.Run("panics if first is nil", func(t *testing.T) {
		err := func() (err error) {
			defer func() {
				err = recover().(error)
			}()
			_ = AppendInto(nil, New("err"))
			return
		}()

		testutils.AssertNotNil(t, err)

		// Panic val is an error with the given message and a stack trace.
		testutils.AssertEqual(t,
			`errors.AppendInto used incorrectly: receiving pointer must not be nil`,
			err.Error())
		withTrace, ok := err.(interface{ Frames() Frames })
		testutils.AssertTrue(t, ok)
		testutils.AssertEqual(t, 4, len(withTrace.Frames()))
	})

	t.Run("merges errors; turns first param into MultiError", func(t *testing.T) {
		err1 := New("err 1")
		err1Backup := err1
		err2 := New("err 2")

		testutils.AssertTrue(t, AppendInto(&err1, err2))
		merr, ok := err1.(*MultiError)
		testutils.AssertTrue(t, ok)
		errs := merr.Unwrap()

		testutils.AssertEqual(t, 2, len(errs))
		testutils.AssertEqual(t, err1Backup, errs[0])
		testutils.AssertEqual(t, err2, errs[1])
	})

	t.Run("handles first param as MultiError", func(t *testing.T) {
		err1 := New("err 1")
		err2 := New("err 2")
		err3 := New("err 3")
		merrT1 := NewMultiError(nilError(), err1, err2)

		err := merrT1.ErrorOrNil()
		testutils.AssertTrue(t, AppendInto(&err, err3))
		merr, ok := err.(*MultiError)
		testutils.AssertTrue(t, ok)
		errs := merr.Unwrap()

		testutils.AssertEqual(t, 3, len(errs))
		testutils.AssertEqual(t, err1, errs[0])
		testutils.AssertEqual(t, err2, errs[1])
		testutils.AssertEqual(t, err3, errs[2])
	})

	t.Run("handles first param as multierror", func(t *testing.T) {
		err1 := New("err 1")
		err2 := New("err 2")
		err3 := New("err 3")
		var err error = &multierrorType{msg: "err", errs: []error{nilError(), err1, err2}}

		testutils.AssertTrue(t, AppendInto(&err, err3))
		merr, ok := err.(*MultiError)
		testutils.AssertTrue(t, ok)
		errs := merr.Unwrap()

		testutils.AssertEqual(t, 3, len(errs))
		testutils.AssertEqual(t, err1, errs[0])
		testutils.AssertEqual(t, err2, errs[1])
		testutils.AssertEqual(t, err3, errs[2])
	})

	t.Run("handles second multierror param", func(t *testing.T) {
		err1 := New("err 1")
		err2 := New("err 2")
		merr := NewMultiError(err1, err2)

		var nilErr error
		var someErr = err1

		testutils.AssertTrue(t, AppendInto(&nilErr, merr))
		merr, ok := nilErr.(*MultiError)
		testutils.AssertTrue(t, ok)
		errs := merr.Unwrap()

		testutils.AssertEqual(t, 2, len(errs))
		testutils.AssertEqual(t, err1, errs[0])
		testutils.AssertEqual(t, err2, errs[1])

		testutils.AssertTrue(t, AppendInto(&someErr, merr))
		merr, ok = someErr.(*MultiError)
		testutils.AssertTrue(t, ok)
		errs = merr.Unwrap()

		testutils.AssertEqual(t, 3, len(errs))
		testutils.AssertEqual(t, err1, errs[0])
		testutils.AssertEqual(t, err1, errs[1])
		testutils.AssertEqual(t, err2, errs[2])

		var merrT error = &multierrorType{msg: "err", errs: []error{nil, err1, err2}}
		nilErr = nil
		someErr = err1

		testutils.AssertTrue(t, AppendInto(&nilErr, merrT))
		merr, ok = nilErr.(*MultiError)
		testutils.AssertTrue(t, ok)
		errs = merr.Unwrap()

		testutils.AssertEqual(t, 2, len(errs))
		testutils.AssertEqual(t, err1, errs[0])
		testutils.AssertEqual(t, err2, errs[1])

		testutils.AssertTrue(t, AppendInto(&someErr, merrT))
		merr, ok = someErr.(*MultiError)
		testutils.AssertTrue(t, ok)
		errs = merr.Unwrap()

		testutils.AssertEqual(t, 3, len(errs))
		testutils.AssertEqual(t, err1, errs[0])
		testutils.AssertEqual(t, err1, errs[1])
		testutils.AssertEqual(t, err2, errs[2])
	})

	t.Run("handles nils", func(t *testing.T) {
		err := New("err")
		merr := NewMultiError(err)

		cases := []struct {
			name       string
			arg1       error
			arg2       error
			argWasNil  bool
			returnsNil bool
			size       int
		}{
			{"first arg nil", nil, err, false, false, 1},
			{"second arg nil", err, nil, true, false, 1},
			{"first arg multi, second arg nil", merr, nil, true, false, 1},
			{"first and second arg nil", nil, nil, true, true, 0},
		}
		for _, tt := range cases {
			t.Run(tt.name, func(t *testing.T) {
				var e = tt.arg1
				testutils.AssertEqual(t, !tt.argWasNil, AppendInto(&e, tt.arg2))
				if tt.returnsNil {
					testutils.AssertNil(t, e)
				} else {
					testutils.AssertNotNil(t, e)
				}
			})
		}
	})
}

type testCloser struct{ err error }

func (t testCloser) Close() error {
	return t.err
}

func newTestCloser(err error) io.Closer {
	return testCloser{err: err}
}

func TestAppendResult(t *testing.T) {
	// NOTE(PH): this just wraps a call to AppendInto, so most testing is
	// done there. Just test that the params are forwarded correctly below.

	var err error

	t.Run("nil appends err", func(t *testing.T) {
		err = func() (e error) {
			c := newTestCloser(errBasic)
			defer AppendResult(&e, c.Close)
			return
		}()
		testutils.AssertTrue(t, Is(err, errBasic))
		testutils.AssertEqual(t, 1, len(ErrorsFrom(err)))
	})

	t.Run("err appends nil", func(t *testing.T) {
		err = func() (e error) {
			c := newTestCloser(nil)
			e = errBasic
			defer AppendResult(&e, c.Close)
			return
		}()
		testutils.AssertTrue(t, Is(err, errBasic))
		testutils.AssertEqual(t, 1, len(ErrorsFrom(err)))
	})

	t.Run("err appends err", func(t *testing.T) {
		err = func() (e error) {
			c := newTestCloser(errSentinel)
			e = errBasic
			defer AppendResult(&e, c.Close)
			return
		}()
		testutils.AssertTrue(t, Is(err, errBasic))
		testutils.AssertTrue(t, Is(err, errSentinel))
		testutils.AssertEqual(t, 2, len(ErrorsFrom(err)))
	})

	t.Run("nil appends nil", func(t *testing.T) {
		err = func() (e error) {
			c := newTestCloser(nil)
			defer AppendResult(&e, c.Close)
			return
		}()
		testutils.AssertNil(t, err)
	})
}
