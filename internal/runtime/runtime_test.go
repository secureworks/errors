package runtime

import (
	"fmt"
	"runtime"
	"testing"

	"github.com/secureworks/errors/internal/testutils"
)

// Callers to build up a call stack in tests.

type callerStruct struct{}

func (c callerStruct) PtrFrameCaller(skip int) runtime.Frame {
	return FrameCaller(skip)
}

func (c callerStruct) PtrStackCaller(skip int) []runtime.Frame {
	return StackCaller(skip)
}

func FrameCaller(skip int) runtime.Frame {
	return GetFrame(skip)
}

func StackCaller(skip int) []runtime.Frame {
	return GetStack(skip)
}

var (
	getFrameLine = 25 // Line no for the utility in the codebase.
	getStackLine = 11 // Line no for the utility in the codebase.
)

func TestGetFrame(t *testing.T) {
	var cs callerStruct
	cases := []struct {
		name  string
		frame runtime.Frame
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
			fn:    `.+\/runtime\.FrameCaller`,
			file:  `.+\/runtime_test\.go`,
			line:  24,
		},
		{
			name:  "skip:2",
			frame: cs.PtrFrameCaller(2),
			fn:    `.+\/runtime\.callerStruct\.PtrFrameCaller`,
			file:  `.+\/runtime_test\.go`,
			line:  16,
		},
		{
			name:  "skip:3",
			frame: cs.PtrFrameCaller(3),
			fn:    `.+\/runtime\.TestGetFrame`,
			file:  `.+\/runtime_test\.go`,
			line:  68,
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
	var cs callerStruct
	stack := cs.PtrStackCaller(0)

	cases := []struct {
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
			fn:   `.+\/runtime\.StackCaller`,
			file: `.+\/runtime_test\.go`,
			line: 28,
		},
		{
			fn:   `.+\/runtime\.callerStruct\.PtrStackCaller`,
			file: `.+\/runtime_test\.go`,
			line: 20,
		},
		{
			fn:   `.+\/runtime\.TestGetStack`,
			file: `.+\/runtime_test\.go`,
			line: 100,
		},
		{
			fn:   `testing\.tRunner`,
			file: `.+\/testing\/testing\.go`,
		},
	}
	testutils.AssertEqual(t, len(cases), len(stack))
	for i, fr := range cases {
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
			got := FuncName(tt.name)
			want := tt.want
			if got != want {
				t.Errorf("funcname(%q): want: %q, got %q", tt.name, want, got)
			}
		})
	}
}
