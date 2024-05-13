package errors

import (
	"bytes"
	"errors"
	"fmt"
	"reflect"
	"testing"

	"github.com/secureworks/errors/internal/testutils"
)

var (
	newMsg = "new err"

	// F - 0 - F - O - F - O - Ø
	framesChainError = func() error {
		return withFrameCaller( // <-- Frame from here.
			func() error {
				return wrapCaller("1",
					func() error {
						return withFrameCaller( // <-- Frame from here.
							func() error {
								return wrapCaller("2",
									func() error {
										return withFrameCaller( // <-- Frame from here.
											func() error { return newErrorCaller() },
										)
									})
							})
					})
			})
	}

	// O - S - O - S - O - Ø
	stackChainError = func() error {
		return wrapCaller("1",
			func() error {
				return withStackTraceCaller(
					func() error {
						return wrapCaller("2",
							func() error {
								return NewWithStackTrace(newMsg) // <-- Frames from here.
							})
					})
			})
	}

	// F - O - S - O - F - O - Ø
	framesAndStackChainError = func() error {
		return withFrameCaller(
			func() error {
				return wrapCaller("1",
					func() error {
						return withStackTraceCaller( // <-- Frames from here.
							func() error {
								return wrapCaller("2",
									func() error {
										return withFrameCaller(
											func() error { return newErrorCaller() },
										)
									})
							})
					})
			})
	}
)

type errorer func() error

//go:noinline
func newErrorCaller() error {
	return New(newMsg)
}

//go:noinline
func wrapCaller(msg string, fn errorer) error {
	if msg == "" {
		msg = "wrap"
	}
	return fmt.Errorf("%s: %w", msg, fn())
}

//go:noinline
func withStackTraceCaller(fn errorer) error {
	return WithStackTrace(fn())
}

//go:noinline
func withFrameCaller(fn errorer) error {
	return WithFrame(fn())
}

//go:noinline
func withCaller(fn errorer) error {
	return fn()
}

var (
	withCallerL     = "96"
	withFrameL      = "91"
	withStackTraceL = "86"
	withWrapL       = "81"

	errorsTestPkgM  = `github\.com/secureworks/errors`
	errorsTestFilM  = `/errors_test\.go`
	withCallerFuncM = "^github\\.com/secureworks/errors.withCaller$"
	withFrameFuncM  = "^github\\.com/secureworks/errors.withFrameCaller$"
	withStackFuncM  = "^github\\.com/secureworks/errors.withStackTraceCaller$"
	withWrapFuncM   = "^github\\.com/secureworks/errors.wrapCaller$"
	errorTestAnonM  = func(fnName string) string { return fmt.Sprintf(`^%s\.glob\.\.func%s$`, errorsTestPkgM, fnName) }
	errorTestFileM  = func(line string) string { return fmt.Sprintf("^\t.+%s:%s$", errorsTestFilM, line) }

	framesChainM = []string{
		"",             // Newline.
		withFrameFuncM, // Every call to frames caller will return the same line.
		errorTestFileM(withFrameL),
		withFrameFuncM, // Called 2x.
		errorTestFileM(withFrameL),
		withFrameFuncM, // Called 3x.
		errorTestFileM(withFrameL),
	}

	stackChainM = []string{
		"", // Newline.
		errorTestAnonM("2.1.1.1"),
		errorTestFileM("43"),
		withWrapFuncM,
		errorTestFileM(withWrapL),
		errorTestAnonM("2.1.1"),
		errorTestFileM("41"),
		withStackFuncM,
		errorTestFileM(withStackTraceL),
		errorTestAnonM("2.1"),
		errorTestFileM("39"),
		withWrapFuncM,
		errorTestFileM(withWrapL),
		errorTestAnonM("2"),
		errorTestFileM("37"),
		// Append top-level caller(s) in test.
	}

	bothChainM = []string{
		"", // Newline.
		withStackFuncM,
		errorTestFileM(withStackTraceL),
		errorTestAnonM("3.1.1"),
		errorTestFileM("55"),
		withWrapFuncM,
		errorTestFileM(withWrapL),
		errorTestAnonM("3.1"),
		errorTestFileM("53"),
		withFrameFuncM,
		errorTestFileM(withFrameL),
		errorTestAnonM("3"),
		errorTestFileM("51"),
		// Append top-level caller(s) in test.
	}
)

func nilError() error {
	return nil
}

type errorType struct{}

func (e errorType) Error() string { return "i'm an error" }

var (
	stackFramerIface = reflect.TypeOf((*interface {
		Frames() Frames
	})(nil)).Elem()
	stackTracerIface = reflect.TypeOf((*interface {
		StackTrace() []uintptr
	})(nil)).Elem()
)

func TestErrorFrames(t *testing.T) {
	t.Run("Stdlib", func(t *testing.T) {
		err := New("")
		_, ok := err.(interface{ Frames() Frames })

		// Does not exist.
		testutils.AssertFalse(t, ok)
	})

	t.Run("WithFrame", func(t *testing.T) {
		err := withFrameCaller(newErrorCaller)
		withFrames, ok := err.(interface{ Frames() Frames })

		// Exists and wraps in one (current) frame.
		testutils.AssertTrue(t, ok)
		testutils.AssertEqual(t, 1, len(withFrames.Frames()))
		testutils.AssertLinesMatch(t, withFrames.Frames(), "%+v",
			[]string{
				"",
				withFrameFuncM,
				errorTestFileM(withFrameL),
			},
		)
	})

	t.Run("WithFrameAt", func(t *testing.T) {
		errorer := func(skip int) error {
			return withCaller(func() error {
				return WithFrameAt(newErrorCaller(), skip)
			})
		}

		cases := []struct {
			skip          int
			frameMatchers []string
		}{
			{
				skip: 0,
				frameMatchers: []string{
					"",
					errorsTestPkgM + `\.TestErrorFrames\.func3\.1\.1$`,
					errorTestFileM(`\d+`), // Offsets based on the anon func above.
				},
			},
			{
				skip: 1,
				frameMatchers: []string{
					"",
					withCallerFuncM,
					errorTestFileM(withCallerL),
				},
			},
			{
				skip: 2,
				frameMatchers: []string{
					"",
					errorsTestPkgM + `\.TestErrorFrames\.func3\.1$`,
					errorTestFileM(`\d+`), // Offsets based on the anon func above.
				},
			},
			{
				skip: 3,
				frameMatchers: []string{
					"",
					errorsTestPkgM + `\.TestErrorFrames\.func3\.2$`,
					errorTestFileM(`\d+`), // Offsets based on the anon func above.
				},
			},
			{
				skip: 4,
				frameMatchers: []string{
					"",
					`^testing\.tRunner$`,
					`^.+/testing/testing.go:\d+$`,
				},
			},
			{
				skip: 5, // Overflow? No problemo.
				frameMatchers: []string{
					"",
					`^unknown$`,
					`^\tunknown:0$`,
				},
			},
		}
		for _, tt := range cases {
			t.Run(fmt.Sprintf("frame %d", tt.skip), func(t *testing.T) {
				err := errorer(tt.skip)
				withFrames, ok := err.(interface{ Frames() Frames })

				testutils.AssertTrue(t, ok)
				testutils.AssertEqual(t, 1, len(withFrames.Frames()))
				testutils.AssertLinesMatch(t, withFrames.Frames(), "%+v", tt.frameMatchers)
			})
		}
	})

	t.Run("WithFrames", func(t *testing.T) {
		err := WithFrames(newErrorCaller(), Frames{
			NewFrame("github.com/secureworks/errors/errors_test.Example1", "file.go", 10),
			NewFrame("github.com/secureworks/errors/errors_test.Example2", "file.go", 20),
		})
		withFrames, ok := err.(interface{ Frames() Frames })

		// Exists and wraps in one (current) frame.
		testutils.AssertTrue(t, ok)
		testutils.AssertEqual(t, 2, len(withFrames.Frames()))
		testutils.AssertLinesMatch(t, withFrames.Frames(), "%+v",
			[]string{
				``,
				`^github.com/secureworks/errors/errors_test\.Example1$`,
				`file.go:10`,
				`^github.com/secureworks/errors/errors_test\.Example2$`,
				`file.go:20`,
			},
		)
	})

	t.Run("WithStackTrace", func(t *testing.T) {
		err := withStackTraceCaller(newErrorCaller)
		withFrames, ok := err.(interface{ Frames() Frames })

		// Exists and wraps in a stack trace starting at current frame.
		testutils.AssertTrue(t, ok)
		testutils.AssertEqual(t, 3, len(withFrames.Frames()))
		testutils.AssertLinesMatch(t, withFrames.Frames()[:1], "%+v",
			[]string{
				"",
				withStackFuncM,
				errorTestFileM(withStackTraceL),
			},
		)
	})
}

func TestErrorStackTrace(t *testing.T) {
	t.Run("Stdlib", func(t *testing.T) {
		err := New("")
		_, ok := err.(interface{ StackTrace() []uintptr })

		// Does not exist.
		testutils.AssertFalse(t, ok)
	})

	t.Run("WithFrame", func(t *testing.T) {
		err := withFrameCaller(newErrorCaller)
		_, ok := err.(interface{ StackTrace() []uintptr })

		// Does not exist.
		testutils.AssertFalse(t, ok)
	})

	t.Run("WithFrames", func(t *testing.T) {
		err := WithFrames(newErrorCaller(), Frames{
			NewFrame("github.com/secureworks/errors/errors_test.Example1", "file.go", 10),
		})
		_, ok := err.(interface{ StackTrace() []uintptr })

		// Does not exist.
		testutils.AssertFalse(t, ok)
	})

	t.Run("WithStackTrace", func(t *testing.T) {
		err := withStackTraceCaller(newErrorCaller)
		withTrace, ok := err.(interface{ StackTrace() []uintptr })

		// Exists and wraps in a stack trace starting at current frame.
		testutils.AssertTrue(t, ok)
		testutils.AssertEqual(t, 3, len(withTrace.StackTrace()))
		fr := withTrace.StackTrace()[0]
		testutils.AssertLinesMatch(t, Frames{FrameFromPC(fr)}, "%+v",
			[]string{
				"",
				withStackFuncM,
				errorTestFileM(withStackTraceL),
			},
		)
	})
}

func TestErrorf(t *testing.T) {
	t.Run("panics on bad format", func(t *testing.T) {
		err := func() (err error) {
			defer func() {
				err = recover().(error)
			}()
			_ = Errorf("does not wrap: %s", newErrorCaller())
			return
		}()

		testutils.AssertNotNil(t, err)

		// Panic val is an error with the given message and a stack trace.
		testutils.AssertEqual(t,
			`invalid use of errors.Errorf: `+
				`format string must wrap an error, but "%w" not found: `+
				`"does not wrap: %s"`, fmt.Sprint(err))
		withTrace, ok := err.(interface{ Frames() Frames })
		testutils.AssertTrue(t, ok)
		testutils.AssertEqual(t, 4, len(withTrace.Frames()))
	})

	t.Run("wraps with message context", func(t *testing.T) {
		err := Errorf("wraps: %w", newErrorCaller())
		testutils.AssertEqual(t, `wraps: new err`, fmt.Sprint(err))
	})

	t.Run("wraps with frame context", func(t *testing.T) {
		err := Errorf("wraps: %w", newErrorCaller())
		withFrames, ok := err.(interface{ Frames() Frames })
		testutils.AssertTrue(t, ok)
		testutils.AssertLinesMatch(t, withFrames.Frames(), "%+v", []string{
			"",
			"^github.com/secureworks/errors\\.TestErrorf.func3$",
			errorTestFileM(`\d+`),
		})
	})

	t.Run("handles variant params", func(t *testing.T) {
		err := Errorf("wraps: %[2]s (%[3]d): %[1]w", newErrorCaller(), "inner", 1)
		testutils.AssertErrorMessage(t, "wraps: inner (1): new err", err)
		_, ok := err.(interface{ Frames() Frames })
		testutils.AssertTrue(t, ok)
	})
}

func TestFramesFrom(t *testing.T) {
	t.Run("when none: returns empty", func(t *testing.T) {
		frames := FramesFrom(newErrorCaller())
		testutils.AssertEqual(t, 0, len(frames))
	})

	t.Run("when only frames: aggregates frames", func(t *testing.T) {
		errChain := framesChainError()
		frames := FramesFrom(errChain)
		testutils.AssertLinesMatch(t,
			frames,
			"%+v",
			framesChainM,
		)
	})

	t.Run("when only traces: returns deepest", func(t *testing.T) {
		errChain := stackChainError()
		frames := FramesFrom(errChain)
		expected := append(stackChainM, []string{
			"^github.com/secureworks/errors\\.TestFramesFrom.func3$",
			errorTestFileM(`\d+`),
			`^testing\.tRunner$`,
			`^.+/testing/testing.go:\d+$`,
		}...)

		testutils.AssertLinesMatch(t,
			frames,
			"%+v",
			expected,
		)
	})

	t.Run("when both: skips frames and uses traces", func(t *testing.T) {
		errChain := framesAndStackChainError()
		frames := FramesFrom(errChain)
		expected := append(bothChainM, []string{
			"^github.com/secureworks/errors\\.TestFramesFrom.func4$",
			errorTestFileM(`\d+`),
			`^testing\.tRunner$`,
			`^.+/testing/testing.go:\d+$`,
		}...)

		testutils.AssertLinesMatch(t,
			frames,
			"%+v",
			expected,
		)
	})
}

func TestErrorFormat(t *testing.T) {
	errChain := NewWithFrame("err")
	errChain = Errorf("wrap: %w", errChain)
	errChain = Errorf("wrap: %w", errChain)
	errStackThenFrame := WithStackTrace(errChain)

	t.Run("WithFrame", func(t *testing.T) {
		cases := []struct {
			format string
			error  error
			expect interface{}
		}{
			{"%s", withFrameCaller(newErrorCaller), `new err`},
			{"%q", withFrameCaller(newErrorCaller), `"new err"`},
			{"%v", withFrameCaller(newErrorCaller), `new err`},
			{"%#v", withFrameCaller(newErrorCaller), `&errors.withFrames{"new err"}`},
			{"%d", withFrameCaller(newErrorCaller), ``}, // empty
			{
				format: "%+v",
				error:  withFrameCaller(newErrorCaller),
				expect: []string{
					newMsg,
					withFrameFuncM,
					errorTestFileM(withFrameL),
				},
			},
			{
				// Test that subsequent withFrames do not print frames recursively.
				format: "%+v",
				error:  errChain,
				expect: []string{
					"err",
					"^github.com/secureworks/errors.TestErrorFormat$",
					errorTestFileM(`456`),
					"^github.com/secureworks/errors.TestErrorFormat$",
					errorTestFileM(`457`),
					"^github.com/secureworks/errors.TestErrorFormat$",
					errorTestFileM(`458`),
				},
			},
		}
		for _, tt := range cases {
			t.Run(tt.format, func(t *testing.T) {
				testutils.AssertLinesMatch(t, tt.error, tt.format, tt.expect)
			})
		}
	})

	t.Run("WithStack", func(t *testing.T) {
		cases := []struct {
			format string
			error  error
			expect interface{}
		}{
			{"%s", withStackTraceCaller(newErrorCaller), `new err`},
			{"%q", withStackTraceCaller(newErrorCaller), `"new err"`},
			{"%v", withStackTraceCaller(newErrorCaller), `new err`},
			{"%#v", withStackTraceCaller(newErrorCaller), `&errors.withStackTrace{"new err"}`},
			{"%d", withStackTraceCaller(newErrorCaller), ``}, // empty
			{
				format: "%+v",
				error:  withStackTraceCaller(newErrorCaller),
				expect: []string{
					newMsg,
					withStackFuncM,
					errorTestFileM(withStackTraceL),
					"^github.com/secureworks/errors\\.TestErrorFormat.func2$",
					errorTestFileM(`\d+`),
					`^testing\.tRunner$`,
					`^.+/testing/testing.go:\d+$`,
				},
			},
			{
				// Test that subsequent withFrames do not print frames recursively.
				format: "%+v",
				error:  errStackThenFrame,
				expect: []string{
					"err",
					"^github.com/secureworks/errors.TestErrorFormat$",
					errorTestFileM(`459`),
					`^testing\.tRunner$`,
					`^.+/testing/testing.go:\d+$`,
				},
			},
		}
		for _, tt := range cases {
			t.Run(tt.format, func(t *testing.T) {
				testutils.AssertLinesMatch(t, tt.error, tt.format, tt.expect)
			})
		}
	})

	t.Run("WithMessage", func(t *testing.T) {
		cases := []struct {
			format string
			error  error
			expect interface{}
		}{
			{"%s", WithMessage(newErrorCaller(), "replace err"), `replace err`},
			{"%q", WithMessage(newErrorCaller(), "replace err"), `"replace err"`},
			{"%v", WithMessage(newErrorCaller(), "replace err"), `replace err`},
			{"%#v", WithMessage(newErrorCaller(), "replace err"), `&errors.withMessage{"replace err"}`},
			{"%d", WithMessage(newErrorCaller(), "replace err"), ``}, // empty
			{
				format: "%+v",
				error:  WithMessage(newErrorCaller(), "replace err"),
				expect: `replace err`,
			},
		}
		for _, tt := range cases {
			t.Run(tt.format, func(t *testing.T) {
				testutils.AssertLinesMatch(t, tt.error, tt.format, tt.expect)
			})
		}
	})
}

func TestMask(t *testing.T) {
	t.Run("nil does nothing", func(t *testing.T) {
		testutils.AssertNil(t, Mask(nil))
	})
	t.Run("collapses wrapped errors, removing all information", func(t *testing.T) {
		signalErr := New("err1")
		err := Errorf("wrap: %w", signalErr)

		testutils.AssertEqual(t, "wrap: err1", err.Error())
		testutils.AssertTrue(t, errors.Is(err, signalErr))
		testutils.AssertTrue(t, len(FramesFrom(err)) == 1)

		err = Mask(err)
		testutils.AssertEqual(t, "wrap: err1", err.Error())
		testutils.AssertFalse(t, errors.Is(err, signalErr))
		testutils.AssertFalse(t, len(FramesFrom(err)) == 1)
	})
}

func TestOpaque(t *testing.T) {
	t.Run("nil does nothing", func(t *testing.T) {
		testutils.AssertNil(t, Opaque(nil))
	})
	t.Run("collapses wrapped errors, but retains frames", func(t *testing.T) {
		signalErr := New("err1")
		err := Errorf("wrap: %w", signalErr)

		testutils.AssertEqual(t, "wrap: err1", err.Error())
		testutils.AssertTrue(t, errors.Is(err, signalErr))
		testutils.AssertTrue(t, len(FramesFrom(err)) == 1)

		err = Opaque(err)
		testutils.AssertEqual(t, "wrap: err1", err.Error())
		testutils.AssertFalse(t, errors.Is(err, signalErr))
		testutils.AssertTrue(t, len(FramesFrom(err)) == 1)
	})
}

func TestErrorFromBytes(t *testing.T) {
	t.Run("basic errors", func(t *testing.T) {
		err := New("err")

		buf := new(bytes.Buffer)
		fmt.Fprintf(buf, "%+v", err)

		actual, ok := ErrorFromBytes(buf.Bytes())

		testutils.AssertTrue(t, ok)
		testutils.AssertEqual(t,
			fmt.Sprintf("%+v", err),
			fmt.Sprintf("%+v", actual),
		)
	})

	t.Run("errors with frames or stack traces", func(t *testing.T) {
		err := framesChainError()

		buf := new(bytes.Buffer)
		fmt.Fprintf(buf, "%+v", err)

		actual, ok := ErrorFromBytes(buf.Bytes())

		testutils.AssertTrue(t, ok)
		testutils.AssertEqual(t,
			fmt.Sprintf("%+v", err),
			fmt.Sprintf("%+v", actual),
		)
	})
}
