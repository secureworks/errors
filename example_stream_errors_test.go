package errors_test

import (
	"bufio"
	"bytes"
	"fmt"
	"io"

	"github.com/secureworks/errors"
)

var splitTokensOn = []byte("\n\n")

// Crate a scanner that tokenizes on two newlines.
func tokenizer(data []byte, atEOF bool) (advance int, token []byte, err error) {
	if atEOF && len(data) == 0 {
		return 0, nil, nil
	}
	if i := bytes.Index(data, splitTokensOn); i >= 0 {
		// We have a full newline-terminated line.
		return i + 2, data[0:i], nil
	}
	if atEOF {
		return len(data), data, nil
	}
	return 0, nil, nil
}

// Here we can see that it's straighforward to send errors over some
// pipe by serializing into bytes.
func Example_streamErrors() {
	r, w := io.Pipe()

	errFrames := errors.NewWithFrame("err w frames")
	errFrames = errors.Errorf("inner context: %w", errFrames)
	errStack := errors.NewWithStackTrace("err w stack")

	go func() {
		var errs = []error{
			errors.Errorf("outer context: %w", errFrames),
			errors.New("basic err"),
			errStack,
		}

		for _, err := range errs {
			fmt.Fprintf(w, "%+v%s", err, splitTokensOn)
		}

		w.Close()
	}()

	scanner := bufio.NewScanner(r)
	scanner.Split(tokenizer)
	for scanner.Scan() {
		err, _ := errors.ErrorFromBytes(scanner.Bytes())
		pprintf("\nREAD IN ERROR: %+v\n", err)
	}

	// Output:
	//
	// READ IN ERROR: outer context: inner context: err w frames
	// github.com/secureworks/errors_test.Example_streamErrors
	// 	/home/testuser/pkgs/errors/example_stream_errors_test.go:0
	// github.com/secureworks/errors_test.Example_streamErrors
	// 	/home/testuser/pkgs/errors/example_stream_errors_test.go:0
	// github.com/secureworks/errors_test.Example_streamErrors.func1
	// 	/home/testuser/pkgs/errors/example_stream_errors_test.go:0
	//
	// READ IN ERROR: basic err
	//
	// READ IN ERROR: err w stack
	// github.com/secureworks/errors_test.Example_streamErrors
	// 	/home/testuser/pkgs/errors/example_stream_errors_test.go:0
	// testing.runExample
	// 	/go/src/testing/run_example.go:0
	// testing.runExamples
	// 	/go/src/testing/example.go:0
	// testing.(*M).Run
	// 	/go/src/testing/testing.go:0
	// main.main
	// 	_testmain.go:0
	// runtime.main
	// 	/go/src/runtime/proc.go:0
}
