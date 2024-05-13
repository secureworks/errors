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
	"bytes"
	"fmt"
	"io"
)

// Stack trace error wrapper.

// withStackTrace implements an error type annotated with a list of
// frames as a full stack trace.
type withStackTrace struct {
	error  error
	frames frames
}

var _ interface { // Assert interface implementation.
	error
	stackTracer
	framer
	Unwrap() error
	fmt.Formatter
} = (*withStackTrace)(nil)

// NewWithStackTrace returns a new error annotated with a stack trace.
func NewWithStackTrace(msg string) error {
	return &withStackTrace{
		error:  New(msg),
		frames: getStack(3),
	}
}

// WithStackTrace adds a stack trace to the error by wrapping it.
func WithStackTrace(err error) error {
	if err == nil {
		return nil
	}
	return &withStackTrace{
		error:  err,
		frames: getStack(3),
	}
}

func (w *withStackTrace) Error() string { return w.error.Error() }

func (w *withStackTrace) Unwrap() error { return w.error }

// StackTrace returns the call stack frames associated with this error
// in the form of program counters; for examples of this see
// https://pkg.go.dev/runtime or
// https://pkg.go.dev/github.com/pkg/errors#Frame, both of which use the
// uintptr type to represent program counters
//
// This method is only available on when the error was generated using
// WithStackTrace or NewWithStackTrace and only returns the frames
// associated with the stack trace on *this specific error* in the error
// chain. This interface is provided to ease interoperability with error
// packages or APIs that expect stack traces to be represented with
// uintptrs: pPrefer the Frames method for general interoperability
// across this package.
func (w *withStackTrace) StackTrace() []uintptr {
	return w.frames.StackTrace()
}

// Frames returns the call stack frames associated with this error.
//
// This method only returns the frames associated with the stack trace
// on *this specific error* in the error chain. Use FramesFrom to get
// all the Frames associated with an error chain.
func (w *withStackTrace) Frames() Frames {
	return w.frames.Frames()
}

func (w *withStackTrace) Format(s fmt.State, verb rune) {
	switch verb {
	case 'v':
		if s.Flag('+') {
			// NOTE: removes '+' from wrapped error formatters, to stop recursive
			// calls to FramesFrom. May have unintended consequences for errors from
			// outside libraries. Don't mix and match.
			fmt.Fprintf(s, "%v", w.error)
			FramesFrom(w).Format(s, verb)
			return
		}
		if s.Flag('#') {
			fmt.Fprintf(s, "&errors.withStackTrace{%q}", w.error)
			return
		}
		fallthrough
	case 's':
		io.WriteString(s, w.Error())
	case 'q':
		fmt.Fprintf(s, "%q", w.Error())
	default:
		// empty
	}
}

// Caller frame error wrapper.

// withFrames implements an error type annotated with list of Frames.
type withFrames struct {
	error  error
	frames frames
}

var _ interface { // Assert interface implementation.
	error
	framer
	Unwrap() error
	fmt.Formatter
} = (*withFrames)(nil)

// NewWithFrame returns a new error annotated with a call stack frame.
func NewWithFrame(msg string) error {
	return NewWithFrameAt(msg, 1)
}

// WithFrame adds a call stack frame to the error by wrapping it.
func WithFrame(err error) error {
	return WithFrameAt(err, 1)
}

// NewWithFrameAt returns a new error annotated with a call stack frame.
// The second param allows you to tune how many callers to skip (in case
// this is called in a helper you want to ignore, for example).
func NewWithFrameAt(msg string, skipCallers int) error {
	return &withFrames{
		error:  New(msg),
		frames: frames{getFrame(3 + skipCallers)},
	}
}

// WithFrameAt adds a call stack frame to the error by wrapping it. The
// second param allows you to tune how many callers to skip (in case
// this is called in a helper you want to ignore, for example).
func WithFrameAt(err error, skipCallers int) error {
	if err == nil {
		return nil
	}
	return &withFrames{
		error:  err,
		frames: frames{getFrame(3 + skipCallers)},
	}
}

// NewWithFrames returns a new error annotated with a list of frames.
func NewWithFrames(msg string, ff Frames) error {
	return WithFrames(New(msg), ff)
}

// WithFrames adds a list of frames to the error by wrapping it.
func WithFrames(err error, ff Frames) error {
	if err == nil {
		return nil
	}
	fframes := make([]*frame, len(ff))
	for i, fr := range ff {
		pc := PCFromFrame(fr)
		if pc != 0 {
			fframes[i] = frameFromPC(pc)
			continue
		}
		fframes[i] = newFrameFrom(fr)
	}
	return &withFrames{
		error:  err,
		frames: fframes,
	}
}

func (w *withFrames) Error() string { return w.error.Error() }

func (w *withFrames) Unwrap() error { return w.error }

// Frames returns the call stack frame associated with this error.
//
// This method only returns the frame on *this specific error* in the
// error chain (the result will have a length of 0 or 1). Use FramesFrom
// to get all the Frames associated with an error chain.
func (w *withFrames) Frames() Frames {
	return w.frames.Frames()
}

func (w *withFrames) Format(s fmt.State, verb rune) {
	switch verb {
	case 'v':
		if s.Flag('+') {
			// NOTE: removes '+' from wrapped error formatters, to stop recursive
			// calls to FramesFrom. May have unintended consequences for errors from
			// outside libraries. Don't mix and match.
			fmt.Fprintf(s, "%v", w.error)
			FramesFrom(w).Format(s, verb)
			return
		}
		if s.Flag('#') {
			fmt.Fprintf(s, "&errors.withFrames{%q}", w.error)
			return
		}
		fallthrough
	case 's':
		io.WriteString(s, w.Error())
	case 'q':
		fmt.Fprintf(s, "%q", w.Error())
	default:
		// empty
	}
}

// Helpers to extract data from the error interface.

// FramesFrom extracts all the Frames annotated across an error chain in
// order (if any). To do this it traverses the chain while aggregating
// frames.
//
// If this method finds any frames on an error that were added as a
// stack trace (ie, the error was wrapped by WithStackTrace) then the
// stack trace deepest in the chain is returned alone, ignoring all
// other stack traces and frames. This lets us we retain the most
// information possible without returning a confusing frame set.
// Therefore, try not to mix the WithFrame and WithStackTrace patterns
// in a single error chain.
func FramesFrom(err error) (ff Frames) {
	var traceFound bool
	for err != nil {
		var errHasTrace bool
		traceErr, ok := err.(stackTracer)
		if ok {
			traceFound = true
			errHasTrace = true
		}
		if framesErr, ok := err.(framer); ok {
			if traceFound && !errHasTrace { // Ignore frames after trace.
			} else if errHasTrace {
				ff = framesFromPCs(traceErr.StackTrace()) // Set, not append, traces.
			} else {
				ff = prependFrame(ff, framesErr.Frames()) // Prepend frames.
			}
		} else if errHasTrace { // Set, not append, traces.
			ff = framesFromPCs(traceErr.StackTrace())
		}
		err = Unwrap(err)
	}
	return
}

func prependFrame(slice Frames, frames Frames) Frames {
	slice = append(slice, frames...)
	copy(slice[len(frames):], slice)
	copy(slice, frames)
	return slice
}

// Helpers to remove context from the error interface.

// Message error wrapper.

// withMessage implements an error type annotated with a message that
// overwrites the wrapped message context.
type withMessage struct {
	error   error
	message string
}

var _ interface { // Assert interface implementation.
	error
	Unwrap() error
	fmt.Formatter
} = (*withMessage)(nil)

// WithMessage overwrites the message for the error by wrapping it. The
// error chain is maintained so that As, Is, and FramesFrom all continue
// to work.
func WithMessage(err error, msg string) error {
	if err == nil {
		return nil
	}
	return &withMessage{
		error:   err,
		message: msg,
	}
}

func (w *withMessage) Error() string { return w.message }

func (w *withMessage) Unwrap() error { return w.error }

func (w *withMessage) Format(s fmt.State, verb rune) {
	switch verb {
	case 'v':
		if s.Flag('#') {
			fmt.Fprintf(s, "&errors.withMessage{%q}", w.Error())
			return
		}
		fallthrough
	case 's':
		io.WriteString(s, w.Error())
	case 'q':
		fmt.Fprintf(s, "%q", w.Error())
	default:
		// empty
	}
}

// Error masking.

// Mask returns an error with the same message context as err, but that
// does not match err and can't be unwrapped. As and Is will return
// false for all meaningful values.
func Mask(err error) error {
	if err == nil {
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
	if err == nil {
		return nil
	}
	newErr := Mask(err)
	if fframes := FramesFrom(err); len(fframes) > 0 {
		newErr = WithFrames(newErr, fframes)
	}
	return newErr
}

// Error deserialization.

// ErrorFromBytes parses a stack trace or stack dump provided as bytes
// into an error. The format of the text is expected to match the output
// of printing with a formatter using the `%+v` verb. When an error is
// successfully parsed the second result is true; otherwise it is false.
// If you receive an error and the second result is false, well congrats
// you got an error.
//
// Currently, this only supports single errors with or without a stack
// trace or appended frames.
//
// TODO(PH): ensure ErrorFromBytes works with: multiError.
func ErrorFromBytes(byt []byte) (err error, ok bool) {
	trimbyt := bytes.TrimRight(byt, "\n")
	if len(trimbyt) == 0 || bytes.Equal(trimbyt, []byte("nil")) || bytes.Equal(trimbyt, []byte("<nil>")) {
		return nil, false
	}

	ok = true
	n := bytes.IndexByte(byt, '\n')
	if n == -1 {
		return New(string(byt)), true
	}

	err = New(string(byt[:n]))
	stack, actualErr := FramesFromBytes(byt[n+1:])
	if actualErr != nil {
		return actualErr, false
	}
	if len(stack) > 0 {
		err = WithFrames(err, stack)
	}
	return
}
