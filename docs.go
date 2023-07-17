// Package errors provides utilities for working with Go errors. It is
// meant to work as a drop-in replacement for the standard library
// https://go.pkg.dev/errors, and is based on:
//
// • https://github.com/pkg/errors,
//
// • https://pkg.go.dev/golang.org/x/xerrors, and
//
// • https://github.com/uber-go/multierr.
//
// # Error context
//
// When we write the following:
//
//	if err != nil {
//	    return err
//	}
//
// ... we allow errors to lose error context (ie human-readable root
// cause and debugging information).
//
// Go 1.13 introduced "error wrapping," where we can add context
// messages like this:
//
//	if err != nil {
//	    return fmt.Errorf("contextual information: %w", err)
//	}
//
// This helps us identify a root causes and place that cause in some
// program context recursively.
//
// However, we are not always in control of the full extent of our
// codebase, and even when we are we don't always write code that
// provides useful error context.
//
// In addition, there are cases where we want more or different
// information appended to an error: caller frame locations and stacks,
// easily identifiable and immutable causes, or collections of errors
// that result from coalescing the outcome of multiple tasks for
// example.
//
// This package allows users:
//
// 1. to add more fine-grained contextual information to their errors;
//
// 2. to chain or group errors and extract their contextual information;
//
// 3. to format errors with all of their context when printing them; and
//
// 4. to retain a simple API for most use cases while retaining the
// ability to directly interact with, or tune, error chains and groups.
//
// # Error wrapping in Go 1.13
//
// This errors package is meant to be used in addition to the updates in
// https://go.dev/blog/go1.13-errors. Therefore, you shouldn't include
// it (and in fact it will cause your build to fail) if you are using Go
// 1.12 or earlier.
//
// Importantly: this package does not attempt to replace this system.
// Instead, errors is meant to enrich it: all the types and interfaces
// here work with Is, As, and Unwrap; using fmt.Errorf is also
// supported. In fact, this package re-exports New, Is, As, and Unwrap
// so that you don't need to import the standard library's "errors"
// package as well:
//
//	import "github.com/secureworks/errors"
//
//	var Err = errors.New("example err") // Still compiles after you switch from: import "errors"
//
// # Stack traces or call frames
//
// For debugging context this package provides the errors.Frame
// interface and errors.Frames type. Frame is based on the runtime.Frame
// and xerrors.Frame types
// (https://pkg.go.dev/golang.org/x/xerrors#Frame) and defines one
// method, Location:
//
//	type Frame interface {
//	    Location() (function string, file string, line int)
//	}
//
// You can create a Frame in your code directly with errors.Caller or
// errors.CallerAt. You can also use the runtime package to acquire a
// "program counter" (https://pkg.go.dev/runtime#Frame) using
// errors.FrameFromPC. Finally, you can generate a "synthetic" frame by
// passing the constituent data directly to errors.NewFrame:
//
//	fr := errors.Caller() // Same as errors.CallerAt(0)
//	fr.Location()         // ... returns "pkg/function.name", "file_name.go", 20
//
// errors.Frames is a slice of error.Frame. You can bunch together a
// group of errors.Frame instances to create this list, or you can use
// errors.CallStack or errors.CallStackAt to get the entire call stack
// (or some subset of it).
//
//	stack := errors.CallStack() // Same as errors.CallStackAt(0)
//	stack[0].Location()         // ... returns "pkg/function.name", "file_name.go", 20
//
// These two approaches to building errors.Frames can be described, from
// the point of view of adding error context, as "appending frames" or
// as "attaching a stack trace." Which you want to do depends on your
// use case: do you want targeted caller references or an entire stack
// trace attached to your error? The "stack trace" approach is the only
// one supported by https://github.com/pkg/errors, while the "append
// frames" approach is supported by
// https://pkg.go.dev/golang.org/x/xerrors, as examples.
//
// This package follows the "attaching a stack trace" approach for errors
// it generates, despite that being more "verbose" than the more compact
// "appending frames" approach, for ease of use and simplicity.
//
// # Wrapping Errors
//
// This package generates errors that capture context automatically whether
// they are wrapping errors or not. When it is wrapping errors, it allows to
// build an "error chain" in which each error wraps an error (or multiple ones;
// see below).
//
// The way to build such error chains stays the same as using the standard library's
// fmt.Errorf function, but using errors.New instead. Here are a few examples - all
// of which capture the current stack trace:
//
//	// Creates a simple standalone error:
//	err := errors.New("ooops")
//
//	// Wrap another error, generating a "chain" (of just two elements):
//	err := errors.New("customer load error: %w", errors.New("file not found"))
//
//	// Wrap another error, but without including its message in the wrapping error's message:
//	err := errors.New("customer load error", errors.New("file not found"))
//	fmt.Println(err) // Prints "customer load error" only
//	fmt.Printf("%+v", err) // Prints the full error chain, including stack traces of the wrapper & wrapped errors
//
// # Multierrors
//
// Wrapping errors is useful enough, but there are instances when we
// want to merge multiple errors and handle them as a group, eg: a
// "singular" result of running multiple tasks, handling a response to
// some graph resolution where each path may include a separate error,
// returning some possible error *and* the coalesced result of running a
// deferred function, etc.
//
// This can be handled with a simple []error slice, but that can be
// frustrating since so many libraries, codebases and standards expect
// that well-formatted code adheres to the Go idiom of returning a
// single error value from a function or providing an error chain to
// errors.Unwrap.
//
// To solve this the package provides a type errors.MultiError that
// wraps a slice of errors and implements the error interface (and
// others: errors.As, errors.Is, fmt.Formatter, etc.)
//
// It also provides helper functions for writing code that handles
// multierrors either as their own type or as a basic error type. For
// example, if you want to merge the results of two functions into a
// multierror, you can use:
//
//	func actionWrapper() *errors.MultiError {
//	    err1 := actionA()
//	    err2 := actionB()
//	    return errors.NewMultiError(err1, err2)
//	}
//
//	if merr := actionWrapper(); merr != nil {
//	    fmt.Printf("%+v\n", merr.Errors())
//	}
//
// # Retrieving error information
//
// The additional types of context this package's wrappers add: call
// frames or stack (for debugging) can most easily be extracted from an
// error or error chain using the errors.ErrorsFrom function, and via
// one of the StackTracer, Framer, ChainStackTracer, ChainFramer interfaces.
//
// errors.ErrorsFrom returns a slice of errors, unwrapping the first
// multierror found in an error chain and returning the results. If none
// is found, the slice of errors contains the given error, or is nil if
// the error is nil:
//
//	merr := errors.NewMultiError(errors.New("err"), errors.New("err"))
//	err := errors.New("wrapper: %w", merr)
//	errs := errors.ErrorsFrom(err)
//	len(errs) // 2
//
// # Masking Errors
//
// Because this errors package allows us to add a fair amount of
// sensitive context to errors, and since Go errors are often used to
// provide end users with useful information, it is important to also
// provide primitives for removing (or "masking") context in an error
// chain.
//
// The simplest way is to wrap the context error with another error, but
// not to use the "%w" verb in the message, thus wrapping the context error,
// but not including its message in the resulting (wrapping) error, like so:
//
//	root := errors.New("user 4356789 missing role Admin: has roles [EndUser] in tenant 42")
//	// ...
//	err := errors.New("user unauthorized", root)
//	fmt.Printf("%v", err) // prints "user unauthorized"
//	fmt.Printf("%+v", err)
//	//> user unauthorized
//	//> ...<full-chain-stack-trace-printed-here>...
//
// The resulting error can be unwrapped:
//
//	unwrapped := errors.Unwrap(err)
//	fmt.Print(unwrapped == root) // prints "true"
//	fmt.Print(unwrapped) // prints "user 4356789 missing role Admin: has roles [EndUser] in tenant 42"
//
// The opposite effect can be had by using errors.Mask to remove all
// non-message context:
//
//	err := errors.New("user unauthorized")
//	fmt.Printf("%+v", err)
//	//> user unauthorized
//	//> ...<stack-trace-leading-to-this-err>...
//	// ...
//	masked := errors.Mask(err)
//	fmt.Printf("%+v", masked) // prints "user unauthorized"
//	fmt.Printf("%+v", masked)
//	//> user unauthorized
//	//> ...<stack-trace-leading-to-MASKED-err>...
//
// While errors.Mask removes all context, errors.Opaque retains all
// context but squashes the error chain so that type information, or any
// context that is not understood by this errors package is removed.
// This can be useful to ensure errors do not wrap some context from an
// outside library not under the calling code's control.
//
// # Formatted printing of errors
//
// All error values returned from this package implement fmt.Formatter
// and can be formatted by the fmt package. The following verbs are
// supported:
//
//	%s    print the error's message, but without the stack-trace
//	%v    same as %s
//	%q    same as %s but quoted
//	%#v   prints the go-syntax representation of the error's message & causing error (if any)
//	%+v   extended format. Prints the message & stack-trace for each error in the chain
package errors
