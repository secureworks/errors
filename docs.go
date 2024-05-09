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
//		return err
//	}
//
// ... we allow errors to lose error context (ie human-readable root
// cause and debugging information).
//
// Go 1.13 introduced "error wrapping," where we can add context
// messages like this:
//
//	if err != nil {
//		return fmt.Errorf("contextual information: %w", err)
//	}
//
// This helps us identify a root causs and place that cause in some
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
// 3. to format errors with all oftheir context when printing them; and
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
//	var Err = errors.New("example err") // Same as if we had used: import "errors"
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
//		Location() (function string, file string, line int)
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
// Since the latter approach (appending frames) leads to more compact
// and efficient debugging information, and since it mirrors the Go
// idiom of recursively building an error context, this package prefers
// its use and includes the errors.Errorf function to that effect. Using
// stack traces is fully supported, however, and errors.FramesFrom will
// extract a stack trace, even if there are appended frames in an error
// chain (if both are available), in order to avoid context loss.
//
// # Wrapping Errors
//
// This package provides functions for adding context to an error with a
// group of "error wrappers" that build an "error chain" of values that
// recursively add that context to some base error. The error wrappers
// it provides are:
//
//	// Attach a stack trace starting at the current caller.
//	err := errors.WithStackTrace(err)
//	// Append a frame for the current caller.
//	err = errors.WithFrame(err)
//	// Appends a frame for the caller *n* steps up the chain (in this case, 1).
//	err = errors.WithFrameAt(err, 1)
//
// These wrappers are accompanied by versions that create a new error
// and immediately wrap it: errors.NewWithStackTrace, errors.NewWithFrame,
// and errors.NewWithFrameAt.
//
// A final helper, errors.Errorf, is provided to allow for the common
// idiom:
//
//	err := errors.WithFrame(fmt.Errorf("message context: %w", err))
//	// ... the same as:
//	// err := errors.Errorf("message context: %w", err)
//
// In order to ensure the user correctly structures errors.Errorf, the
// function will panic if you are not wrapping an error with the "%w"
// verb.
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
// others: errors.As, errors.Is, fmt.Formatter, et al).
//
// It also provides helper functions for writing code that handles
// multierrors either as their own type or as a basic error type. For
// example, if you want to merge the results of two functions into a
// multierror, you can use:
//
//	func actionWrapper() *errors.MultiError {
//		err1 := actionA()
//		err2 := actionB()
//		return errors.NewMultiError(err1, err2)
//	}
//
//	if merr := actionWrapper(); merr != nil {
//		fmt.Printf("%+v\n", merr.Errors())
//	}
//
// # Retrieving error information
//
// The additional types of context this package's wrappers add: call
// frames or stack (for debugging) can most easily be extracted from an
// error or error chain using errors.FramesFrom and errors.ErrorsFrom.
//
// errors.FramesFrom returns an errors.Frames slice. It identifies if
// the error chain has a stack trace, and if it does it will return the
// oldest / deepest one available (to get the most context). If the
// error chain does not have a stack trace, but has frames appended,
// errors.FramesFrom merges those frames in order from most recent to
// oldest and returns it.
//
//	err := errors.NewWithStackTrace("err")
//	frames := errors.FramesFrom(err)
//	len(frames)
//	// 6
//
//	err := errors.NewWithFrame("err")
//	err = errors.WithFrame(err)
//	frames := errors.FramesFrom(err)
//	len(frames)
//	// 2
//
//	err := errors.New("err")
//	frames := errors.FramesFrom(err)
//	len(frames)
//	// 0
//
// errors.ErrorsFrom returns a slice of errors, unwrapping the first
// multierror found in an error chain and returning the results. If none
// is found, the slice of errors contains the given error, or is nil if
// the error is nil:
//
//	merr := errors.NewMultiError(errors.New("err"), errors.New("err"))
//	err := errors.WithStackTrace(merr)
//	errs := errors.ErrorsFrom(err)
//	len(errs)
//	// 2
//
// # Masking Errors
//
// Because this errors package allows us to add a fair amount of
// sensitive context to errors, and since Go errors are often used to
// provide end users with useful information, it is important to also
// provide primitives for removing (or "masking") context in an error
// chain.
//
// Foremost is the wrapper function errors.WithMessage, which will reset
// a message context (often including information that is logged or that
// will be provided to an end user), while leaving the rest of the
// context and type information available on the error chain to be used
// by calling code. For example:
//
//	err := errors.NewWithFrame("user 4356789 missing role Admin: has roles [EndUser] in tenant 42")
//	// ...
//	err = errors.WithMessage(err, "user unauthorized")
//	fmt.Printf("%+v", err)
//	//> user unauthorized
//	//> pkg/function.name
//	//>     file_name.go:20
//
// The resulting error can be unwrapped:
//
//	err := errors.Unwrap(err)
//	fmt.Print(err.Error())
//	//> user 4356789 missing role Admin: has roles [EndUser] in tenant 42
//
// The opposite effect can be had by using errors.Mask to remove all
// non-message context:
//
//	err := errors.NewWithFrame("user unauthorized")
//	// ...
//	err = errors.Mask(err)
//	fmt.Printf("%+v", err)
//	//> user unauthorized
//	errors.FramesFrom(err)
//	// []
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
//	%s    print the error's message context. Frames are not included in
//	      the message.
//	%v    see %s
//	%+v   extended format. Each Frame of the error's Frames will
//	      be printed in detail.
//
// This not an exhaustive list, see the tests for more.
//
// # Unexported interfaces
//
// Following the precedent of other errors packages, this package is
// implemented using a series of unexported structs that all conform to
// various interfaces in order to be activated. All the error wrappers,
// for example, implement the error interface face type and the unwrap
// interface:
//
//	interface {
//		Error() string // The built-in language type "error."
//		Unwrap() error // The unexported standard library interface used to unwrap errors.
//	}
//
// This package does export the important interface errors.Frame, but
// otherwise it does not export interfaces that are not necessary to use
// the library. However, if you want to write more complex code that
// makes use of, or augments, this package, there are unexported
// interfaces throughout that can be used to work more directly with its
// types. These include:
//
//	interface {
//		// Used for extracting context.
//		Frames() errors.Frames // Ie "framer," the interface for getting any frames from an error.
//
//		// Used to distinguish a stack trace from other appended frames:
//		StackTrace() []uintptr // Ie "stackTracer," the interface for getting a local stack trace from an error.
//
//		// Used to distinguish a frame that was generated from runtime (instead of synthetically):
//		PC() uintptr // Ie "programCounter," the interface for getting a frame's program counter.
//
//		// Used to identify an error that coalesces multiple errors:
//		Errors() []error // Ie "multiError," the interface for getting multiple merged errors.
//	}
//
// Though none of these are exported by this package, they are
// considered a part of its stable public interface.
package errors
