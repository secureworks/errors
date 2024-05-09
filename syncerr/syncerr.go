package syncerr // "github.com/secureworks/errors/syncerr"

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"sync"

	"github.com/secureworks/errors"
)

// taskGroup defines an interface shared by the two ways of grouping
// tasks: ParallelGroup and CoordinatedGroup. Using this interface
// allows you to handle either in an obvious way. It is unexported but
// enforced and should be thought of as part of the syncerr API.
type taskGroup interface {
	Go(func() error, ...string)
	Wait() error
}

// NewGroup creates a new CoordinatedGroup with the given context, and a
// reference to the "inner context" that is cancelled when any subtask
// returns an error or when all tasks are complete.
//
// NewGroup is an alias of NewCoordinatedGroup.
func NewGroup(ctx context.Context) (group *CoordinatedGroup, innerCtx context.Context) {
	return NewCoordinatedGroup(ctx)
}

// CoordinatedGroup is a collection of goroutines working on subtasks
// that are part of the same overall task, and which coordinate to
// cancel the overall task when any subtask fails.
type CoordinatedGroup struct {
	wg   sync.WaitGroup
	once sync.Once

	ctx    context.Context
	cancel context.CancelFunc

	err error
}

var _ taskGroup = (*CoordinatedGroup)(nil)

// NewCoordinatedGroup creates a new CoordinatedGroup with the given
// context, and a reference to the "inner context" that is cancelled
// when any subtask returns an error or when all tasks are complete.
func NewCoordinatedGroup(ctx context.Context) (group *CoordinatedGroup, innerCtx context.Context) {
	innerCtx, cancel := context.WithCancel(ctx)
	g := &CoordinatedGroup{ctx: innerCtx, cancel: cancel}
	return g, innerCtx
}

// Go registers and runs a new subtask for the CoordinatedGroup. The
// first call to return a non-nil error cancels the group; its error
// will be returned by Wait.
//
// Go also accepts a "list" of "task names" that are appended to any
// errors this subtask generates.
//
// In order to keep the interface simpler, we do not enforce any
// parameters on the given task runner: if you want to inject the
// context supplied to the group, for example, pass it with a closure:
//
//	group, _ := syncerr.NewCoordinatedGroup(ctx)
//	group.Go(func() error {
//		return someTask(ctx)
//	}, "someTask", "1")
//
// Use this pattern for running tasks that need any number of
// parameters.
func (g *CoordinatedGroup) Go(f func() error, taskNames ...string) {
	g.wg.Add(1)

	go func() {
		defer g.wg.Done()
		if err := f(); !isNil(err) {
			g.once.Do(func() {
				g.err = wrapWithNames(taskNames, err)
				if g.cancel != nil {
					g.cancel()
				}
			})
		}
	}()
}

// Wait blocks until all function calls from the Go method have
// returned, then returns the first non-nil error (if any) from them.
//
// Tasks are in charge of ending themselves if the group's context is
// cancelled, in the case where they may not end on their own.
func (g *CoordinatedGroup) Wait() error {
	g.wg.Wait()
	if g.cancel != nil {
		g.cancel()
	}
	return g.err
}

// ParallelGroup is a collection of goroutines working on subtasks that
// are part of the same overall task, and which return errors that need
// to be handled or coalesced.
//
// There is no factory function to create a ParallelGroup since the zero
// value is a viable instance.
//
//	group := new(syncerr.ParallelGroup)
type ParallelGroup struct {
	mu sync.Mutex
	wg sync.WaitGroup

	merr *errors.MultiError
}

var _ taskGroup = (*ParallelGroup)(nil)

// Go registers and runs a new subtask for the ParallelGroup.
//
// Go also accepts a "list" of "task names" that are appended to any
// errors this subtask generates.
//
// In order to keep the interface simpler, we do not enforce any
// parameters on the given task runner: if you want to inject a
// context to cancel the task, for example, pass it with a closure:
//
//	group := new(syncerr.ParallelGroup)
//	group.Go(func() error {
//		return someTask(ctx)
//	}, "someTask", "1")
//
// Use this pattern for running tasks that need any number of
// parameters.
func (g *ParallelGroup) Go(f func() error, taskNames ...string) {
	g.wg.Add(1)

	go func() {
		defer g.wg.Done()
		if err := f(); !isNil(err) {
			g.mu.Lock()
			defer g.mu.Unlock()
			g.merr = errors.Append(g.merr, wrapWithNames(taskNames, err))
		}
	}()
}

// WaitForMultiError blocks on either all workers completing or the
// group's context being cancelled. All errors generated by the workers
// are returned as an errors.MultiError.
func (g *ParallelGroup) WaitForMultiError() *errors.MultiError {
	g.wg.Wait()
	return g.merr
}

// Wait blocks on either all workers completing or the group's context
// being cancelled. All errors generated by the workers are returned.
func (g *ParallelGroup) Wait() error {
	return g.WaitForMultiError()
}

// wrapWithNames adds identifiers to the error context for a task.
func wrapWithNames(names []string, err error) error {
	if len(names) == 0 {
		return err
	}
	return errors.WithFrameAt(
		fmt.Errorf("%s: %w",
			strings.Join(names, ": "),
			err,
		), 1)
}

func isNil(err error) bool {
	if err == nil {
		return true
	}
	switch reflect.TypeOf(err).Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Ptr, reflect.Slice:
		if reflect.ValueOf(err).IsNil() {
			return true
		}
	default:
	}
	return false
}
