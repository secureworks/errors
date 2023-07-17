package errors_test

import (
	"sync"
	"time"

	"github.com/secureworks/errors"
)

type wrapperType struct{}

func (_ *wrapperType) ReturnError() error {
	return errors.New("err from wrapper type")
}

func runSomeTask(n int) error {
	var wrapper *wrapperType
	time.Sleep(time.Duration(100*n) * time.Millisecond)
	if n%2 == 0 {
		return errors.New("while running some task (%d): %w", n, wrapper.ReturnError())
	}
	return nil
}

// Add useful debugging information and error context to go routines,
// which can be hard to track down.
//
// We can also coalesce errors into an errors.MultiError and handle it
// using single error idioms, which is useful for managing subtasks.
func Example_debugTasks() {
	var wg sync.WaitGroup

	errCh := make(chan error)
	for i := 0; i < 4; i++ {
		wg.Add(1)
		i := i
		go func() {
			defer wg.Done()
			if root := runSomeTask(i); root != nil {
				errCh <- errors.New("wrapper: %w", root)
			}
		}()
	}

	go func() {
		wg.Wait()
		close(errCh)
	}()

	var merr error
	for err := range errCh {
		errors.AppendInto(&merr, err)
	}

	if merr != nil {
		pprintf("%+v", merr)
	}

	// Output: multiple errors:
	//
	// * error 1 of 2: wrapper: while running some task (0): err from wrapper type
	//      github.com/secureworks/errors_test.Example_debugTasks.func1
	//      	/home/testuser/pkgs/errors/example_debug_tasks_test.go:0
	//
	// CAUSED BY: while running some task (0): err from wrapper type
	//      github.com/secureworks/errors_test.runSomeTask
	//      	/home/testuser/pkgs/errors/example_debug_tasks_test.go:0
	//      github.com/secureworks/errors_test.Example_debugTasks.func1
	//      	/home/testuser/pkgs/errors/example_debug_tasks_test.go:0
	//
	// CAUSED BY: err from wrapper type
	//      github.com/secureworks/errors_test.(*wrapperType).ReturnError
	//      	/home/testuser/pkgs/errors/example_debug_tasks_test.go:0
	//      github.com/secureworks/errors_test.runSomeTask
	//      	/home/testuser/pkgs/errors/example_debug_tasks_test.go:0
	//      github.com/secureworks/errors_test.Example_debugTasks.func1
	//      	/home/testuser/pkgs/errors/example_debug_tasks_test.go:0
	//
	// * error 2 of 2: wrapper: while running some task (2): err from wrapper type
	//      github.com/secureworks/errors_test.Example_debugTasks.func1
	//      	/home/testuser/pkgs/errors/example_debug_tasks_test.go:0
	//
	// CAUSED BY: while running some task (2): err from wrapper type
	//      github.com/secureworks/errors_test.runSomeTask
	//      	/home/testuser/pkgs/errors/example_debug_tasks_test.go:0
	//      github.com/secureworks/errors_test.Example_debugTasks.func1
	//      	/home/testuser/pkgs/errors/example_debug_tasks_test.go:0
	//
	// CAUSED BY: err from wrapper type
	//      github.com/secureworks/errors_test.(*wrapperType).ReturnError
	//      	/home/testuser/pkgs/errors/example_debug_tasks_test.go:0
	//      github.com/secureworks/errors_test.runSomeTask
	//      	/home/testuser/pkgs/errors/example_debug_tasks_test.go:0
	//      github.com/secureworks/errors_test.Example_debugTasks.func1
	//      	/home/testuser/pkgs/errors/example_debug_tasks_test.go:0
}
