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

// MultiError is a list of errors. For compatibility, this type also
// implements the standard library error interfaces (including Unwrap,
// and the unexported interfaces for As, and Is) and includes helpers
// for managing groups of errors using Go patterns.
//
// MultiErrors are guaranteed to be flat: no errors contained in its
// list are (or wrap) a MultiError. The MultiError pattern is for
// top-level collection of error groups only. MultiErrors may not
// themselves (since they implement the error interface) wrap another
// error, so Unwrap always returns nil.
//
// MultiErrors are not synchronized: you must handle them in a
// concurrency safe way when accessing from multiple goroutines.
//
// Unlike some error collection / multiple-error packages, we rely on an
// exported MultiError type make it obvious how they should be handled
// in the codebase. They can be treated as errors if necessary, but
// usually we want to explicitly handle a multiple-error scenario.
//
// However, any package function that expects a multiple error
// implementation relies on the unexported interface:
//
//	type multiError interface {
//	    Errors() []error
//	}
//
// This is for simplicity and interoperability: you can still extract
// a multiple error from any error using NewMultiError.
type MultiError struct {
	errors []error
}

// A simple interface for identifying an error wrapper for multiple
// errors (including MultiError).
type multiError interface {
	Errors() []error
}

var _ interface { // Assert interface implementation.
	error
	multiError
	Unwrap() error
	As(interface{}) bool
	Is(error) bool
	fmt.Formatter
} = (*MultiError)(nil)

// NewMultiError returns a MultiError from a group of errors. Nil error
// values are not included, so the size of the MultiError may be less
// than the number of errors passed to the function.
//
// If any of the given errors is a MultiError, it is flattened into the
// new MultiError.
//
// If any of the errors is not itself a MultiError, but wraps a
// MultiError, then that MultiError is unwrapped and each of its errors
// is flattened into the new MultiError. In this way we could lose
// information about an error chain, so the simple rule is
// ***do not wrap MultiErrors!***
func NewMultiError(errors ...error) (merr *MultiError) {
	for _, err := range errors {
		if isNil(err) {
			continue
		}
		if merr == nil {
			merr = &MultiError{}
		}
		if mm := unwrapMultiErr(err); mm != nil {
			merr.errors = append(merr.errors, flatten(mm)...)
		} else {
			merr.errors = append(merr.errors, err)
		}
	}
	return
}

// TODO(PH): are there ways to optimize allocations below (and above)?

// flatten gets a list of errors from a multiError that is certain not
// to contain any other multiErrors or wrapped multiErrors.
func flatten(m multiError) (errs []error) {
	// We can skip a deep unwrap/flatten pass on a MultiError.
	if merr, ok := m.(*MultiError); ok {
		return merr.Errors()
	}

	for _, err := range m.Errors() {
		if isNil(err) {
			continue
		}
		if mm := unwrapMultiErr(err); mm != nil {
			errs = append(errs, flatten(mm)...)
		} else {
			errs = append(errs, err)
		}
	}
	return
}

// unwrapMultiErr finds the first multiError in the error chain. If none
// is found it returns nil.
func unwrapMultiErr(err error) multiError {
	merr := new(multiError)
	if As(err, merr) {
		return *merr
	}
	return nil
}

func (merr *MultiError) Error() string {
	// TODO(PH): prealloc and possibly use strings.Builder; eg:
	//     var buf strings.Builder
	//     buf.Grow(merr.Len() * 128 + 2)
	buf := new(bytes.Buffer)
	formatMessages(buf, merr, [2]string{"[", "]"})
	return buf.String()
}

// Errors returns the underlying value of the MultiError: a slice of
// errors. It is how we extract the underlying errors. Returns a nil
// slice if the error is nil or has no errors.
//
// This interface may be used to treat MultiErrors as an interface for
// use in code that may not want to expect a MultiError type directly:
//
//	if merr, ok := err.(interface{ Errors() [] error }); ok {
//	    // ...
//	}
//
// Do not modify the returned errors and expect the MultiError to remain
// stable.
func (merr *MultiError) Errors() []error {
	if isNil(merr) || len(merr.errors) == 0 {
		return nil
	}
	return merr.errors
}

// Len returns the number of errors currently in the MultiError.
func (merr *MultiError) Len() int {
	if isNil(merr) {
		return 0
	}
	return len(merr.errors)
}

// ErrorN returns the error at the given index in the MultiError. If
// this index does not exist then we return nil.
func (merr *MultiError) ErrorN(n int) error {
	if isNil(merr) {
		return nil
	}
	l := len(merr.errors)
	if n < 0 || n >= l {
		return nil
	}
	return merr.errors[n]
}

// ErrorOrNil is used to get a clean error interface for reflection. If
// the MultiError is empty it returns nil, and if there is a single
// error then it is unnested. Otherwise, it returns the MultiError
// retyped for the error interface.
//
// Retrieving the MultiError is simple, since NewMultiError flattens
// MultiErrors passed to it:
//
//	err := errors.NewMultiError(e1, e2, e3).ErrorOrNil()
//	newMErr := errors.NewMultiError(err)
//	newMErr.Errors() // => []error{e1, e2, e3}
func (merr *MultiError) ErrorOrNil() error {
	if len(merr.Errors()) == 0 {
		return nil
	}
	if len(merr.Errors()) == 1 {
		return merr.errors[0]
	}
	return merr
}

// Err is an alias for ErrorOrNil. It is used to get a clean error
// interface for reflection. If the MultiError is empty it returns nil,
// and if there is a single error then it is unnested. Otherwise, it
// returns the MultiError retyped for the error interface.
func (merr *MultiError) Err() error {
	return merr.ErrorOrNil()
}

// Unwrap implements the error Unwrap interface. It always returns nil
// since a MultiError may not wrap another error. The errors in a
// MultiError may be able to be Unwrapped, however.
func (merr *MultiError) Unwrap() error { return nil }

// As finds the first error that matches target, and if so, sets target
// to that error value and returns true. Otherwise, it returns false.
//
// This function allows As to traverse the values stored on the
// MultiError, even though the type has a null Unwrap implementation.
func (merr *MultiError) As(target interface{}) bool {
	for _, err := range merr.Errors() {
		if As(err, target) {
			return true
		}
	}
	return false
}

// Is reports whether any error matches target.
//
// This function allows Is to traverse the values stored on the
// MultiError, even though the type has a null Unwrap implementation.
func (merr *MultiError) Is(target error) bool {
	for _, err := range merr.Errors() {
		if Is(err, target) {
			return true
		}
	}
	return false
}

func (merr *MultiError) Format(s fmt.State, verb rune) {
	switch verb {
	case 'v':
		switch {
		case s.Flag('+'):
			size := len(merr.Errors())
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

func formatMessages(w io.Writer, merr multiError, delimiters [2]string) {
	first := true
	io.WriteString(w, delimiters[0])
	for _, err := range merr.Errors() {
		if !first {
			io.WriteString(w, "; ")
		}
		io.WriteString(w, err.Error())
		first = false
	}
	io.WriteString(w, delimiters[1])
}

// ErrorsFrom returns a list of errors that the supplied error is
// composed of. multiErrors are unwrapped, flattened, and returned. If
// the error is nil, or is a multiError with no errors, a nil slice is
// returned. It is useful when an API has forced a MultiError to be
// returned as an error type, or when it is unknown if a given error is
// a MultiError or not:
//
//	var err error
//	// ...
//	if errors.AppendInto(&err, w.Close()) {
//	    errs := errors.ErrorsFrom(err)
//	}
//
// If the error is not composed of other errors, the returned slice
// contains just the error that was passed in.
//
// Callers of this function are free to modify the returned slice.
func ErrorsFrom(err error) []error {
	if isNil(err) {
		return nil
	}
	if merr, ok := err.(*MultiError); ok {
		errs := merr.Errors()
		if len(errs) == 0 {
			return nil
		}
		result := make([]error, len(errs))
		copy(result, errs)
		return result

	}
	if mm := unwrapMultiErr(err); mm != nil {
		errs := flatten(mm)
		if len(errs) == 0 {
			return nil
		}
		result := make([]error, len(errs))
		copy(result, errs)
		return result
	}
	return []error{err}
}

// Append is a version of NewMultiError optimized for the most common
// case of appending errors: two errors where the first may be a
// multiError but the second definitely is not. If you pass a multiError
// as the second error Append will ignore it and add a new, specific
// error to the returned MultiError.
//
// The following pattern may also be used to record failure of deferred
// operations without losing information about the original error.
//
//	func doSomething(..) (err error) {
//		f := acquireResource()
//		defer func() {
//			err = errors.Append(err, f.Close())
//		}()
//
// QUESTION(PH): should we panic instead of add error?
func Append(receivingErr error, appendingErr error) *MultiError {
	receivingErrIsNil := isNil(receivingErr)
	appendingErrIsNil := isNil(appendingErr)
	if receivingErrIsNil && appendingErrIsNil {
		return nil
	}

	switch {
	case receivingErrIsNil:
		if mAppendingErr := unwrapMultiErr(appendingErr); mAppendingErr != nil {
			appendingErr = New("errors.Append used incorrectly: " +
				"second parameter may not be a multiError")
		}
		return &MultiError{errors: []error{appendingErr}}
	case appendingErrIsNil:
		if mReceivingErr := unwrapMultiErr(receivingErr); mReceivingErr != nil {
			return &MultiError{errors: flatten(mReceivingErr)}
		}
		return &MultiError{errors: []error{receivingErr}}
	default:
		if mAppendingErr := unwrapMultiErr(appendingErr); mAppendingErr != nil {
			appendingErr = New("errors.Append used incorrectly: " +
				"second parameter may not be a multiError")
		}
		if mReceivingErr := unwrapMultiErr(receivingErr); mReceivingErr != nil {
			return &MultiError{errors: append(flatten(mReceivingErr), appendingErr)}
		}
		return &MultiError{errors: []error{receivingErr, appendingErr}}
	}
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
//	err := errors.Append(r.Close(), w.Close()).ErrorOrNil()
//
// As AppendInto reports whether the provided error was non-nil, it may
// be used to build an errors error in a loop more ergonomically. For
// example:
//
//	var err error
//	for line := range lines {
//	    var item Item
//	    if errors.AppendInto(&err, parse(line, &item)) {
//	        continue
//	    }
//	    items = append(items, item)
//	}
//	if err != nil {
//	    log.Fatal(err)
//	}
//
// Compare this with a version that relies solely on Append:
//
//	var merr *errors.MultiError
//	for line := range lines {
//	    var item Item
//	    if parseErr := parse(line, &item); parseErr != nil {
//	        merr = errors.Append(merr, parseErr)
//	        continue
//	    }
//	    items = append(items, item)
//	}
//	err := merr.ErrorOrNil()
//	if err != nil {
//	    log.Fatal(err)
//	}
//
// As in Append, if you pass a multiError as the second error AppendInto
// will ignore it and add a new, specific error to the returned
// MultiError.
//
// QUESTION(PH): should we panic instead of add error?
func AppendInto(receivingErr *error, appendingErr error) bool {
	switch {
	case receivingErr == nil:
		// We panic if 'into' is nil. This is not documented above
		// because suggesting that the pointer must be non-nil may
		// confuse users into thinking that the error that it points
		// to must be non-nil.
		panic(NewWithStackTrace(
			"errors.AppendInto used incorrectly: receiving pointer must not be nil"))
	case isNil(appendingErr):
		*receivingErr = NewMultiError(*receivingErr).ErrorOrNil()
		return false
	default:
		if mm := unwrapMultiErr(appendingErr); mm != nil {
			appendingErr = New("errors.AppendInto used incorrectly: " +
				"second parameter may not be a multiError")
		}
		*receivingErr = Append(*receivingErr, appendingErr).ErrorOrNil()
		return true
	}
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
//	    // ...
//	    f, err := openFile(..)
//	    if err != nil {
//	        return err
//	    }
//
//	    // errors will call f.Close() when this function returns, and if the
//	    // operation fails it will append its error into the returned error.
//	    defer errors.AppendInvoke(&err, f.Close)
//
//	    scanner := bufio.NewScanner(f)
//	    // Similarly, this scheduled scanner.Err to be called and inspected
//	    // when the function returns and append its error into the returned
//	    // error.
//	    defer errors.AppendResult(&err, scanner.Err)
//
//	    // ...
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
