// Package syncerr provides synchronization utilities for working with
// errors generated by goroutines.
//
// When used in this package, the term "task" defines a simple signature
// for a function that can be run as part of a "group" of goroutines.
// The purpose of the groups (ParallelGroup and CoordinatedGroup) is to
// handle the outcome of running these tasks in a unified fashion: you
// can wait for the *group* to finish and check the outcome as a single
// error status.
//
// All tasks have the signature:
//
//     type task func() error
//
// All groups share the interface:
//
//     type taskGroup interface {
//         Go(func() error, ...string)
//         Wait() error
//     }
//
// CoordinatedGroup
//
// If you want the tasks to run until at least one task returns an
// error, and receive the first error as the outcome, use
// CoordinatedGroup. CoordinatedGroups are modeled directly on
// errgroups: https://pkg.go.dev/golang.org/x/sync/errgroup.
//
// > In fact, the taskGroup interface is implemented by
// > "golang.org/x/sync/errgroup".
//
//     group, ctx := errors.NewCoordinatedGroup(ctx)
//     group.Go(taskRunner, "task", "1")
//     // ...
//     if err := group.Wait(); err != nil {
//         // ...
//     }
//
// The more terse, default version of the task runner returns a
// CoordinatedGroup:
//
//     group, ctx := errors.NewGroup(ctx) // Same as errors.NewCoordinatedGroup.
//
// ParallelGroup
//
// If you want all tasks to run to completion and have their errors
// coalesced, use ParallelGroup:
//
//     group := new(errors.ParallelGroup)
//     group.Go(taskRunner, "task", "1")
//     // ...
//     err := group.Wait()
//     merr, _ := err.(*errors.MultiError)
//     fmt.Println(merr.Errors())
//
// ParallelGroup includes the function WaitForMultiError that skips the
// step of asserting the multierror interface on the result:
//
//     group := new(errors.ParallelGroup)
//     group.Go(taskRunner, "task", "1")
//     // ...
//     merr := group.WaitForMultiError()
//     fmt.Println(merr.Errors())
//
package syncerr // "github.com/secureworks/errors/syncerr"
