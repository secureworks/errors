package errors

// Attribution: portions of the below code and documentation are modeled
// directly on the https://pkg.go.dev/golang.org/x/xerrors library, used
// with the permission available under the software license
// (BSD 3-Clause):
// https://cs.opensource.google/go/x/xerrors/+/master:LICENSE
//
// Attribution: portions of the below code and documentation are modeled
// directly on the https://github.com/pkg/errors library, used
// with the permission available under the software license
// (BSD 2-Clause):
// https://github.com/pkg/errors/blob/master/LICENSE

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"
	stdruntime "runtime"
	"strconv"
	"strings"

	"github.com/secureworks/errors/internal/runtime"
)

// Frame defines an interface for accessing and displaying stack frame
// information for debugging, optimizing or inspection. Usually you will
// find Frame in a Frames slice, acting as a stack trace or stack dump.
//
// Frames are meant to be seen, so we have implemented the following
// default formatting verbs on it:
//
//	"%s"  – the base name of the file (or `unknown`) and the line number (if known)
//	"%q"  – the same as `%s` but wrapped in `"` delimiters
//	"%d"  – the line number
//	"%n"  – the basic function name, ie without a full package qualifier
//	"%v"  – the full path of the file (or `unknown`) and the line number (if known)
//	"%+v" – a standard line in a stack trace: a full function name on one line,
//	        and a full file name and line number on a second line
//	"%#v" – a Golang representation with the type (`errors.Frame`)
//
// Marshaling a frame as text uses the `%+v` format.
// Marshaling as JSON returns an object with location data:
//
//	{"function":"test.pkg.in/example.init","file":"/src/example.go","line":10}
//
// A Frame is immutable, so no setters are provided, but you can copy
// one trivially with:
//
//	function, file, line := oldFrame.Location()
//	newFrame := errors.NewFrame(function, file, line)
type Frame interface {
	// Location returns the frame's caller's characteristics for help with
	// identifying and debugging the codebase.
	//
	// Location results are generated uniquely per Frame implementation.
	// When using this package's implementation, note that the results are
	// evaluated and expanded lazily when the frame was generated from the
	// local call stack: Location is not safe for concurrent access.
	Location() (function string, file string, line int)
}

// programCounter defines an interface for extracting a program counter
// on the call stack from a frame type. The absence of a program counter
// (when it is 0) means it was generated synthetically.
type programCounter interface {
	PC() uintptr
}

// frame is this package's default implementation of Frame in such a way
// that we can create one either from the actual call stack or
// "synthetically:" by parsing a stack trace or even specifically
// designating the location characteristics. frame also implements
// interfaces to integrate with runtime (via program counters) and
// serialization and deserialization processes.
type frame struct {
	pc        uintptr
	runtimeFn *stdruntime.Func
	function  string
	file      string
	line      int
}

var _ interface { // Assert interface implementation.
	Frame
	programCounter
	fmt.Formatter
	json.Marshaler
} = (*frame)(nil)

// PC returns the Frame's local frame program counter.
func (f *frame) PC() uintptr { return f.pc }

// Location returns the frame's caller's characteristics for help with
// identifying and debugging the codebase.
//
// The results are evaluated and expanded lazily when the frame was
// generated from the local call stack: Location is not safe for
// concurrent access.
func (f *frame) Location() (function string, file string, line int) {
	return f.getFunction(), f.getFile(), f.getLine()
}

// Format gives this interface control over how the location information
// is structured when it is displayed. Including it in the interface
// ensures that a stack of Frames can structure how the entire stack is
// displayed.
func (f *frame) Format(s fmt.State, verb rune) {
	var appendD = func(line int) {
		if line > 0 {
			io.WriteString(s, ":")
			io.WriteString(s, strconv.Itoa(line))
		}
	}
	var formatS = func(file string, line int) {
		io.WriteString(s, escaper.Replace(filepath.Base(file)))
		appendD(line)
	}

	// FIXME(PH): does not handle Windows paths correctly, which means
	//   it's likely that we can't ensure Windows stack traces are formatted
	//   correctly, and we definitely can't deserialize Windows stack traces
	//   on a non-Windows system.

	function, file, line := f.Location()
	switch verb {
	case 's':
		formatS(file, line)
	case 'q':
		io.WriteString(s, `"`)
		formatS(file, line)
		io.WriteString(s, `"`)
	case 'd':
		io.WriteString(s, strconv.Itoa(line))
	case 'n':
		io.WriteString(s, escaper.Replace(runtime.FuncName(function)))
	case 'v':
		switch {
		case s.Flag('+'):
			prefix := ""
			width, ok := s.Width()
			if ok {
				prefix = strings.Repeat(" ", width)
			}
			io.WriteString(s, prefix)
			io.WriteString(s, escaper.Replace(function))
			io.WriteString(s, "\n"+prefix+"\t")
			io.WriteString(s, escaper.Replace(file))
			io.WriteString(s, ":")
			io.WriteString(s, strconv.Itoa(line))
		case s.Flag('#'):
			io.WriteString(s, "errors.Frame(\"")
			io.WriteString(s, escaper.Replace(file))
			appendD(line)
			io.WriteString(s, "\")")
		default:
			io.WriteString(s, escaper.Replace(file))
			appendD(line)
		}
	}
}

// MarshalJSON allows this interface to integrate its default formatting
// into JSON for serialization.
func (f frame) MarshalJSON() ([]byte, error) {
	function, file, line := f.Location()
	str := fmt.Sprintf(`{"function":%q,"file":%q,"line":%d}`,
		escaper.Replace(function), escaper.Replace(file), line)
	return []byte(str), nil
}

// escaper escapes some characters that will keep a stack trace from
// being parsable / deserializable.
var escaper = strings.NewReplacer(`\`, `\\`, "\t", `\t`, "\n", `\n`, `"`, `\"`)

// unescaper unescapes characters on deserialization.
var unescaper = strings.NewReplacer(`\t`, "\t", `\n`, "\n", `\"`, `"`, `\\`, `\`)

// getFunction gets the frame's full caller function name. Prioritizes
// synthetic values if available, otherwise expands the pc using runtime
// and memoizes the result.
func (f *frame) getFunction() (function string) {
	function = f.function
	if function == "" {
		function = "unknown"
		if f.pc != 0 {
			function = f.fn().Name()
			f.function = function
		}
	}
	return
}

// getFile gets the frame's caller's filename. Prioritizes synthetic
// values if available, otherwise expands the pc using runtime and
// memoizes the result.
func (f *frame) getFile() (file string) {
	file = f.file
	if file == "" {
		file = "unknown"
		if f.pc != 0 {
			file, _ = f.fn().FileLine(f.pc)
			f.file = file
		}
	}
	return
}

// getLine gets the frame's caller's file line. Prioritizes synthetic
// values if available, otherwise expands the pc using runtime and
// memoizes the result.
func (f *frame) getLine() (line int) {
	line = f.line
	if line == 0 {
		if f.pc != 0 {
			_, line = f.fn().FileLine(f.pc)
			f.line = line
		}
	}
	return
}

// fn is the way to cleanly access the runtimeFn field: if none is found
// it attempts to look it up from the frame location program counter
// (pc). This lookup will only happen once.
func (f *frame) fn() *stdruntime.Func {
	if f.runtimeFn == nil && f.pc != 0 {
		f.runtimeFn = stdruntime.FuncForPC(f.pc)
	}
	return f.runtimeFn
}

// NewFrame creates a "synthetic" Frame that describes the given
// location characteristics. This can be used to deserialize stack
// traces or stack dumps, or write clear tests that work with these.
func NewFrame(function string, file string, line int) Frame {
	return &frame{
		function: function,
		file:     file,
		line:     line,
	}
}

// FrameFromPC creates a Frame from a program counter.
func FrameFromPC(pc uintptr) Frame {
	return frameFromPC(pc)
}

// PCFromFrame extracts the frame location program counter (pc) from
// either this package's Frame implementation (using an unexported
// interface), a raw uintptr (for identity), or runtime.Frame. Does not
// distinguish between an empty or nil frame, an unsupported frame
// implementation, or some other error: all return 0.
func PCFromFrame(v interface{}) uintptr {
	if v == nil {
		return 0
	}
	switch fr := v.(type) {
	case uintptr:
		return fr
	case stdruntime.Frame:
		return fr.PC
	case programCounter:
		return fr.PC()
	default:
		return 0
	}
}

// frameFromPC creates a frame struct from a program counter.
func frameFromPC(pc uintptr) *frame {
	return &frame{pc: pc}
}

// newFrameFrom creates a frame struct from a Frame interface.
func newFrameFrom(fr Frame) *frame {
	function, file, line := fr.Location()
	return &frame{
		function: function,
		file:     file,
		line:     line,
	}
}

// Frames is a slice of Frame data. This can represent a stack trace or
// some subset of a stack trace.
type Frames []Frame

var _ interface { // Assert interface implementation.
	fmt.Formatter
	json.Marshaler
} = (Frames)(nil)

func (ff Frames) Format(s fmt.State, verb rune) {
	switch verb {
	case 's':
		ff.formatSlice(s, verb, [2]string{"[", "]"})
	case 'n':
		ff.formatSlice(s, verb, [2]string{"[", "]"})
	case 'v':
		switch {
		case s.Flag('+'):
			for _, f := range ff {
				io.WriteString(s, "\n")
				f.(fmt.Formatter).Format(s, verb)
			}
		case s.Flag('#'):
			io.WriteString(s, "errors.Frames")
			ff.formatSlice(s, 's', [2]string{"{", "}"})
		default:
			ff.formatSlice(s, verb, [2]string{"[", "]"})
		}
	}
}

func (ff Frames) MarshalJSON() ([]byte, error) {
	if len(ff) == 0 {
		return []byte("null"), nil
	}

	buf := new(bytes.Buffer)

	_, err := buf.Write([]byte(`[`))
	if err != nil {
		return nil, err
	}
	frBytes := make([][]byte, len(ff))
	for i, fr := range ff {
		frBytes[i], err = json.Marshal(fr)
		if err != nil {
			return nil, err
		}
	}
	_, err = buf.Write(bytes.Join(frBytes, []byte(`,`)))
	if err != nil {
		return nil, err
	}
	_, err = buf.Write([]byte(`]`))
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// formatSlice wraps a list of formatted frames with brackets.
func (ff Frames) formatSlice(s fmt.State, verb rune, delimiters [2]string) {
	io.WriteString(s, delimiters[0])
	for i, f := range ff {
		if i > 0 {
			io.WriteString(s, " ")
		}
		f.(fmt.Formatter).Format(s, verb)
	}
	io.WriteString(s, delimiters[1])
}

// FramesFromBytes parses a stack trace or stack dump provided as bytes
// into a stack of Frames. The format of the text is expected to match
// the output of printing with a formatter using the `%+v` verb.
func FramesFromBytes(byt []byte) (Frames, error) {
	rawFrames, err := framesFromBytes(byt)
	if err != nil {
		return nil, err
	}

	ff := make([]Frame, len(rawFrames))
	for i, fr := range rawFrames {
		ff[i] = fr
	}
	return ff, nil
}

// FramesFromJSON parses a stack trace or stack dump provided as
// JSON-encoded bytes into a stack of Frames. json. Unmarshal does not
// work because it is meant to marshal into pre-allocated items, where
// Frames are defined only as interfaces.
func FramesFromJSON(byt []byte) (Frames, error) {
	rawFrames, err := framesFromJSON(byt)
	if err != nil {
		return nil, err
	}

	ff := make([]Frame, len(rawFrames))
	for i, fr := range rawFrames {
		ff[i] = fr
	}
	return ff, nil
}

// framer defines an interface for accessing Frames, which can
// represent a stack trace or a subset of a stack trace. This is the
// preferred method for getting stack information in this package.
type framer interface {
	Frames() Frames
}

// stackTracer defines an interface for accessing a slice of `uintptr`s,
// which can be trivially converted to a
// `github.com/pkg/errors.StackTrace` (and will be completely
// interchangeable once we use Go 1.18 Generics), but also works where
// we are using reflection to handle the interaction since the slice
// item types are assignable to one another.
//
// See: https://github.com/getsentry/sentry-go/blob/v0.12.0/stacktrace.go#L81
type stackTracer interface {
	StackTrace() []uintptr
}

// frames stores a slice of frame structs and implements both the
// StackFrames and stackTracer interfaces.
type frames []*frame

var _ interface { // Assert interface implementation.
	stackTracer
	framer
	json.Marshaler
} = (frames)(nil)

// NOTE(PH): because we don't export the helper that generates frames
//   from the call stack, the tests for generated stacks (as opposed to
//   the Frames interface) are easier to run as part of the error test
//   suite.

// Frames implements the StackFrames interface, returning Frames.
func (ff frames) Frames() Frames {
	st := make([]Frame, 0, len(ff))
	for _, f := range ff {
		st = append(st, Frame(f))
	}
	return st
}

// StackTrace implements the stackTracer interface, returning a slice of
// program counters.
func (ff frames) StackTrace() []uintptr {
	st := make([]uintptr, len(ff))
	for i, f := range ff {
		st[i] = PCFromFrame(f)
	}
	return st
}

func (ff frames) MarshalJSON() ([]byte, error) {
	return ff.Frames().MarshalJSON()
}

var errIncompleteFrame = New("incomplete frame data")
var errMalformedFrame = New("missing frame data: function name must come first")

// framesFromBytes is the underlying text (stack trace dump) parser for
// creating synthetic frames. Expects the text to be formatted as if it
// were printed using the `%+v` verb: newlines are necessary for it to
// scan.
//
// Returns partially completed frames along with an error if one is
// encountered. Handles arbitrary leading and trailing whitespace, and
// allows for a single "error context line" with the printed message
// context prepended directly to the stack (it may not contain any
// newlines: *only one line allowed*).
func framesFromBytes(byt []byte) (rawFrames []*frame, err error) {
	byt = bytes.TrimSpace(byt)

	// Handle empty text.
	if len(byt) == 0 {
		return
	}

	// Check for prepended message context.
	firstNL := bytes.IndexByte(byt, '\n')
	firstNT := bytes.Index(byt, []byte("\n\t"))
	if firstNL > 0 && firstNT > 0 && firstNL != firstNT {
		byt = bytes.SplitN(byt, []byte{'\n'}, 2)[1]
	}

	index := 0
	lines := bytes.Split(byt, []byte{'\n'})
	for index+2 <= len(lines) {
		var line int64
		// Take next two lines, strip whitespace, and check for a colon in the
		// second line to split on: if exists, split off the line number.
		function := bytes.TrimSpace(lines[index])
		file := bytes.TrimSpace(lines[index+1])
		colonIdx := bytes.IndexByte(file, ':')
		if colonIdx > 0 {
			line, err = strconv.ParseInt(string(file[colonIdx+1:]), 10, 64)
			if err != nil {
				err = fmt.Errorf(
					"%w: %q: unparsable line number: %s",
					errMalformedFrame,
					string(lines[index+1]),
					err,
				)
				break
			}
			file = file[:colonIdx]
		}
		// Add the frame to the list and advance the index.
		rawFrames = append(rawFrames, &frame{
			function: unescaper.Replace(string(function)),
			file:     unescaper.Replace(string(file)),
			line:     int(line),
		})
		index += 2
	}

	// If lines don't line up, send incomplete error with frames.
	if index < len(lines) {
		err = fmt.Errorf("%w: %q", errIncompleteFrame, lines[index])
	}
	return
}

// framesFromJSON is the underlying JSON parser for creating synthetic
// frames from JSON.
func framesFromJSON(byt []byte) ([]*frame, error) {
	if string(byt) == "null" { // No-op by convention.
		return nil, nil
	}

	type frameUnmarshaler struct {
		Function string `json:"function"`
		File     string `json:"file"`
		Line     int    `json:"line"`
	}

	var rawFrames []frameUnmarshaler
	err := json.Unmarshal(byt, &rawFrames)
	if err != nil {
		return nil, err
	}

	frames := make([]*frame, len(rawFrames))
	for i, fr := range rawFrames {
		frames[i] = NewFrame(
			unescaper.Replace(fr.Function),
			unescaper.Replace(fr.File),
			fr.Line,
		).(*frame)
	}
	return frames, nil
}

// framesFromPCs turns a stack trace of program counters into Frames.
func framesFromPCs(pcs []uintptr) Frames {
	ff := make(Frames, len(pcs))
	for i, pc := range pcs {
		ff[i] = FrameFromPC(pc)
	}
	return ff
}
