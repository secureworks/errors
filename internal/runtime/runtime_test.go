package runtime_test

import (
	"fmt"
	stdruntime "runtime"
	"testing"

	"github.com/secureworks/errors/internal/runtime"
	"github.com/secureworks/errors/internal/testutils"
)

// Callers to build up a call stack in tests.

type CallerStruct struct{}

func (c *CallerStruct) PtrFrameCaller(skip int) stdruntime.Frame {
	return FrameCaller(skip)
}

func (c *CallerStruct) PtrStackCaller(skip int) []stdruntime.Frame {
	return StackCaller(skip)
}

func FrameCaller(skip int) stdruntime.Frame {
	return runtime.GetFrame(skip)
}

func StackCaller(skip int) []stdruntime.Frame {
	return runtime.GetStack(skip)
}

var (
	getFrameLine = 25 // Line no for the utility in the codebase.
	getStackLine = 11 // Line no for the utility in the codebase.
)

func TestGetFrame(t *testing.T) {
	var cs *CallerStruct
	cases := []struct {
		name  string
		frame stdruntime.Frame
		fn    string
		file  string
		line  int
	}{
		{
			name:  "skip:0",
			frame: cs.PtrFrameCaller(0),
			fn:    `.+\/runtime\.GetFrame`,
			file:  `.+\/runtime\.go`,
			line:  getFrameLine,
		},
		{
			name:  "skip:1",
			frame: cs.PtrFrameCaller(1),
			fn:    `.+\/runtime_test\.FrameCaller`,
			file:  `.+\/runtime_test\.go`,
			line:  25,
		},
		{
			name:  "skip:2",
			frame: cs.PtrFrameCaller(2),
			fn:    `.+\/runtime_test\.\(\*CallerStruct\)\.PtrFrameCaller`,
			file:  `.+\/runtime_test\.go`,
			line:  17,
		},
		{
			name:  "skip:3",
			frame: cs.PtrFrameCaller(3),
			fn:    `.+\/runtime_test\.TestGetFrame`,
			file:  `.+\/runtime_test\.go`,
			line:  69,
		},
		{
			name:  "skip:4",
			frame: cs.PtrFrameCaller(4),
			fn:    `testing\.tRunner`,
			file:  `.+\/testing\/testing\.go`,
		},
		{
			name:  "skip:5",
			frame: cs.PtrFrameCaller(5), // Empty.
			fn:    "",
			file:  "",
			line:  0,
		},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			testutils.AssertMatch(t, tt.fn, tt.frame.Function)
			testutils.AssertMatch(t, tt.file, tt.frame.File)
			if tt.line != 0 {
				testutils.AssertEqual(t, tt.line, tt.frame.Line)
			}
		})
	}
}

func TestGetStack(t *testing.T) {
	var cs *CallerStruct
	stack := cs.PtrStackCaller(0)

	frames := []struct {
		fn   string
		file string
		line int
	}{
		{
			fn:   `.+\/runtime\.GetStack`,
			file: `.+\/runtime\.go`,
			line: getStackLine,
		},
		{
			fn:   `.+\/runtime_test\.StackCaller`,
			file: `.+\/runtime_test\.go`,
			line: 29,
		},
		{
			fn:   `.+\/runtime_test\.\(\*CallerStruct\)\.PtrStackCaller`,
			file: `.+\/runtime_test\.go`,
			line: 21,
		},
		{
			fn:   `.+\/runtime_test\.TestGetStack`,
			file: `.+\/runtime_test\.go`,
			line: 101,
		},
		{
			fn:   `testing\.tRunner`,
			file: `.+\/testing\/testing\.go`,
		},
	}
	testutils.AssertEqual(t, len(frames), len(stack))
	for i, fr := range frames {
		t.Run(fmt.Sprintf("call depth %d", i), func(t *testing.T) {
			testutils.AssertMatch(t, fr.fn, stack[i].Function)
			testutils.AssertMatch(t, fr.file, stack[i].File)
			if fr.line != 0 {
				testutils.AssertEqual(t, fr.line, stack[i].Line)
			}
		})
	}
}

func TestFuncName(t *testing.T) {
	cases := []struct {
		name, want string
	}{
		{name: "", want: ""},
		{name: "runtime.main", want: "main"},
		{name: "github.com/secureworks/errors.funcname", want: "funcname"},
		{name: "funcname", want: "funcname"},
		{name: "io.copyBuffer", want: "copyBuffer"},
		{name: "main.(*R).Write", want: "(*R).Write"},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			got := runtime.FuncName(tt.name)
			want := tt.want
			if got != want {
				t.Errorf("funcname(%q): want: %q, got %q", tt.name, want, got)
			}
		})
	}
}
