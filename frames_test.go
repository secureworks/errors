package errors_test

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/secureworks/errors"
	"github.com/secureworks/errors/internal/runtime"
	"github.com/secureworks/errors/internal/testutils"
)

// Callers to build up a call stack in tests.

type FrameCallerStruct struct{}

//go:noinline
func (cs FrameCallerStruct) ValFrameCaller() errors.Frame {
	return getFrame()
}

//go:noinline
func (cs *FrameCallerStruct) PtrFrameCaller() errors.Frame {
	return getFrame()
}

//go:noinline
func StackCaller() []errors.Frame {
	return getStack()
}

// Unexported interfaces.

type pcer interface{ PC() uintptr }

// Default values.

var (
	rtimeFrame = getFrame()
	synthFrame = errors.NewFrame(
		`github.com/secureworks/errors_test.(*ExampleStruct).MethodName.func1`,
		"/usr/u/src/github.com/secureworks/e/errors_example_test.go",
		44,
	)
	emptyFrame = errors.FrameFromPC(0)
	rtimeStack = StackCaller()

	rtimeFramePC = rtimeFrame.(pcer).PC()
)

func TestFrame(t *testing.T) {
	var function, file string
	var line int

	t.Run("runtime frame", func(t *testing.T) {
		function, file, line = rtimeFrame.Location()
		testutils.AssertMatch(t, ".+init", function)
		testutils.AssertMatch(t, ".+frames_test.go", file)
		testutils.AssertEqual(t, 40, line)

		rtimePCer, ok := rtimeFrame.(pcer)
		testutils.AssertTrue(t, ok)
		testutils.AssertNotEqual(t, uintptr(0), rtimePCer.PC())
	})

	t.Run("synthetic frame", func(t *testing.T) {
		function, file, line = synthFrame.Location()
		testutils.AssertMatch(t, ".+MethodName.func1", function)
		testutils.AssertMatch(t, ".+e/errors_example_test.go", file)
		testutils.AssertEqual(t, 44, line)

		synthPCer, ok := synthFrame.(pcer)
		testutils.AssertTrue(t, ok)
		testutils.AssertEqual(t, uintptr(0), synthPCer.PC())
	})
}

func TestFrameFormat(t *testing.T) {
	var cases = []struct {
		name   string
		pc     errors.Frame
		format string
		expect string
	}{
		{"empty %s", emptyFrame, "%s", `^unknown$`},
		{"rtime %s", rtimeFrame, "%s", `^frames_test.go:40$`},
		{"synth %s", synthFrame, "%s", `^errors_example_test.go:44$`},
		{"empty %q", emptyFrame, "%q", `^"unknown"$`},
		{"rtime %q", rtimeFrame, "%q", `^"frames_test.go:40"$`},
		{"synth %q", synthFrame, "%q", `^"errors_example_test.go:44"$`},
		{"empty %d", emptyFrame, "%d", `0`},
		{"rtime %d", rtimeFrame, "%d", `40`},
		{"synth %d", synthFrame, "%d", `44`},
		{"empty %n", emptyFrame, "%n", `unknown$`},
		{"rtime %n", rtimeFrame, "%n", `^init$`},
		{"synth %n", synthFrame, "%n", `^\(\*ExampleStruct\)\.MethodName\.func1$`},
		{
			"ptr method %n",
			func() errors.Frame {
				var cs *FrameCallerStruct
				return cs.PtrFrameCaller()
			}(),
			"%n",
			`\(\*FrameCallerStruct\)\.PtrFrameCaller`,
		},
		{
			"val method %n",
			func() errors.Frame {
				var cs FrameCallerStruct
				return cs.ValFrameCaller()
			}(),
			"%n",
			`FrameCallerStruct\.ValFrameCaller`,
		},
		{"empty %v", emptyFrame, "%v", `^unknown$`},
		{"rtime %v", rtimeFrame, "%v", `.+/frames_test.go:40$`},
		{"synth %v", synthFrame, "%v", `^/usr/u/src/github\.com/secureworks/e/errors_example_test.go:44$`},
		{"empty %+v", emptyFrame, "%+v", `^unknown\n\tunknown:0$`},
		{
			"rtime %+v",
			rtimeFrame,
			"%+v",
			`^github\.com/secureworks/errors_test\.init\n\t` +
				`.+/frames_test\.go:40$`,
		},
		{
			"synth %+v",
			synthFrame,
			"%+v",
			`^github\.com/secureworks/errors_test\.\(\*ExampleStruct\)\.MethodName\.func1\n\t` +
				`/usr/u/src/github\.com/secureworks/e/errors_example_test\.go:44$`,
		},
		{
			"empty %#v",
			emptyFrame,
			"%#v",
			`^errors.Frame\("unknown"\)$`,
		},
		{
			"rtime %#v",
			rtimeFrame,
			"%#v",
			`^errors.Frame\(".+/frames_test\.go:40"\)$`,
		},
		{
			"synth %#v",
			synthFrame,
			"%#v",
			`^errors.Frame\("/usr/u/src/github\.com/secureworks/e/errors_example_test\.go:44"\)$`,
		},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			testutils.AssertMatch(t, tt.expect, fmt.Sprintf(tt.format, tt.pc))
		})
	}
}

func TestFrameMarshalJSON(t *testing.T) {
	var cases = []struct {
		name string
		errors.Frame
		expectJSONObject map[string]string
	}{{
		"runtime",
		rtimeFrame,
		map[string]string{
			"function": `^github\.com/secureworks/errors_test\.init$`,
			"file":     `^.+/frames_test.go$`,
			"line":     `^40$`,
		},
	}, {
		"synthetic",
		synthFrame,
		map[string]string{
			"function": `^github\.com/secureworks/errors_test\.\(\*ExampleStruct\)\.MethodName\.func1$`,
			"file":     `^.+github\.com/secureworks/e/errors_example_test.go$`,
			"line":     `^44$`,
		},
	}, {
		"partial",
		errors.NewFrame("runtime.doInit", "", 0),
		map[string]string{
			"function": `^runtime\.doInit$`,
			"file":     `^unknown$`,
			"line":     `^0$`,
		},
	}, {
		"empty",
		errors.FrameFromPC(0),
		map[string]string{
			"function": `^unknown$`,
			"file":     `^unknown$`,
			"line":     `^0$`,
		},
	}}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			byt, err := json.Marshal(tt.Frame)
			testutils.AssertNil(t, err)

			// Ensures the JSON is parsable.
			var parsed map[string]interface{}
			err = json.Unmarshal(byt, &parsed)
			testutils.AssertNil(t, err)

			// Ensures the JSON fields are same as expected.
			testutils.AssertEqual(t, len(tt.expectJSONObject), len(parsed))
			for k, v := range parsed {
				if n, ok := v.(float64); ok { // Line number.
					v = fmt.Sprintf("%d", int(n))
				}
				testutils.AssertMatch(t, tt.expectJSONObject[k], v.(string),
					fmt.Sprintf("key: %q", k))
			}
		})
	}
}

func TestPCFromFrame(t *testing.T) {
	testutils.AssertTrue(t, rtimeFramePC > 0)
	testutils.AssertEqual(t, rtimeFramePC, errors.PCFromFrame(rtimeFrame))

	// Get a frame from the std lib runtime.
	fr := runtime.GetFrame(1)
	testutils.AssertTrue(t, fr.PC > 0)
	testutils.AssertEqual(t, fr.PC, errors.PCFromFrame(fr))

	// PC identity.
	pfr := uintptr(1789100)
	testutils.AssertEqual(t, pfr, errors.PCFromFrame(pfr))
}

func TestFramesFormat(t *testing.T) {
	var cases = []struct {
		name string
		errors.Frames
		format string
		match  string
	}{
		{"empty %s", nil, "%s", `\[\]`},
		{"zero %s", make(errors.Frames, 0), "%s", `\[\]`},
		{
			"default %s",
			rtimeStack,
			"%s",
			`^\[frames_test\.go:30 frames_test\.go:47 proc\.go:\d+ proc\.go:\d+ proc\.go:\d+\]$`, // FIXME
		},

		{"empty %+s", nil, "%+s", `\[\]`},
		{"zero %+s", make(errors.Frames, 0), "%+s", `\[\]`},
		{
			"default %+s",
			rtimeStack,
			"%+s",
			`^\[frames_test\.go:30 frames_test\.go:47 proc\.go:\d+ proc\.go:\d+ proc\.go:\d+\]$`,
		},

		{"empty %n", nil, "%n", `\[\]`},
		{"zero %n", make(errors.Frames, 0), "%n", `\[\]`},
		{
			"default %n",
			rtimeStack,
			"%n",
			`^\[StackCaller init doInit doInit main\]$`,
		},

		{"empty %v", nil, "%v", `\[\]`},
		{"zero %v", make(errors.Frames, 0), "%v", `\[\]`},
		{
			"default %v",
			rtimeStack,
			"%v",
			`^\[.+/frames_test\.go:30 ` +
				`.+/frames_test\.go:47 ` +
				`.+src/runtime/proc\.go:\d+ .+src/runtime/proc\.go:\d+ .+src/runtime/proc\.go:\d+\]$`,
		},

		{"empty %+v", nil, "%+v", ``},
		{"zero %+v", make(errors.Frames, 0), "%+v", ``},
		{
			"default %+v",
			rtimeStack,
			"%+v",
			`^
github\.com/secureworks/errors_test.StackCaller
	.+/frames_test.go:30
github\.com/secureworks/errors_test.init
	.+/frames_test.go:47
runtime\.doInit
	.+/runtime/proc.go:\d+
runtime\.doInit
	.+/runtime/proc.go:\d+
runtime\.main
	.+/runtime/proc.go:\d+$`,
		},

		{"empty %#v", nil, "%#v", `errors\.Frames\{\}`},
		{"zero %#v", make(errors.Frames, 0), "%#v", `errors\.Frames\{\}`},
		{
			"default %#v",
			rtimeStack,
			"%#v",
			`^errors.Frames{frames_test\.go:30 frames_test\.go:47 proc\.go:\d+ proc\.go:\d+ proc\.go:\d+}$`,
		},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			testutils.AssertMatch(t, tt.match, fmt.Sprintf(tt.format, tt.Frames))
		})
	}
}

func TestFramesMarshalJSON(t *testing.T) {
	partialFrame := errors.NewFrame("runtime.doInit", "", 0)
	frames := errors.Frames{rtimeFrame, errors.FrameFromPC(0), synthFrame, partialFrame}

	byt, err := json.Marshal(frames)
	testutils.AssertNil(t, err)

	var parsed []json.RawMessage
	err = json.Unmarshal(byt, &parsed)
	testutils.AssertNil(t, err)

	// Just assure that we get an array of the JSON output from the frame
	// marshaler.
	testutils.AssertEqual(t, 4, len(parsed))
	for i, frBytes := range parsed {
		byt, err = json.Marshal(frames[i])
		testutils.AssertNil(t, err)
		testutils.AssertEqual(t, string(byt), string(frBytes))
	}

	t.Run("when empty", func(t *testing.T) {
		byt, err := json.Marshal((errors.Frames)(nil))
		testutils.AssertNil(t, err)
		testutils.AssertEqual(t, "null", string(byt))

		byt, err = json.Marshal(errors.Frames{})
		testutils.AssertNil(t, err)
		testutils.AssertEqual(t, "null", string(byt))
	})
}

func TestFramesFromBytes(t *testing.T) {
	expects := []struct {
		function string
		file     string
		line     int
	}{
		{
			"github.com/secureworks/errors/errors_test.(*ExampleStruct).MethodName.func1",
			"/usr/u/src/github.com/secureworks/e/errors_example_test.go",
			48,
		},
		{
			"github.com/secureworks/errors/errors_test.(*ExampleStruct).MethodName",
			"/usr/u/src/github.com/secureworks/e/errors_example_test.go",
			43,
		},
		{
			"unknown",
			"unknown",
			0,
		},
		{
			"github.com/secureworks/errors/errors_test.init",
			"/usr/u/src/github.com/secureworks/e/errors_example_test.go",
			7,
		},
		{
			"runtime.doInit",
			"unknown",
			0,
		},
	}
	t.Run("well formatted trace-only output", func(t *testing.T) {
		trace := []byte(strings.TrimSpace(`
github.com/secureworks/errors/errors_test.(*ExampleStruct).MethodName.func1
	/usr/u/src/github.com/secureworks/e/errors_example_test.go:48
github.com/secureworks/errors/errors_test.(*ExampleStruct).MethodName
	/usr/u/src/github.com/secureworks/e/errors_example_test.go:43
unknown
	unknown:0
github.com/secureworks/errors/errors_test.init
	/usr/u/src/github.com/secureworks/e/errors_example_test.go:7
runtime.doInit
	unknown:0`) + "\n")
		frames, err := errors.FramesFromBytes(trace)
		testutils.AssertNil(t, err)

		testutils.AssertEqual(t, 5, len(frames))
		for i, exp := range expects {
			function, file, line := frames[i].Location()
			testutils.AssertEqual(t, exp.function, function, fmt.Sprintf("frame %d: function", i))
			testutils.AssertEqual(t, exp.file, file, fmt.Sprintf("frame %d: file", i))
			testutils.AssertEqual(t, exp.line, line, fmt.Sprintf("frame %d: line", i))
		}
	})

	t.Run("with message context and whitespace padding", func(t *testing.T) {
		trace := []byte(`
 err: this is a wrapped context: this is an error message: basic error (code-45334)
github.com/secureworks/errors/errors_test.(*ExampleStruct).MethodName.func1
	/usr/u/src/github.com/secureworks/e/errors_example_test.go:48
github.com/secureworks/errors/errors_test.(*ExampleStruct).MethodName
	/usr/u/src/github.com/secureworks/e/errors_example_test.go:43
unknown
	unknown:0
github.com/secureworks/errors/errors_test.init
	/usr/u/src/github.com/secureworks/e/errors_example_test.go:7
runtime.doInit
	unknown:0

        `)
		frames, err := errors.FramesFromBytes(trace)
		testutils.AssertNil(t, err)

		testutils.AssertEqual(t, 5, len(frames))
		for i, exp := range expects {
			function, file, line := frames[i].Location()
			testutils.AssertEqual(t, exp.function, function, fmt.Sprintf("frame %d: function", i))
			testutils.AssertEqual(t, exp.file, file, fmt.Sprintf("frame %d: file", i))
			testutils.AssertEqual(t, exp.line, line, fmt.Sprintf("frame %d: line", i))
		}
	})
}

func TestFramesFromJSON(t *testing.T) {
	byt := []byte(`[
	{
		"function":"github.com/secureworks/errors/errors_test.init",
		"file":"/Users/uname/code/github.com/secureworks/errors/frames_test.go",
		"line":44
	},
	{
		"function":"unknown",
		"line":0,
		"file":"unknown"
	},
	{},
	{
		"file":"/usr/u/src/github.com/secureworks/e/errors_example_test.go",
		"line":48
	},
	{
		"function":"github.com/secureworks/errors/errors_test.(*ExampleStruct).MethodName.func1"
	}
]`)
	fr0 := errors.NewFrame(
		"github.com/secureworks/errors/errors_test.init",
		"/Users/uname/code/github.com/secureworks/errors/frames_test.go",
		44,
	)
	fr1 := errors.NewFrame("", "", 0)
	fr2 := errors.NewFrame("", "", 0)
	fr3 := errors.NewFrame(
		"", "/usr/u/src/github.com/secureworks/e/errors_example_test.go", 48)
	fr4 := errors.NewFrame(
		"github.com/secureworks/errors/errors_test.(*ExampleStruct).MethodName.func1", "", 0)

	parsed, err := errors.FramesFromJSON(byt)
	testutils.AssertNil(t, err)

	// Test each entry.
	testutils.AssertEqual(t, 5, len(parsed))
	testutils.AssertEqual(t, fmt.Sprintf("%+v", fr0), fmt.Sprintf("%+v", parsed[0]))
	testutils.AssertEqual(t, fmt.Sprintf("%+v", fr1), fmt.Sprintf("%+v", parsed[1]))
	testutils.AssertEqual(t, fmt.Sprintf("%+v", fr2), fmt.Sprintf("%+v", parsed[2]))
	testutils.AssertEqual(t, fmt.Sprintf("%+v", fr3), fmt.Sprintf("%+v", parsed[3]))
	testutils.AssertEqual(t, fmt.Sprintf("%+v", fr4), fmt.Sprintf("%+v", parsed[4]))

	t.Run("when null", func(t *testing.T) {
		ff, err := errors.FramesFromJSON([]byte("null"))
		testutils.AssertNil(t, err)
		testutils.AssertEqual(t, 0, len(ff))
	})
}

func TestFrameEscapes(t *testing.T) {
	// Use big characters in there to ensure we are rune-aware.
	frFunction := "example.com/_\" Poorly\tNamed\"/可口可乐/path to a\n\npackage\\name/pkg.(欢迎地图).Funčtįøñ"
	frFile := "/Example /_\" Poorly\tNamed\"/path\t\"to\"\\ a\n\n文件.exe"
	fr := errors.NewFrame(
		frFunction,
		frFile,
		10,
	)

	t.Run("formatting", func(t *testing.T) {
		testutils.AssertEqual(t, `path\t\"to\"\\ a\n\n文件.exe:10`,
			fmt.Sprintf("%s", fr))
		testutils.AssertEqual(t, `(欢迎地图).Funčtįøñ`,
			fmt.Sprintf("%n", fr))
		testutils.AssertEqual(t, `/Example /_\" Poorly\tNamed\"/path\t\"to\"\\ a\n\n文件.exe:10`,
			fmt.Sprintf("%v", fr))
		testutils.AssertEqual(t, `example.com/_\" Poorly\tNamed\"/可口可乐/path to a\n\npackage\\name/pkg.(欢迎地图).Funčtįøñ
	/Example /_\" Poorly\tNamed\"/path\t\"to\"\\ a\n\n文件.exe:10`,
			fmt.Sprintf("%+v", fr))
		testutils.AssertEqual(t, `errors.Frame("/Example /_\" Poorly\tNamed\"/path\t\"to\"\\ a\n\n文件.exe:10")`,
			fmt.Sprintf("%#v", fr))

	})

	t.Run("marshal and unmarshal JSON", func(t *testing.T) {
		byt, err := errors.Frames([]errors.Frame{fr}).MarshalJSON()
		testutils.AssertNil(t, err)
		testutils.AssertEqual(t,
			`[{"function":"example.com/_\\\" Poorly\\tNamed\\\"/可口可乐/path to a\\n\\npackage\\\\name/pkg.(欢迎地图).Funčtįøñ","file":"/Example /_\\\" Poorly\\tNamed\\\"/path\\t\\\"to\\\"\\\\ a\\n\\n文件.exe","line":10}]`,
			string(byt))

		ff, err := errors.FramesFromJSON(byt)
		testutils.AssertNil(t, err)
		testutils.AssertEqual(t, 1, len(ff))

		function, file, line := ff[0].Location()
		testutils.AssertEqual(t, frFunction, function)
		testutils.AssertEqual(t, frFile, file)
		testutils.AssertEqual(t, 10, line)
	})
}

// These helpers are copies of the internal getStack and getFrame: do
// not want to export these, they are dead simple, and we want to run
// tests in an isolated package.

//go:noinline
func getStack() errors.Frames {
	st := runtime.GetStack(2) // Skip runtime.GetStack.
	ff := make([]errors.Frame, len(st))
	for i, fr := range st {
		ff[i] = errors.FrameFromPC(fr.PC)
	}
	return ff
}

//go:noinline
func getFrame() errors.Frame {
	return errors.FrameFromPC(runtime.GetFrame(2).PC) // Skip runtime.GetFrame and errors_test.getFrame.
}
