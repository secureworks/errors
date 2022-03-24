package errors_test

import (
	"fmt"
	"testing"

	"github.com/secureworks/errors"
	"github.com/secureworks/errors/internal/testutils"
)

// Callers to build up a call stack in tests.

type CallerStruct struct{}

func (c *CallerStruct) PtrFrameCaller(skip int) errors.Frame {
	return FrameCallerAt(skip)
}

func (c *CallerStruct) PtrStackCaller(skip int) []errors.Frame {
	return StackCallerAt(skip)
}

func FrameCallerAt(skip int) errors.Frame {
	return errors.CallerAt(skip)
}

func StackCallerAt(skip int) []errors.Frame {
	return errors.CallStackAt(skip)
}

func testCallerWrapper(skip int) errors.Frames {
	var cs *CallerStruct
	return cs.PtrStackCaller(skip)
}

func (c *CallerStruct) PtrStackMostCaller(skip int, max int) []errors.Frame {
	return StackCallerAtMost(skip, max)
}

func StackCallerAtMost(skip int, max int) []errors.Frame {
	return errors.CallStackAtMost(skip, max)
}

func testCallerMostWrapper(skip int, max int) errors.Frames {
	var cs *CallerStruct
	return cs.PtrStackMostCaller(skip, max)
}

var (
	callerWrapLine     = 16
	callStackWrapLine  = 20
	callerAtLine       = 24
	callStackAtLine    = 28
	funcWrapLine       = 33
	callerMostWrapLine = 37
	callerAtMostLine   = 41
	funcMostLine       = 46
)

func TestCallerAt(t *testing.T) {
	var cs *CallerStruct
	cases := []struct {
		name  string
		frame errors.Frame
		fn    string
		file  string
		line  int
	}{
		{
			name:  "skip:0",
			frame: cs.PtrFrameCaller(0),
			fn:    `.+\/errors_test\.FrameCaller`,
			file:  `.+\/callers_test\.go`,
			line:  callerAtLine,
		},
		{
			name:  "skip:1",
			frame: cs.PtrFrameCaller(1),
			fn:    `.+\/errors_test\.\(\*CallerStruct\)\.PtrFrameCaller`,
			file:  `.+\/callers_test\.go`,
			line:  callerWrapLine,
		},
		{
			name:  "skip:2",
			frame: cs.PtrFrameCaller(2),
			fn:    `.+\/errors_test\.TestCallerAt`,
			file:  `.+\/callers_test\.go`,
			line:  85,
		},
		{
			name:  "skip:3",
			frame: cs.PtrFrameCaller(3),
			fn:    `testing\.tRunner`,
			file:  `.+\/testing\/testing\.go`,
		},
		{
			name:  "skip:4",
			frame: cs.PtrFrameCaller(4), // Overflow returns empty.
			fn:    "unknown",
			file:  "unknown",
			line:  0,
		},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			function, file, line := tt.frame.Location()
			testutils.AssertMatch(t, tt.fn, function)
			testutils.AssertMatch(t, tt.file, file)
			if tt.line != 0 {
				testutils.AssertEqual(t, tt.line, line)
			}
		})
	}
}

func TestCaller(t *testing.T) {
	var function1, file1, function2, file2 string
	var line1, line2 int

	caller := func() {
		function1, file1, line1 = errors.CallerAt(0).Location()
		function2, file2, line2 = errors.Caller().Location()
	}
	caller()

	testutils.AssertEqual(t, function1, function2)
	testutils.AssertEqual(t, file1, file2)
	testutils.AssertEqual(t, line1, line2-1)
}

func TestCallStackAt(t *testing.T) {
	var testFnFrame = func(line int) testFrame {
		return testFrame{
			`.+\/errors_test\.TestCallStackAt`,
			`.+\/callers_test\.go`,
			line,
		}
	}

	cases := []struct {
		name   string
		stack  errors.Frames
		frames []testFrame
	}{
		{
			name:  "skip:0",
			stack: testCallerWrapper(0),
			frames: []testFrame{
				callerFrame,
				callerStructFrame,
				wrapperFrame,
				testFnFrame(147),
				testRunnerFrame,
			},
		},
		{
			name:  "skip:1",
			stack: testCallerWrapper(1),
			frames: []testFrame{
				callerStructFrame,
				wrapperFrame,
				testFnFrame(158),
				testRunnerFrame,
			},
		},
		{
			name:  "skip:2",
			stack: testCallerWrapper(2),
			frames: []testFrame{
				wrapperFrame,
				testFnFrame(168),
				testRunnerFrame,
			},
		},
		{
			name:  "skip:3",
			stack: testCallerWrapper(3),
			frames: []testFrame{
				testFnFrame(177),
				testRunnerFrame,
			},
		},
		{
			name:  "skip:4",
			stack: testCallerWrapper(4),
			frames: []testFrame{
				testRunnerFrame,
			},
		},
		{
			name:   "skip:5",
			stack:  testCallerWrapper(5),
			frames: []testFrame{}, // Overflow is empty.
		},
		{
			name:   "skip:6",
			stack:  testCallerWrapper(6),
			frames: []testFrame{}, // Overflow is empty.
		},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			testutils.AssertEqual(t, len(tt.frames), len(tt.stack))
			for i, fr := range tt.frames {
				function, file, line := tt.stack[i].Location()
				testutils.AssertMatch(t, fr.fn, function, fmt.Sprintf("frame %d", i))
				testutils.AssertMatch(t, fr.file, file, fmt.Sprintf("frame %d", i))
				if fr.line != 0 {
					testutils.AssertEqual(t, fr.line, line, fmt.Sprintf("frame %d", i))
				}
			}
		})
	}
}

func TestCallStack(t *testing.T) {
	var stack1, stack2 errors.Frames

	caller := func() {
		stack1 = errors.CallStackAt(0)
		stack2 = errors.CallStack()
	}
	caller()

	testutils.AssertEqual(t, len(stack1), len(stack2))
	for i := range stack1 {
		function1, file1, line1 := stack1[i].Location()
		function2, file2, line2 := stack2[i].Location()
		testutils.AssertEqual(t, function1, function2, fmt.Sprintf("frame %d", i))
		testutils.AssertEqual(t, file1, file2, fmt.Sprintf("frame %d", i))
		if i == 0 {
			testutils.AssertEqual(t, line1, line2-1, fmt.Sprintf("frame %d", i))
		} else {
			testutils.AssertEqual(t, line1, line2, fmt.Sprintf("frame %d", i))
		}
	}
}

func TestCallStackAtMost(t *testing.T) {
	var testFnFrame = func(line int) testFrame {
		return testFrame{
			`.+\/errors_test\.TestCallStackAtMost`,
			`.+\/callers_test\.go`,
			line,
		}
	}

	cases := []struct {
		name   string
		stack  errors.Frames
		frames []testFrame
	}{
		{
			name:  "skip:0,max:0",
			stack: testCallerMostWrapper(0, 0),
			frames: []testFrame{
				callerMostFrame,
				callerMostStructFrame,
				wrapperMostFrame,
				testFnFrame(255),
				testRunnerFrame,
			},
		},
		{
			name:  "skip:0,max:3",
			stack: testCallerMostWrapper(0, 3),
			frames: []testFrame{
				callerMostFrame,
				callerMostStructFrame,
				wrapperMostFrame,
			},
		},
		{
			name:  "skip:0,max:6",
			stack: testCallerMostWrapper(0, 6),
			frames: []testFrame{
				callerMostFrame,
				callerMostStructFrame,
				wrapperMostFrame,
				testFnFrame(275),
				testRunnerFrame,
			},
		},
		{
			name:  "skip:2,max:0",
			stack: testCallerMostWrapper(2, 0),
			frames: []testFrame{
				wrapperMostFrame,
				testFnFrame(286),
				testRunnerFrame,
			},
		},
		{
			name:  "skip:2,max:2",
			stack: testCallerMostWrapper(2, 2),
			frames: []testFrame{
				wrapperMostFrame,
				testFnFrame(295),
			},
		},
		{
			name:  "skip:2,max:6",
			stack: testCallerMostWrapper(2, 6),
			frames: []testFrame{
				wrapperMostFrame,
				testFnFrame(303),
				testRunnerFrame,
			},
		},
		{
			name:   "skip:6,max:0",
			stack:  testCallerMostWrapper(6, 0),
			frames: []testFrame{}, // Overflow is empty.
		},
		{
			name:   "skip:6,max:6",
			stack:  testCallerMostWrapper(6, 6),
			frames: []testFrame{}, // Overflow is empty.
		},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			testutils.AssertEqual(t, len(tt.frames), len(tt.stack))
			for i, fr := range tt.frames {
				function, file, line := tt.stack[i].Location()
				testutils.AssertMatch(t, fr.fn, function, fmt.Sprintf("frame %d", i))
				testutils.AssertMatch(t, fr.file, file, fmt.Sprintf("frame %d", i))
				if fr.line != 0 {
					testutils.AssertEqual(t, fr.line, line, fmt.Sprintf("frame %d", i))
				}
			}
		})
	}
}

type testFrame struct {
	fn   string
	file string
	line int
}

var (
	callerFrame = testFrame{
		fn:   `.+\/errors_test\.StackCaller`,
		file: `.+\/callers_test\.go`,
		line: callStackAtLine,
	}
	callerStructFrame = testFrame{
		fn:   `.+\/errors_test\.\(\*CallerStruct\)\.PtrStackCaller`,
		file: `.+\/callers_test\.go`,
		line: callStackWrapLine,
	}
	wrapperFrame = testFrame{
		fn:   `.+\/errors_test\.testCallerWrapper`,
		file: `.+\/callers_test\.go`,
		line: funcWrapLine,
	}
	callerMostFrame = testFrame{
		fn:   `.+\/errors_test\.StackCallerAtMost`,
		file: `.+\/callers_test\.go`,
		line: callerAtMostLine,
	}
	callerMostStructFrame = testFrame{
		fn:   `.+\/errors_test\.\(\*CallerStruct\)\.PtrStackMostCaller`,
		file: `.+\/callers_test\.go`,
		line: callerMostWrapLine,
	}
	wrapperMostFrame = testFrame{
		fn:   `.+\/errors_test\.testCallerMostWrapper`,
		file: `.+\/callers_test\.go`,
		line: funcMostLine,
	}
	testRunnerFrame = testFrame{
		fn:   `testing\.tRunner`,
		file: `.+\/testing\/testing\.go`,
	}
)
