package errors

// Attribution: portions of the below code and documentation are modeled
// directly on the https://pkg.go.dev/golang.org/x/xerrors library, used
// with the permission available under the software license
// (BSD 3-Clause):
// https://cs.opensource.google/go/x/xerrors/+/master:LICENSE
//
// Attribution: portions of the below code and documentation are modeled
// directly on the https://github.com/pkg/errors library, used
// with the permission available under the software license
// (BSD 2-Clause):
// https://github.com/pkg/errors/blob/master/LICENSE

import (
	"fmt"
	"io"
)

// Chain error wrapper.

// chain implements an error participating in a chain of errors. This is different from a list of errors by having
// each participating error having its own message, stack trace, and optionally a wrapped (causing) error.
type chain struct {
	message string
	cause   error
	frames  frames
}

var _ interface { // Assert interface implementation.
	error
	stackTracer
	framer
	Unwrap() error
	fmt.Formatter
} = (*chain)(nil)

// Chain returns a new error with its own message, annotated with a stack trace, wrapping the causing error. Chain
// errors' messages do not (usually) contain the "%w" verb, as they are meant to be printed as a whole chain (e.g. in
// a server request log).
func Chain(message string, cause error) error {
	return &chain{
		message: message,
		cause:   cause,
		frames:  getStack(3),
	}
}

func (w *chain) Error() string { return w.message }

func (w *chain) Unwrap() error { return w.cause }

// StackTrace returns the call stack frames associated with this error
// in the form of program counters; for examples of this see
// https://pkg.go.dev/runtime or
// https://pkg.go.dev/github.com/pkg/errors#Frame, both of which use the
// uintptr type to represent program counters
//
// This method only returns the frames associated with the stack trace on
// *this specific error* in the error chain. This interface is provided to
// ease interoperability with error packages or APIs that expect stack
// traces to be represented with uintptrs: Prefer the Frames method for
// general interoperability across this package.
func (w *chain) StackTrace() []uintptr { return w.frames.StackTrace() }

// Frames returns the call stack frames associated with this error.
//
// This method only returns the frames associated with the stack trace
// on *this specific error* in the error chain. Use FramesFrom to get
// all the Frames associated with an error chain.
func (w *chain) Frames() Frames { return w.frames.Frames() }

func (w *chain) Format(s fmt.State, verb rune) {
	switch verb {
	case 'v':
		if s.Flag('+') {
			_, _ = fmt.Fprintf(s, "%s%+5v", w.message, w.Frames())
			if w.cause != nil {
				_, _ = fmt.Fprintf(s, "\n\nCAUSED BY: %+5v", w.cause)
			}
			return
		}
		if s.Flag('#') {
			_, _ = fmt.Fprintf(s, "&errors.chain{%q %q}", w.message, w.cause)
			return
		}
		fallthrough
	case 's':
		_, _ = io.WriteString(s, w.Error())
	case 'q':
		_, _ = fmt.Fprintf(s, "%q", w.Error())
	default:
		// empty
	}
}
