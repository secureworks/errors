package errors

// Attribution: portions of the below code and documentation are
// modeled directly on the https://github.com/uber-go/multierr library,
// used with the permission available under the software license (MIT):
// https://github.com/uber-go/multierr/blob/master/LICENSE.txt

import (
	"bytes"
	"fmt"
	"io"
)

// A simple interface for identifying an error wrapper for multiple
// errors (including MultiError). This is the [standard interface] as
// defined in Go.
//
// [standard interface]: https://pkg.go.dev/errors@go1.20#pkg-overview
type multierror interface {
	Unwrap() []error
}

// MultiError is a list of errors. For compatibility, this type also
// implements the standard library error interface we refer to in this
// documentation as "multierror:"
//
//	type interface {
//		Unwrap() []error
//	}
//
// MultiError includes helpers for managing groups of errors using Go
// patterns. They are not synchronized for concurrent use, but can be
// used in concurrent code if the user manages synchronization.
//
// The MultiError pattern is for top-level collection of error groups
// only, and are flattened when appended: no errors contained in its
// list are a multierror. (This is the major difference with the
// standard library implementation, which stores errors as a tree if
// they are progressively built). This flattening is not recursive,
// however: if a multierror is wrapped inside another error, it is not
// flattened, since this could cause us to lose information or context.
//
// Unlike some error collection / multiple-error packages, we rely on an
// exported MultiError type to make it obvious how it should be handled
// in the codebase. While it can be treated as an error when necessary,
// we must be vigilant about nil-checking with MultiError:
//
//	if merr := errors.NewMultiError(nil); merr != nil {
//		// This will always be true!
//	}
//
//	// Instead, check the length of the errors:
//	if merr := errors.NewMultiError(nil); len(merr.Unwrap()) > 0 {
//		// This works ...
//	}
//
//	// Or use ErrorsOrNil to get a clean error interface:
//	if merr := errors.NewMultiError(nil); merr.ErrorOrNil() != nil {
//		// This works ...
//	}
//
// For simple error-joining, use Append or AppendInto, which only speak
// in the error interface.
type MultiError struct {
	errors []error
}

var _ interface { // Assert interface implementation.
	error
	multierror
	fmt.Formatter
} = (*MultiError)(nil)

// NewMultiError returns a MultiError from a group of errors. Nil error
// values are not included, so the size of the MultiError may be less
// than the number of errors passed to the function.
//
// If any of the given errors is a multierror, it is flattened into the
// new MultiError, allowing "append-like" behavior.
func NewMultiError(errs ...error) (merr *MultiError) {
	merr = new(MultiError)
	for _, err := range errs {
		if err == nil {
			continue
		}
		if mm, ok := err.(multierror); ok {
			for _, err := range mm.Unwrap() {
				if err == nil { // Sanity check: we don't know the implementation.
					continue
				}
				merr.errors = append(merr.errors, err)
			}
			continue
		}
		merr.errors = append(merr.errors, err)
	}
	return
}

func (merr *MultiError) Error() string {
	buf := new(bytes.Buffer)
	formatMessages(buf, merr, [2]string{"[", "]"})
	return buf.String()
}

// Unwrap returns the underlying value of the MultiError: a slice of
// errors. It returns a nil slice if the error is nil or has no errors.
//
// This interface may be used to handle multierrors in code that may not
// want to expect a MultiError type directly:
//
//	if merr, ok := err.(interface{ Unwrap() [] error }); ok {
//		// ...
//	}
//
// Do not modify the returned errors and expect the MultiError to remain
// stable.
func (merr *MultiError) Unwrap() []error {
	return merr.errors
}

// Errors is the version v0.1 interface for multierrors. This pre-dated
// the release of Go 1.20, so Unwrap() []error was not a clear standard
// yet. It now is.
//
// Deprecated: use Unwrap instead.
func (merr *MultiError) Errors() []error {
	return merr.Unwrap()
}

// ErrorOrNil is used to get a clean error interface for reflection, nil
// checking and other comparisons. If the MultiError is empty it returns
// nil, and if there is a single error then it is unnested. Otherwise,
// it returns a MultiError retyped for the error interface.
//
// Retrieving the MultiError is simple, since NewMultiError flattens
// MultiErrors passed to it:
//
//	err := errors.NewMultiError(e1, e2, e3).ErrorOrNil()
//	newMErr := errors.NewMultiError(err)
//	newMErr.Errors() // => []error{e1, e2, e3}
func (merr *MultiError) ErrorOrNil() error {
	if len(merr.errors) == 0 {
		return nil
	}
	if len(merr.errors) == 1 {
		return merr.errors[0]
	}
	return merr
}

func (merr *MultiError) Format(s fmt.State, verb rune) {
	switch verb {
	case 'v':
		switch {
		case s.Flag('+'):
			size := len(merr.Unwrap())
			if size < 1 {
				io.WriteString(s, "empty errors: []")
				return
			}
			buf := new(bytes.Buffer)
			io.WriteString(s, "multiple errors:\n")
			for i, err := range merr.errors {
				if i > 0 {
					io.WriteString(s, "\n")
				}
				fmt.Fprintf(buf, "\n* error %d of %d: %+v", i+1, size, err)
				s.Write(buf.Bytes())
				buf.Reset()
			}
			io.WriteString(s, "\n")
		case s.Flag('#'):
			io.WriteString(s, "*errors.MultiError")
			formatMessages(s, merr, [2]string{"{", "}"})
		default:
			formatMessages(s, merr, [2]string{"[", "]"})
		}
	case 's':
		formatMessages(s, merr, [2]string{"[", "]"})
	case 'q':
		formatMessages(s, merr, [2]string{`"[`, `]"`})
	default:
		// empty
	}
}

func formatMessages(w io.Writer, merr multierror, delimiters [2]string) {
	first := true
	io.WriteString(w, delimiters[0])
	for _, err := range merr.Unwrap() {
		if !first {
			io.WriteString(w, "; ")
		}
		io.WriteString(w, err.Error())
		first = false
	}
	io.WriteString(w, delimiters[1])
}

// ErrorsFrom returns a list of errors that the supplied error is
// composed of. If the error is a multierror then Unwrap is called on
// it. If the given error is nil, a nil slice is returned. If the error
// is not composed of other errors, the returned slice contains just the
// error that was passed in.
//
// ErrorsFrom is useful when it is unknown if a given error is a
// multierror or not.
//
// Callers of this function are free to modify the returned slice, but
// modifications to the errors themselves may race.
func ErrorsFrom(err error) []error {
	if err == nil {
		return nil
	}
	if merr, ok := err.(multierror); ok {
		errs := merr.Unwrap()

		result := make([]error, len(errs))
		copy(result, errs)
		return result
	}
	return []error{err}
}

// Append is a version of NewMultiError optimized for the common case of
// merging a small group of errors and expecting the outcome to be an
// error or nil, akin to the standard library's errors.Join (and it is,
// in fact, used for this library's implementation of Join).
//
// The following pattern may also be used to record failure of deferred
// operations without losing information about the original error.
//
//	func doSomething(..) (err error) {
//		f := acquireResource()
//		defer func() {
//			err = errors.Append(err, f.Close())
//		}()
func Append(errs ...error) error {
	if len(errs) == 0 {
		return nil
	}

	// Optimized cases: 1 or 2 errors.
	var singleErr error
	switch {
	case len(errs) == 1:
		singleErr = errs[0]
	case len(errs) == 2 && errs[0] == nil:
		singleErr = errs[1]
	case len(errs) == 2 && errs[1] == nil:
		singleErr = errs[0]
	}
	if singleErr != nil {
		// Ensure we flatten.
		if mm, ok := singleErr.(multierror); ok {
			return NewMultiError(mm.Unwrap()...).ErrorOrNil()
		}
		return singleErr
	}

	// Do the work.
	return NewMultiError(errs...).ErrorOrNil()
}

// AppendInto appends an error into the destination of an error pointer
// and returns whether the error being appended was non-nil.
//
//	var err error
//	errors.AppendInto(&err, r.Close())
//	errors.AppendInto(&err, w.Close())
//
// The above is equivalent to,
//
//	err := errors.Append(r.Close(), w.Close())
//
// As AppendInto reports whether the provided error was non-nil, it may
// be used to build an errors error in a loop more ergonomically. For
// example:
//
//	var err error
//	for line := range lines {
//		var item Item
//		if errors.AppendInto(&err, parse(line, &item)) {
//			continue
//		}
//		items = append(items, item)
//	}
//	if err != nil {
//		log.Fatal(err)
//	}
func AppendInto(receivingErr *error, appendingErr error) bool {
	if receivingErr == nil {
		// We panic if 'into' is nil. This is not documented above
		// because suggesting that the pointer must be non-nil may
		// confuse users into thinking that the error that it points
		// to must be non-nil.
		panic(NewWithStackTrace(
			"errors.AppendInto used incorrectly: receiving pointer must not be nil"))
	}

	if appendingErr == nil {
		return false
	}
	*receivingErr = Append(*receivingErr, appendingErr)
	return true
}

// ErrorResulter is a function that may fail with an error. Use it with
// AppendResult to append the result of calling the function into an
// error. This allows you to conveniently defer capture of failing
// operations.
type ErrorResulter func() error

// AppendResult appends the result of calling the given ErrorResulter
// into the provided error pointer. Use it with named returns to safely
// defer invocation of fallible operations until a function returns, and
// capture the resulting errors.
//
//	func doSomething(...) (err error) {
//		// ...
//		f, err := openFile(..)
//		if err != nil {
//			return err
//		}
//
//		// errors will call f.Close() when this function returns, and if the
//		// operation fails it will append its error into the returned error.
//		defer errors.AppendInvoke(&err, f.Close)
//
//		scanner := bufio.NewScanner(f)
//		// Similarly, this scheduled scanner.Err to be called and inspected
//		// when the function returns and append its error into the returned
//		// error.
//		defer errors.AppendResult(&err, scanner.Err)
//
//		// ...
//	}
//
// Without defer, AppendResult behaves exactly like AppendInto.
//
//	err := // ...
//	errors.AppendResult(&err, errorableFn)
//
//	// ...is roughly equivalent to...
//
//	err := // ...
//	errors.AppendInto(&err, errorableFn())
//
// The advantage of the indirection introduced by ErrorResulter is to
// make it easy to defer the invocation of a function. Without this
// indirection, the invoked function will be evaluated at the time of
// the defer block rather than when the function returns.
//
//	// BAD: This is likely not what the caller intended. This will evaluate
//	// foo() right away and append its result into the error when the
//	// function returns.
//	defer errors.AppendInto(&err, errorableFn())
//
//	// GOOD: This will defer invocation of foo until the function returns.
//	defer errors.AppendResult(&err, errorableFn)
func AppendResult(receivingErr *error, resulterFn ErrorResulter) {
	AppendInto(receivingErr, resulterFn())
}
