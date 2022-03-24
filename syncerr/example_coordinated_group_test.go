package syncerr_test

import (
	"context"
	"fmt"

	"github.com/secureworks/errors/syncerr"
)

// CoordinatedGroups are a great, easy shorthand for running a set of
// tasks that must be completed before moving on in a routine: there's
// no need to write out the sync.WaitGroup logic.
//
// In the example below we don't even need to set the shared
// cancellation context: we can use a zero value version of the group to
// synchronize on all tasks completing. If we are interested in
// retaining the error status of these tasks, use a ParallelGroup
// instead.
func Example_simplestCoordinatedGroup() {
	printTask := func(workload string) {
		fmt.Print("\ntask: ", workload)
	}

	group := new(syncerr.CoordinatedGroup)
	for _, name := range []string{"wk", "wk", "wk", "wk"} {
		name := name // https://golang.org/doc/faq#closures_and_goroutines
		group.Go(func() error { printTask(name); return nil })
	}
	_ = group.Wait()

	// Output:
	// task: wk
	// task: wk
	// task: wk
	// task: wk
}

// CoordinatedGroups are a great tool for building pipelines by adding
// a small layer of synchronization on top of them. The below example
// shows a 3-step pipeline that kills all incomplete steps if an error
// arrives in any one step.
func Example_pipeline() {
	// Define a workload unit and pipelines for messaging.
	type workload struct{ V string }
	pipelineReadIn := make(chan workload, 5)
	pipelineMapTo := make(chan workload, 5)
	pipelineResultOut := make(chan string, 5)

	group, ctx := syncerr.NewGroup(context.Background())

	// Step 1: generate values.
	group.Go(func() error {
		defer close(pipelineReadIn)
		for _, wk := range []workload{{"w"}, {"w"}, {"w"}, {"w"}, {"?"}} {
			select {
			case pipelineReadIn <- wk:
			case <-ctx.Done():
				return ctx.Err()
			}
		}
		return nil
	})

	// Step 2: map values.
	group.Go(func() error {
		defer close(pipelineMapTo)
		for wk := range pipelineReadIn {
			select {
			case pipelineMapTo <- workload{V: wk.V + "k"}:
			case <-ctx.Done():
				return ctx.Err()
			}
		}
		return nil
	})

	// Step 3: workers print the path.
	const numWorkers = 2
	for i := 0; i < numWorkers; i++ {
		group.Go(func() error {
			for wk := range pipelineMapTo {
				if wk.V != "wk" {
					return fmt.Errorf("pipeline broken: invalid workload found: %q", wk.V)
				}
				select {
				case pipelineResultOut <- wk.V:
				case <-ctx.Done():
					return ctx.Err()
				}
			}
			return nil
		}, "worker N")
	}

	go func() { // Sync.
		_ = group.Wait()
		close(pipelineResultOut)
	}()

	for str := range pipelineResultOut {
		fmt.Println(str)
	}
	if err := group.Wait(); err != nil {
		fmt.Println(err)
	}

	// Racy, so not making testable.
}
