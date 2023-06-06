package errors_test

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/secureworks/errors"
	"github.com/secureworks/errors/internal/testutils"
)

var (
	newMsg     = "new err"
	wrapperMsg = "wrapper"

	// F - 0 - F - O - F - O - Ø
	FramesChainError = func() error {
		return WithFrameCaller( // <-- Frame from here.
			func() error {
				return WrapCaller("1",
					func() error {
						return WithFrameCaller( // <-- Frame from here.
							func() error {
								return WrapCaller("2",
									func() error {
										return WithFrameCaller( // <-- Frame from here.
											func() error { return NewErrorCaller() },
										)
									})
							})
					})
			})
	}

	// O - S - O - S - O - Ø
	StackChainError = func() error {
		return WrapCaller("1",
			func() error {
				return WithStackTraceCaller(
					func() error {
						return WrapCaller("2",
							func() error {
								return errors.NewWithStackTrace(newMsg) // <-- Frames from here.
							})
					})
			})
	}

	// F - O - S - O - F - O - Ø
	FramesAndStackChainError = func() error {
		return WithFrameCaller(
			func() error {
				return WrapCaller("1",
					func() error {
						return WithStackTraceCaller( // <-- Frames from here.
							func() error {
								return WrapCaller("2",
									func() error {
										return WithFrameCaller(
											func() error { return NewErrorCaller() },
										)
									})
							})
					})
			})
	}
)

type Errorer func() error

//go:noinline
func NewErrorCaller() error {
	return errors.New(newMsg)
}

//go:noinline
func WrapCaller(msg string, fn Errorer) error {
	if msg == "" {
		msg = "wrap"
	}
	return fmt.Errorf("%s: %w", msg, fn())
}

//go:noinline
func WithStackTraceCaller(fn Errorer) error {
	return errors.WithStackTrace(fn())
}

//go:noinline
func WithFrameCaller(fn Errorer) error {
	return errors.WithFrame(fn())
}

//go:noinline
func WithCaller(fn Errorer) error {
	return fn()
}

func ChainCaller(msg string, fn Errorer) error {
	return errors.Chain(msg, fn())
}

var (
	withCallerL     = "96"
	withFrameL      = "91"
	withStackTraceL = "86"
	withWrapL       = "81"
	chainCallerL    = "100"

	errorsTestPkgM  = `github\.com/secureworks/errors_test`
	errorsTestFilM  = `/errors_test\.go`
	withCallerFuncM = "^github\\.com/secureworks/errors_test.WithCaller$"
	withFrameFuncM  = "^github\\.com/secureworks/errors_test.WithFrameCaller$"
	withStackFuncM  = "^github\\.com/secureworks/errors_test.WithStackTraceCaller$"
	chainFuncM      = "^github\\.com/secureworks/errors_test.ChainCaller$"
	withWrapFuncM   = "^github\\.com/secureworks/errors_test.WrapCaller$"
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

func TestErrorFrames(t *testing.T) {
	t.Run("Stdlib", func(t *testing.T) {
		err := errors.New("")
		_, ok := err.(interface{ Frames() errors.Frames })

		// Does not exist.
		testutils.AssertFalse(t, ok)
	})

	t.Run("WithFrame", func(t *testing.T) {
		err := WithFrameCaller(NewErrorCaller)
		withFrames, ok := err.(interface{ Frames() errors.Frames })

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
			return WithCaller(func() error {
				return errors.WithFrameAt(NewErrorCaller(), skip)
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
				withFrames, ok := err.(interface{ Frames() errors.Frames })

				testutils.AssertTrue(t, ok)
				testutils.AssertEqual(t, 1, len(withFrames.Frames()))
				testutils.AssertLinesMatch(t, withFrames.Frames(), "%+v", tt.frameMatchers)
			})
		}
	})

	t.Run("WithFrames", func(t *testing.T) {
		err := errors.WithFrames(NewErrorCaller(), errors.Frames{
			errors.NewFrame("github.com/secureworks/errors/errors_test.Example1", "file.go", 10),
			errors.NewFrame("github.com/secureworks/errors/errors_test.Example2", "file.go", 20),
		})
		withFrames, ok := err.(interface{ Frames() errors.Frames })

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
		err := WithStackTraceCaller(NewErrorCaller)
		withFrames, ok := err.(interface{ Frames() errors.Frames })

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

	t.Run("Chain", func(t *testing.T) {
		err := ChainCaller(wrapperMsg, NewErrorCaller)
		chain, ok := err.(interface{ Frames() errors.Frames })

		// Exists and wraps in a stack trace starting at current frame.
		testutils.AssertTrue(t, ok)
		testutils.AssertEqual(t, 3, len(chain.Frames()))
		testutils.AssertLinesMatch(t, chain.Frames()[:1], "%+v",
			[]string{
				"",
				chainFuncM,
				errorTestFileM(chainCallerL),
			},
		)
	})
}

func TestErrorStackTrace(t *testing.T) {
	t.Run("Stdlib", func(t *testing.T) {
		err := errors.New("")
		_, ok := err.(interface{ StackTrace() []uintptr })

		// Does not exist.
		testutils.AssertFalse(t, ok)
	})

	t.Run("WithFrame", func(t *testing.T) {
		err := WithFrameCaller(NewErrorCaller)
		_, ok := err.(interface{ StackTrace() []uintptr })

		// Does not exist.
		testutils.AssertFalse(t, ok)
	})

	t.Run("WithFrames", func(t *testing.T) {
		err := errors.WithFrames(NewErrorCaller(), errors.Frames{
			errors.NewFrame("github.com/secureworks/errors/errors_test.Example1", "file.go", 10),
		})
		_, ok := err.(interface{ StackTrace() []uintptr })

		// Does not exist.
		testutils.AssertFalse(t, ok)
	})

	t.Run("WithStackTrace", func(t *testing.T) {
		err := WithStackTraceCaller(NewErrorCaller)
		withTrace, ok := err.(interface{ StackTrace() []uintptr })

		// Exists and wraps in a stack trace starting at current frame.
		testutils.AssertTrue(t, ok)
		testutils.AssertEqual(t, 3, len(withTrace.StackTrace()))
		fr := withTrace.StackTrace()[0]
		testutils.AssertLinesMatch(t, errors.Frames{errors.FrameFromPC(fr)}, "%+v",
			[]string{
				"",
				withStackFuncM,
				errorTestFileM(withStackTraceL),
			},
		)
	})

	t.Run("Chain", func(t *testing.T) {
		err := ChainCaller(wrapperMsg, NewErrorCaller)
		chain, ok := err.(interface{ StackTrace() []uintptr })

		// Exists and wraps in a stack trace starting at current frame.
		testutils.AssertTrue(t, ok)
		testutils.AssertEqual(t, 3, len(chain.StackTrace()))
		fr := chain.StackTrace()[0]
		testutils.AssertLinesMatch(t, errors.Frames{errors.FrameFromPC(fr)}, "%+v",
			[]string{
				"",
				chainFuncM,
				errorTestFileM(chainCallerL),
			},
		)
	})
}

func TestNilInputs(t *testing.T) {
	t.Run("WithFrame", func(t *testing.T) {
		testutils.AssertTrue(t, errors.WithFrame(nil) == nil)
		testutils.AssertTrue(t, errors.WithFrame((*errorType)(nil)) == nil)
	})
	t.Run("WithFrameAt", func(t *testing.T) {
		testutils.AssertTrue(t, errors.WithFrameAt(nil, 4) == nil)
		testutils.AssertTrue(t, errors.WithFrameAt((*errorType)(nil), 4) == nil)
	})
	t.Run("WithFrames", func(t *testing.T) {
		ff := errors.Frames{}
		testutils.AssertTrue(t, errors.WithFrames(nil, ff) == nil)
		testutils.AssertTrue(t, errors.WithFrames((*errorType)(nil), ff) == nil)
	})
	t.Run("WithStackTrace", func(t *testing.T) {
		testutils.AssertTrue(t, errors.WithStackTrace(nil) == nil)
		testutils.AssertTrue(t, errors.WithStackTrace((*errorType)(nil)) == nil)
	})
	t.Run("Chain", func(t *testing.T) {
		testutils.AssertTrue(t, errors.Chain(wrapperMsg, nil) != nil)
		testutils.AssertTrue(t, errors.Unwrap(errors.Chain(wrapperMsg, nil)) == nil)
	})
	t.Run("WithMessage", func(t *testing.T) {
		testutils.AssertTrue(t, errors.WithMessage(nil, "new msg") == nil)
		testutils.AssertTrue(t, errors.WithMessage((*errorType)(nil), "new msg") == nil)
	})
}

func TestErrorf(t *testing.T) {
	t.Run("panics on bad format", func(t *testing.T) {
		err := func() (err error) {
			defer func() {
				err = recover().(error)
			}()
			_ = errors.Errorf("does not wrap: %s", NewErrorCaller())
			return
		}()

		testutils.AssertNotNil(t, err)

		// Panic val is an error with the given message and a stack trace.
		testutils.AssertEqual(t,
			`invalid use of errors.Errorf: `+
				`format string must wrap an error, but "%w" not found: `+
				`"does not wrap: %s"`, fmt.Sprint(err))
		withTrace, ok := err.(interface{ Frames() errors.Frames })
		testutils.AssertTrue(t, ok)
		testutils.AssertEqual(t, 4, len(withTrace.Frames()))
	})

	t.Run("wraps with message context", func(t *testing.T) {
		err := errors.Errorf("wraps: %w", NewErrorCaller())
		testutils.AssertEqual(t, `wraps: new err`, fmt.Sprint(err))
	})

	t.Run("wraps with frame context", func(t *testing.T) {
		err := errors.Errorf("wraps: %w", NewErrorCaller())
		withFrames, ok := err.(interface{ Frames() errors.Frames })
		testutils.AssertTrue(t, ok)
		testutils.AssertLinesMatch(t, withFrames.Frames(), "%+v", []string{
			"",
			"^github.com/secureworks/errors_test\\.TestErrorf.func3$",
			errorTestFileM(`\d+`),
		})
	})

	t.Run("handles variant params", func(t *testing.T) {
		err := errors.Errorf("wraps: %[2]s (%[3]d): %[1]w", NewErrorCaller(), "inner", 1)
		testutils.AssertErrorMessage(t, "wraps: inner (1): new err", err)
		_, ok := err.(interface{ Frames() errors.Frames })
		testutils.AssertTrue(t, ok)
	})
}

func TestFramesFrom(t *testing.T) {
	t.Run("when none: returns empty", func(t *testing.T) {
		frames := errors.FramesFrom(NewErrorCaller())
		testutils.AssertEqual(t, 0, len(frames))
	})

	t.Run("when only frames: aggregates frames", func(t *testing.T) {
		errChain := FramesChainError()
		frames := errors.FramesFrom(errChain)
		testutils.AssertLinesMatch(t,
			frames,
			"%+v",
			framesChainM,
		)
	})

	t.Run("when only traces: returns deepest", func(t *testing.T) {
		errChain := StackChainError()
		frames := errors.FramesFrom(errChain)
		expected := append(stackChainM, []string{
			"^github.com/secureworks/errors_test\\.TestFramesFrom.func3$",
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
		errChain := FramesAndStackChainError()
		frames := errors.FramesFrom(errChain)
		expected := append(bothChainM, []string{
			"^github.com/secureworks/errors_test\\.TestFramesFrom.func4$",
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
	errChain := errors.NewWithFrame("err")
	errChain = errors.Errorf("wrap: %w", errChain)
	errChain = errors.Errorf("wrap: %w", errChain)
	errStackThenFrame := errors.WithStackTrace(errChain)

	root := errors.NewWithStackTrace("err")
	wrapper1 := errors.Chain("wrapper1", root)
	wrapper2 := errors.Chain("wrapper2", wrapper1)

	t.Run("WithFrame", func(t *testing.T) {
		cases := []struct {
			format string
			error  error
			expect interface{}
		}{
			{"%s", WithFrameCaller(NewErrorCaller), `new err`},
			{"%q", WithFrameCaller(NewErrorCaller), `"new err"`},
			{"%v", WithFrameCaller(NewErrorCaller), `new err`},
			{"%#v", WithFrameCaller(NewErrorCaller), `&errors.withFrames{"new err"}`},
			{"%d", WithFrameCaller(NewErrorCaller), ``}, // empty
			{
				format: "%+v",
				error:  WithFrameCaller(NewErrorCaller),
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
					"^github.com/secureworks/errors_test.TestErrorFormat$",
					errorTestFileM(`506`),
					"^github.com/secureworks/errors_test.TestErrorFormat$",
					errorTestFileM(`507`),
					"^github.com/secureworks/errors_test.TestErrorFormat$",
					errorTestFileM(`508`),
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
			{"%s", WithStackTraceCaller(NewErrorCaller), `new err`},
			{"%q", WithStackTraceCaller(NewErrorCaller), `"new err"`},
			{"%v", WithStackTraceCaller(NewErrorCaller), `new err`},
			{"%#v", WithStackTraceCaller(NewErrorCaller), `&errors.withStackTrace{"new err"}`},
			{"%d", WithStackTraceCaller(NewErrorCaller), ``}, // empty
			{
				format: "%+v",
				error:  WithStackTraceCaller(NewErrorCaller),
				expect: []string{
					newMsg,
					withStackFuncM,
					errorTestFileM(withStackTraceL),
					"^github.com/secureworks/errors_test\\.TestErrorFormat.func2$",
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
					"^github.com/secureworks/errors_test.TestErrorFormat$",
					errorTestFileM(`509`),
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
			{"%s", errors.WithMessage(NewErrorCaller(), "replace err"), `replace err`},
			{"%q", errors.WithMessage(NewErrorCaller(), "replace err"), `"replace err"`},
			{"%v", errors.WithMessage(NewErrorCaller(), "replace err"), `replace err`},
			{"%#v", errors.WithMessage(NewErrorCaller(), "replace err"), `&errors.withMessage{"replace err"}`},
			{"%d", errors.WithMessage(NewErrorCaller(), "replace err"), ``}, // empty
			{
				format: "%+v",
				error:  errors.WithMessage(NewErrorCaller(), "replace err"),
				expect: `replace err`,
			},
		}
		for _, tt := range cases {
			t.Run(tt.format, func(t *testing.T) {
				testutils.AssertLinesMatch(t, tt.error, tt.format, tt.expect)
			})
		}
	})

	t.Run("Chain", func(t *testing.T) {
		cases := []struct {
			format string
			error  error
			expect interface{}
		}{
			{"%s", ChainCaller(wrapperMsg, NewErrorCaller), `wrapper`},
			{"%q", ChainCaller(wrapperMsg, NewErrorCaller), fmt.Sprintf("\"%s\"", wrapperMsg)},
			{"%v", ChainCaller(wrapperMsg, NewErrorCaller), `wrapper`},
			{"%#v", ChainCaller(wrapperMsg, NewErrorCaller), fmt.Sprintf(`&errors.chain{"%s" "%s"}`, wrapperMsg, newMsg)},
			{"%d", ChainCaller(wrapperMsg, NewErrorCaller), ``}, // empty
			{
				format: "%+v",
				error:  ChainCaller(wrapperMsg, NewErrorCaller),
				expect: []string{
					wrapperMsg,
					chainFuncM,
					errorTestFileM(chainCallerL),
					"^github.com/secureworks/errors_test\\.TestErrorFormat.func4$",
					errorTestFileM(`\d+`),
					`^testing\.tRunner$`,
					`^.+/testing/testing.go:\d+$`,
					newMsg,
				},
			},
			{
				// Test that subsequent chains DO print frames recursively.
				format: "%+v",
				error:  wrapper2,
				expect: []string{
					wrapperMsg + "2",
					"^github.com/secureworks/errors_test.TestErrorFormat$",
					errorTestFileM(`513`),
					`^testing\.tRunner$`,
					`^.+/testing/testing.go:\d+$`,
					wrapperMsg + "1",
					"^github.com/secureworks/errors_test.TestErrorFormat$",
					errorTestFileM(`512`),
					`^testing\.tRunner$`,
					`^.+/testing/testing.go:\d+$`,
					"err",
					"^github.com/secureworks/errors_test.TestErrorFormat$",
					errorTestFileM(`511`),
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
}

func TestMask(t *testing.T) {
	t.Run("nil does nothing", func(t *testing.T) {
		testutils.AssertNil(t, errors.Mask(nil))
		testutils.AssertNil(t, errors.Mask((*errorType)(nil)))
	})
	t.Run("collapses wrapped errors, removing all information", func(t *testing.T) {
		signalErr := errors.New("err1")
		err := errors.Errorf("wrap: %w", signalErr)

		testutils.AssertEqual(t, "wrap: err1", err.Error())
		testutils.AssertTrue(t, errors.Is(err, signalErr))
		testutils.AssertTrue(t, len(errors.FramesFrom(err)) == 1)

		err = errors.Mask(err)
		testutils.AssertEqual(t, "wrap: err1", err.Error())
		testutils.AssertFalse(t, errors.Is(err, signalErr))
		testutils.AssertFalse(t, len(errors.FramesFrom(err)) == 1)
	})
}

func TestOpaque(t *testing.T) {
	t.Run("nil does nothing", func(t *testing.T) {
		testutils.AssertNil(t, errors.Opaque(nil))
		testutils.AssertNil(t, errors.Opaque((*errorType)(nil)))
	})
	t.Run("collapses wrapped errors, but retains frames", func(t *testing.T) {
		signalErr := errors.New("err1")
		err := errors.Errorf("wrap: %w", signalErr)

		testutils.AssertEqual(t, "wrap: err1", err.Error())
		testutils.AssertTrue(t, errors.Is(err, signalErr))
		testutils.AssertTrue(t, len(errors.FramesFrom(err)) == 1)

		err = errors.Opaque(err)
		testutils.AssertEqual(t, "wrap: err1", err.Error())
		testutils.AssertFalse(t, errors.Is(err, signalErr))
		testutils.AssertTrue(t, len(errors.FramesFrom(err)) == 1)
	})
}

func TestErrorFromBytes(t *testing.T) {
	t.Run("basic errors", func(t *testing.T) {
		err := errors.New("err")

		buf := new(bytes.Buffer)
		fmt.Fprintf(buf, "%+v", err)

		actual, ok := errors.ErrorFromBytes(buf.Bytes())

		testutils.AssertTrue(t, ok)
		testutils.AssertEqual(t,
			fmt.Sprintf("%+v", err),
			fmt.Sprintf("%+v", actual),
		)
	})

	t.Run("errors with frames or stack traces", func(t *testing.T) {
		err := FramesChainError()

		buf := new(bytes.Buffer)
		fmt.Fprintf(buf, "%+v", err)

		actual, ok := errors.ErrorFromBytes(buf.Bytes())

		testutils.AssertTrue(t, ok)
		testutils.AssertEqual(t,
			fmt.Sprintf("%+v", err),
			fmt.Sprintf("%+v", actual),
		)
	})
}
