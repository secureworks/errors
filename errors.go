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
	"strings"
)

// Unwrapper is an interface implemented by errors from this package, and allows clients to unwrap the underlying error
// in the chain (if any). It is implemented regardless of the presence of any underlying error in the chain.
type Unwrapper interface {
	Unwrap() error
}

// errorImpl implements an error type that provides a message, optional causing error (next in the chain), and the
// stack-trace that led to its creation.
type errorImpl struct {
	msg    string
	error  error
	frames frames
}

var _ interface { // Assert interface implementation.
	error
	StackTracer
	Framer
	Unwrapper
	fmt.Formatter
	ChainStackTracer
	ChainFramer
} = (*errorImpl)(nil)

// New returns a new error that formats as the given text, args and optionally a wrapped error, and also captures the
// current stack trace that led to its creation. The following permutations of arguments are allowed:
//
//  1. no arguments - an empty error with a stack trace
//  2. a string message and optionally formatting arguments (which may include an error to wrap)
//  3. an error to wrap (in this case no message is presented)
//
// Note, however, that unlike fmt.Errorf, when you are wrapping another error you do NOT have to add the "%w" verb to
// the message - which allows you to hide the wrapped error's message (when this is appropriate - e.g. when presenting
// the wrapping error to a user).
//
// For example:
//
//	root := errors.New("permission 'admin.read.tenants' is missing")
//	err := errors.New("permission denied", root)
//	fmt.Println(err) // <- will print "permission denied"
//	fmt.Printf("%+v\n", err) // <- prints full error chain with stack traces of the wrapping & wrapped errors
func New(args ...interface{}) error {

	// if no args - just an empty error with a stack trace
	if len(args) == 0 {
		return &errorImpl{
			msg:    "",
			error:  nil,
			frames: getStack(3),
		}
	}

	// if first arg is an error - ensure no other args are given, and use it as the wrapped error
	if e, ok := args[0].(error); ok {
		if len(args) > 1 {
			panic("errors.New requires no additional arguments are provided when the first argument is an error")
		} else {
			return &errorImpl{
				msg:    "",
				error:  e,
				frames: getStack(3),
			}
		}
	}

	var msg string
	var wrapped error
	var wrappedIndex = -1

	// ensure first arg is a string, and use it as the message, and remove it from the list of args
	if s, ok := args[0].(string); !ok {
		panic("errors.New requires that the first argument (if provided) is either a string or an error")
	} else {
		msg = s
		args = args[1:]
	}

	// find wrapped error, if any, and ensure
	for i, e := range args {
		if e, ok := e.(error); ok {
			if wrapped != nil {
				// TODO: we can instead create a multi-error here instead! e.g. "errors.New("foo", err1, err2)"
				panic("errors.New does not support multiple error arguments")
			}
			wrapped = e
			wrappedIndex = i
		}
	}

	// If msg does not contain "%w", we are to wrap the given error, but not show it in our message
	// Therefore, we remove the wrapped error from the args, so that our call to "fmt.Sprintf" does not
	// complain about extra arguments
	// TODO: we need a more robust way to detect if user specified "%w" or not
	if strings.Contains(msg, "%w") {
		if wrapped == nil {
			panic("errors.New requires a wrapped error when using the %w verb")
		} else {
			msg = fmt.Errorf(msg, args...).Error()
		}
	} else {
		if wrapped != nil {
			args = append(args[:wrappedIndex], args[wrappedIndex+1:]...)
		}
		msg = fmt.Sprintf(msg, args...)
	}

	return &errorImpl{
		msg:    msg,
		error:  wrapped,
		frames: getStack(3),
	}
}

// Error returns this error's message.
func (w *errorImpl) Error() string { return w.msg }

// Unwrap returns the next error in the error chain, if any.
func (w *errorImpl) Unwrap() error { return w.error }

// StackTrace returns the call stack frames associated with this error
// in the form of program counters; for examples of this see
// https://pkg.go.dev/runtime or https://pkg.go.dev/github.com/pkg/errors#Frame,
// both of which use the uintptr type to represent program counters
//
// This method returns the frames associated with the stack trace on
// *this specific error* in the error chain.
//
// This interface is provided to ease interoperability with error
// packages or APIs that expect stack traces to be represented with
// uintptrs: pPrefer the Frames method for general interoperability
// across this package.
func (w *errorImpl) StackTrace() []uintptr {
	return w.frames.StackTrace()
}

// ChainStackTrace returns all call stack frames associated with this error
// chain, in the form of program counters; for examples of this see
// https://pkg.go.dev/runtime or https://pkg.go.dev/github.com/pkg/errors#Frame,
// both of which use the uintptr type to represent program counters
//
// This interface is provided to ease interoperability with error packages or APIs
// that expect stack traces to be represented with uintptrs: prefer the
// Frames method for general interoperability across this package.
func (w *errorImpl) ChainStackTrace() [][]uintptr {
	var traces [][]uintptr
	var e error = w
	for {
		if st, ok := e.(StackTracer); ok {
			traces = append(traces, st.StackTrace())
		}
		if e = Unwrap(e); e == nil {
			break
		}
	}
	return traces
}

// Frames returns the call stack frames associated with this error.
//
// This method only returns the frames associated with the stack trace
// on *this specific error* in the error chain.
func (w *errorImpl) Frames() Frames {
	return w.frames.Frames()
}

// ChainFrames returns the call stack frames associated with the entire error chain.
func (w *errorImpl) ChainFrames() []Frames {
	var chainFrames []Frames
	var e error = w
	for {
		if st, ok := e.(Framer); ok {
			chainFrames = append(chainFrames, st.Frames())
		}
		if e = Unwrap(e); e == nil {
			break
		}
	}
	return chainFrames
}

// Format allows this error to integrate into Go's formatted strings framework.
// See the package documentation for supported formats.
func (w *errorImpl) Format(s fmt.State, verb rune) {
	switch verb {
	case 'v':
		if s.Flag('+') {
			_, _ = fmt.Fprintf(s, "%s%+5v", w.Error(), w.Frames())
			if cause := w.Unwrap(); cause != nil {
				_, _ = fmt.Fprintf(s, "\n\nCAUSED BY: %+5v", cause)
			}
			return
		}
		if s.Flag('#') {
			_, _ = fmt.Fprintf(s, "&errors.errorImpl{%q %q}", w.Error(), w.error)
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

// Error masking.

// Mask returns an error with the same message context as err, but that
// does not match err and can't be unwrapped. As and Is will return
// false for all meaningful values.
func Mask(err error) error {
	if isNil(err) {
		return nil
	}
	return New(err.Error())
}

// Opaque returns an error with the same message context as err, but
// that does not match err. As and Is will return false for all
// meaningful values.
//
// If err is a chain with Frames, then those are retained as wrappers
// around the opaque error, so that the error does not lose any
// information. Otherwise, err cannot be unwrapped.
//
// You can think of Opaque as squashing the history of an error.
func Opaque(err error) error {
	if isNil(err) {
		return nil
	}

	var ff frames
	if st, ok := err.(Framer); ok {
		stack := st.Frames()
		ff := make([]*frame, len(stack))
		for i, fr := range stack {
			pc := PCFromFrame(fr)
			if pc != 0 {
				ff[i] = frameFromPC(pc)
				continue
			}
			ff[i] = newFrameFrom(fr)
		}
	} else {
		ff = getStack(3)
	}

	var cause error
	if st, ok := err.(Unwrapper); ok {
		cause = st.Unwrap()
	}

	return &errorImpl{
		msg:    err.Error(),
		error:  Opaque(cause),
		frames: ff,
	}
}
