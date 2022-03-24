package errors_test

import (
	"fmt"
	"io"
	"reflect"
	"testing"

	"github.com/secureworks/errors"
	"github.com/secureworks/errors/internal/testutils"
)

var (
	errBasic        = errors.New("new err")
	errSentinel     = errors.New("sentinel err")
	errWrapSentinel = fmt.Errorf("wrap: %w", errSentinel)
	errWithFrames   = errors.NewWithStackTrace("stack trace err")
	errMultiWrap    = fmt.Errorf("wrap 2: %w", fmt.Errorf("wrap 1: %w", errors.New("err")))
	errWrappedMulti = fmt.Errorf("wrap: %w", &multiErrorType{msg: "err", errs: []error{errBasic, errBasic}})
)

func nilError() error {
	return nil
}

type multiErrorType struct {
	msg  string
	errs []error
}

func (m *multiErrorType) Error() string { return m.msg }

func (m *multiErrorType) Errors() []error { return m.errs }

func TestMultiError(t *testing.T) {
	// Tests below for NewMultiError and Errors.

	t.Run("combines errors together, retaining order", func(t *testing.T) {
		err1 := errors.New("err 1")
		err2 := errors.New("err 2")
		err3 := errors.New("err 3")

		merr := errors.NewMultiError(err1, err2, err3)

		errs := merr.Errors()
		testutils.AssertEqual(t, 3, len(errs))
		testutils.AssertEqual(t, err1, errs[0])
		testutils.AssertEqual(t, err2, errs[1])
		testutils.AssertEqual(t, err3, errs[2])
	})

	t.Run("removes nil errors", func(t *testing.T) {
		err1 := errors.New("err 1")
		err2 := errors.New("err 2")
		err3 := errors.New("err 3")

		merr := errors.NewMultiError(
			nilError(), err1, nilError(), err2, nilError(), err3, nilError())

		errs := merr.Errors()
		testutils.AssertEqual(t, 3, len(errs))
		testutils.AssertEqual(t, err1, errs[0])
		testutils.AssertEqual(t, err2, errs[1])
		testutils.AssertEqual(t, err3, errs[2])
	})

	t.Run("unwraps and flattens MultiErrors", func(t *testing.T) {
		err1 := errors.New("err 1")
		err2 := errors.New("err 2")
		err3 := errors.New("err 3")

		merr1 := errors.NewMultiError(err1, err2, err3)
		// Unwraps to the MultiError.
		merr2 := fmt.Errorf("wrap: %w", errors.NewMultiError(err1, err2, err3))
		merr := errors.NewMultiError(merr1, nilError(), merr2)

		errs := merr.Errors()
		testutils.AssertEqual(t, 6, len(errs))
		testutils.AssertEqual(t, err1, errs[0])
		testutils.AssertEqual(t, err2, errs[1])
		testutils.AssertEqual(t, err3, errs[2])
		testutils.AssertEqual(t, err1, errs[3])
		testutils.AssertEqual(t, err2, errs[4])
		testutils.AssertEqual(t, err3, errs[5])
	})

	t.Run("unwraps and flattens multiErrors", func(t *testing.T) {
		err1 := errors.New("err 1")
		err2 := errors.New("err 2")
		err3 := errors.New("err 3")

		// Unwraps to the multiError.
		// Unlike MultiErrors these can be nested.
		merrT1 := fmt.Errorf("wrap: %w", &multiErrorType{msg: "err", errs: []error{nilError(), err1, err2, err3}})
		merrT2 := fmt.Errorf("wrap: %w", &multiErrorType{msg: "err", errs: []error{err1, err2}})
		merrT3 := fmt.Errorf("wrap: %w", &multiErrorType{msg: "err", errs: []error{merrT2, nilError(), err3}})
		merr := errors.NewMultiError(merrT1, nilError(), err2, merrT3)

		errs := merr.Errors()
		testutils.AssertEqual(t, 7, len(errs))
		testutils.AssertEqual(t, err1, errs[0])
		testutils.AssertEqual(t, err2, errs[1])
		testutils.AssertEqual(t, err3, errs[2])
		testutils.AssertEqual(t, err2, errs[3])
		testutils.AssertEqual(t, err1, errs[4])
		testutils.AssertEqual(t, err2, errs[5])
		testutils.AssertEqual(t, err3, errs[6])
	})

	t.Run("retains types for flattened errors", func(t *testing.T) {
		cerr := customErr{msg: "custom err"}
		merrT := fmt.Errorf("wrap: %w", &multiErrorType{
			msg: "err",
			errs: []error{
				fmt.Errorf("wrap: %w", &multiErrorType{
					msg: "err",
					errs: []error{
						cerr,
						errWithFrames,
						errSentinel,
						errBasic,
					},
				}),
			},
		})

		errs := errors.NewMultiError(merrT).Errors()

		// Type names for value types.
		testutils.AssertEqual(t, "customErr", reflect.TypeOf(errs[0]).Name())

		// Type names for pointer types.
		testutils.AssertEqual(t, "withStackTrace", reflect.TypeOf(errs[1]).Elem().Name())
		testutils.AssertEqual(t, "errorString", reflect.TypeOf(errs[2]).Elem().Name())
		testutils.AssertEqual(t, "errorString", reflect.TypeOf(errs[3]).Elem().Name())

		// Implements.
		testutils.AssertTrue(t, reflect.TypeOf(errs[1]).Implements(stackFramer))
		testutils.AssertTrue(t, reflect.TypeOf(errs[1]).Implements(stackTracer))
		testutils.AssertTrue(t, reflect.TypeOf(errs[3]).Implements(reflect.TypeOf((*interface {
			error
		})(nil)).Elem()))
	})
}

func TestLen(t *testing.T) {
	cases := []struct {
		name string
		merr *errors.MultiError
		len  int
	}{
		{"nil", nil, 0},
		{"0", errors.NewMultiError(), 0},
		{"1", errors.NewMultiError(errBasic), 1},
		{"n", errors.NewMultiError(errBasic, errBasic, errBasic), 3},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			testutils.AssertEqual(t, tt.len, tt.merr.Len())
		})
	}
}

func TestErrorN(t *testing.T) {
	merr := errors.NewMultiError(errBasic, errBasic, errSentinel)
	cases := []struct {
		name   string
		merr   *errors.MultiError
		n      int
		expect error
	}{
		{"nil", nil, 0, nil},
		{"len 0", errors.NewMultiError(), 0, nil},
		{"negative idx", merr, -1, nil},
		{"idx overflow", merr, 3, nil},
		{"check 0", merr, 0, errBasic},
		{"check 1", merr, 1, errBasic},
		{"check 2", merr, 2, errSentinel},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			testutils.AssertEqual(t, tt.expect, tt.merr.ErrorN(tt.n))
		})
	}
}

func TestMultiErrorErrorOrNil(t *testing.T) {
	t.Run("returns nil when nil", func(t *testing.T) {
		testutils.AssertNil(t, (*errors.MultiError)(nil).ErrorOrNil())
	})

	t.Run("returns nil when nil errors list", func(t *testing.T) {
		testutils.AssertNil(t, (&errors.MultiError{}).ErrorOrNil())
	})

	t.Run("returns nil when no errors", func(t *testing.T) {
		testutils.AssertNil(t, errors.NewMultiError(nil, nil).ErrorOrNil())
	})

	t.Run("returns an error when errors", func(t *testing.T) {
		err := errors.NewMultiError(errBasic, nil).ErrorOrNil()
		testutils.AssertNotNil(t, err)
		testutils.AssertTrue(t, reflect.TypeOf(err).Implements(reflect.TypeOf((*error)(nil)).Elem()))
	})

	t.Run("Err() is an alias", func(t *testing.T) {
		testutils.AssertNil(t, (*errors.MultiError)(nil).Err())
		testutils.AssertNil(t, (&errors.MultiError{}).Err())
		testutils.AssertNil(t, errors.NewMultiError(nil, nil).Err())

		err := errors.NewMultiError(errBasic, nil).Err()
		testutils.AssertNotNil(t, err)
		testutils.AssertTrue(t, reflect.TypeOf(err).Implements(reflect.TypeOf((*error)(nil)).Elem()))
	})

	t.Run("handles nil", func(t *testing.T) {
		testutils.AssertNil(t, (*errors.MultiError)(nil).ErrorOrNil())
	})
}

func TestMultiErrorUnwrap(t *testing.T) {
	t.Run("returns nil", func(t *testing.T) {
		merr := errors.NewMultiError(
			errWithFrames,
			errMultiWrap,
		)
		testutils.AssertNil(t, errors.Unwrap(merr))
	})

	t.Run("handles nil", func(t *testing.T) {
		testutils.AssertNil(t, errors.Unwrap((*errors.MultiError)(nil)))
	})
}

func TestMultiErrorAs(t *testing.T) {
	err1 := customErr{msg: "err 1"}
	err2 := customErr{msg: "err 2"}
	merr := errors.NewMultiError(
		errBasic,
		fmt.Errorf("wrap: %w", err1),
		fmt.Errorf("wrap: %w", errors.New(newMsg)),
		err2,
	)

	t.Run("includes any wrapped error in any error item", func(t *testing.T) {
		var testErr customErr // Value type.
		testutils.AssertTrue(t, errors.As(merr, &testErr))

		var errErr error // Interface type.
		testutils.AssertTrue(t, errors.As(merr, &errErr))
		testutils.AssertEqual(t, "[new err; wrap: err 1; wrap: new err; err 2]", errErr.Error())
	})

	t.Run("matches the error in order", func(t *testing.T) {
		var testErr customErr
		errors.As(merr, &testErr)
		testutils.AssertEqual(t, err1, testErr)
	})

	t.Run("handles nil", func(t *testing.T) {
		var testErr customErr
		testutils.AssertFalse(t, errors.As((*errors.MultiError)(nil), &testErr))
	})
}

func TestMultiErrorIs(t *testing.T) {
	errNotFound := errors.New("err not found")
	merr := errors.NewMultiError(
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
				testutils.AssertEqual(t, tt.found, merr.Is(tt.error))
			})
		}
	})

	t.Run("handles nil", func(t *testing.T) {
		testutils.AssertFalse(t, errors.Is((*errors.MultiError)(nil), errBasic))
	})
}

func TestMultiErrorFormat(t *testing.T) {
	t.Run("message context", func(t *testing.T) {
		merr := errors.NewMultiError(
			errWithFrames,
			errMultiWrap,
		)
		testutils.AssertEqual(t, "[stack trace err; wrap 2: wrap 1: err]", merr.Error())

		// Order matters.
		merr = errors.NewMultiError(
			errMultiWrap,
			errWithFrames,
		)
		testutils.AssertEqual(t, "[wrap 2: wrap 1: err; stack trace err]", merr.Error())
	})

	t.Run("formatted output", func(t *testing.T) {
		merr := errors.NewMultiError(
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
github\.com/secureworks/errors_test\.init
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

	t.Run("formatted output handles nils", func(t *testing.T) {
		merr := (*errors.MultiError)(nil)

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

func TestErrorsFrom(t *testing.T) {
	cases := []struct {
		name   string
		error  error
		result []error
	}{
		{"nil", nil, nil},
		{"single error", errBasic, []error{errBasic}},
		{"multiError", errWrappedMulti, []error{errBasic, errBasic}},
		{"empty multiError", &multiErrorType{}, nil},
		{"MultiError", errors.NewMultiError(errBasic, errBasic), []error{errBasic, errBasic}},
		{"empty MultiError", errors.NewMultiError(), nil},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			testutils.AssertEqual(t, tt.result, errors.ErrorsFrom(tt.error))
		})
	}
}

func TestAppend(t *testing.T) {
	t.Run("merges errors", func(t *testing.T) {
		err1 := errors.New("err 1")
		err2 := errors.New("err 2")

		merr := errors.Append(err1, err2)

		errs := merr.Errors()
		testutils.AssertEqual(t, 2, len(errs))
		testutils.AssertEqual(t, err1, errs[0])
		testutils.AssertEqual(t, err2, errs[1])
	})

	t.Run("handles first param as MultiError", func(t *testing.T) {
		err1 := errors.New("err 1")
		err2 := errors.New("err 2")
		merrT1 := errors.NewMultiError(nilError(), err1, err2)
		err3 := errors.New("err 3")

		merr := errors.Append(merrT1, err3)

		errs := merr.Errors()
		testutils.AssertEqual(t, 3, len(errs))
		testutils.AssertEqual(t, err1, errs[0])
		testutils.AssertEqual(t, err2, errs[1])
		testutils.AssertEqual(t, err3, errs[2])
	})

	t.Run("handles first param as multiError", func(t *testing.T) {
		err1 := errors.New("err 1")
		err2 := errors.New("err 2")
		merrT1 := fmt.Errorf("wrap: %w", &multiErrorType{msg: "err", errs: []error{nilError(), err1, err2}})
		err3 := errors.New("err 3")

		merr := errors.Append(merrT1, err3)

		errs := merr.Errors()
		testutils.AssertEqual(t, 3, len(errs))
		testutils.AssertEqual(t, err1, errs[0])
		testutils.AssertEqual(t, err2, errs[1])
		testutils.AssertEqual(t, err3, errs[2])
	})

	t.Run("replaces the second error if it is a MultiError", func(t *testing.T) {
		err1 := errors.New("err 1")
		err2 := errors.New("err 2")

		merr := errors.Append(err1, errors.NewMultiError(err2))

		errs := merr.Errors()
		testutils.AssertEqual(t, 2, len(errs))
		testutils.AssertEqual(t, err1, errs[0])
		testutils.AssertNotEqual(t, err2, errs[1])
		testutils.AssertEqual(t,
			"errors.Append used incorrectly: second parameter may not be a multiError",
			errs[1].Error())
	})

	t.Run("replaces the second error if it is a multiError", func(t *testing.T) {
		err1 := errors.New("err 1")
		err2 := errors.New("err 2")

		merr := errors.Append(err1, &multiErrorType{msg: "err", errs: []error{err2}})

		errs := merr.Errors()
		testutils.AssertEqual(t, 2, len(errs))
		testutils.AssertEqual(t, err1, errs[0])
		testutils.AssertNotEqual(t, err2, errs[1])
		testutils.AssertEqual(t,
			"errors.Append used incorrectly: second parameter may not be a multiError",
			errs[1].Error())
	})

	t.Run("handles nils", func(t *testing.T) {
		err := errors.New("err")
		merr := errors.NewMultiError(err)

		cases := []struct {
			name string
			arg1 error
			arg2 error
			size int
		}{
			{"first arg nil", nil, err, 1},
			{"second arg nil", err, nil, 1},
			{"first arg multi, second arg nil", merr, nil, 1},
			{"first and second arg nil", nil, nil, 0},
		}
		for _, tt := range cases {
			t.Run(tt.name, func(t *testing.T) {
				actual := errors.Append(tt.arg1, tt.arg2)
				testutils.AssertEqual(t, tt.size, len(actual.Errors()))
			})
		}
	})
}

func TestAppendInto(t *testing.T) {
	t.Run("panics if first is nil", func(t *testing.T) {
		err := func() (err error) {
			defer func() {
				err = recover().(error)
			}()
			_ = errors.AppendInto(nil, errors.New("err"))
			return
		}()

		testutils.AssertNotNil(t, err)

		// Panic val is an error with the given message and a stack trace.
		testutils.AssertEqual(t,
			`errors.AppendInto used incorrectly: receiving pointer must not be nil`,
			err.Error())
		withTrace, ok := err.(interface{ Frames() errors.Frames })
		testutils.AssertTrue(t, ok)
		testutils.AssertEqual(t, 4, len(withTrace.Frames()))
	})

	t.Run("merges errors; turns first param into MultiError", func(t *testing.T) {
		err1 := errors.New("err 1")
		err1Backup := err1
		err2 := errors.New("err 2")

		testutils.AssertTrue(t, errors.AppendInto(&err1, err2))
		merr, ok := err1.(*errors.MultiError)
		testutils.AssertTrue(t, ok)
		errs := merr.Errors()

		testutils.AssertEqual(t, 2, len(errs))
		testutils.AssertEqual(t, err1Backup, errs[0])
		testutils.AssertEqual(t, err2, errs[1])
	})

	t.Run("handles first param as MultiError", func(t *testing.T) {
		err1 := errors.New("err 1")
		err2 := errors.New("err 2")
		err3 := errors.New("err 3")
		merrT1 := errors.NewMultiError(nilError(), err1, err2)

		err := merrT1.ErrorOrNil()
		testutils.AssertTrue(t, errors.AppendInto(&err, err3))
		merr, ok := err.(*errors.MultiError)
		testutils.AssertTrue(t, ok)
		errs := merr.Errors()

		testutils.AssertEqual(t, 3, len(errs))
		testutils.AssertEqual(t, err1, errs[0])
		testutils.AssertEqual(t, err2, errs[1])
		testutils.AssertEqual(t, err3, errs[2])
	})

	t.Run("handles first param as multiError", func(t *testing.T) {
		err1 := errors.New("err 1")
		err2 := errors.New("err 2")
		err3 := errors.New("err 3")
		err := fmt.Errorf("wrap: %w", &multiErrorType{msg: "err", errs: []error{nilError(), err1, err2}})

		testutils.AssertTrue(t, errors.AppendInto(&err, err3))
		merr, ok := err.(*errors.MultiError)
		testutils.AssertTrue(t, ok)
		errs := merr.Errors()

		testutils.AssertEqual(t, 3, len(errs))
		testutils.AssertEqual(t, err1, errs[0])
		testutils.AssertEqual(t, err2, errs[1])
		testutils.AssertEqual(t, err3, errs[2])
	})

	t.Run("replaces the second error if it is a multiError", func(t *testing.T) {
		err1 := errors.New("err 1")
		err1Backup := err1
		err2 := errors.New("err 2")

		testutils.AssertTrue(t, errors.AppendInto(&err1, &multiErrorType{msg: "err", errs: []error{err2}}))
		merr, ok := err1.(*errors.MultiError)
		testutils.AssertTrue(t, ok)
		errs := merr.Errors()

		testutils.AssertEqual(t, 2, len(errs))
		testutils.AssertEqual(t, err1Backup, errs[0])
		testutils.AssertNotEqual(t, err2, errs[1])
		testutils.AssertEqual(t,
			"errors.AppendInto used incorrectly: second parameter may not be a multiError",
			errs[1].Error())
	})

	t.Run("handles nils", func(t *testing.T) {
		err := errors.New("err")
		merr := errors.NewMultiError(err)

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
				testutils.AssertEqual(t, !tt.argWasNil, errors.AppendInto(&e, tt.arg2))
				mm, ok := e.(*errors.MultiError)
				if tt.returnsNil {
					testutils.AssertFalse(t, ok)
					testutils.AssertNil(t, e)
				} else {
					testutils.AssertTrue(t, ok)
					testutils.AssertEqual(t, tt.size, len(mm.Errors()))
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
			defer errors.AppendResult(&e, c.Close)
			return
		}()
		testutils.AssertTrue(t, errors.Is(err, errBasic))
		testutils.AssertEqual(t, 1, len(errors.ErrorsFrom(err)))
	})

	t.Run("err appends nil", func(t *testing.T) {
		err = func() (e error) {
			c := newTestCloser(nil)
			e = errBasic
			defer errors.AppendResult(&e, c.Close)
			return
		}()
		testutils.AssertTrue(t, errors.Is(err, errBasic))
		testutils.AssertEqual(t, 1, len(errors.ErrorsFrom(err)))
	})

	t.Run("err appends err", func(t *testing.T) {
		err = func() (e error) {
			c := newTestCloser(errSentinel)
			e = errBasic
			defer errors.AppendResult(&e, c.Close)
			return
		}()
		testutils.AssertTrue(t, errors.Is(err, errBasic))
		testutils.AssertTrue(t, errors.Is(err, errSentinel))
		testutils.AssertEqual(t, 2, len(errors.ErrorsFrom(err)))
	})

	t.Run("nil appends nil", func(t *testing.T) {
		err = func() (e error) {
			c := newTestCloser(nil)
			defer errors.AppendResult(&e, c.Close)
			return
		}()
		testutils.AssertNil(t, err)
	})
}
