package errors

import "github.com/secureworks/errors/internal/runtime"

// Caller returns a Frame that describes the proximate frame on the
// caller's stack.
func Caller() Frame {
	return getFrame(3)
}

// CallerAt returns a Frame that describes a frame on the caller's
// stack. The argument skipCaller is the number of frames to skip over.
func CallerAt(skipCallers int) Frame {
	return getFrame(skipCallers + 3)
}

// CallStack returns all the Frames that describe the caller's stack.
func CallStack() Frames {
	st := getStack(3)
	ff := make(Frames, len(st))
	for i, fr := range st {
		ff[i] = fr
	}
	return ff
}

// CallStackAt returns all the Frames that describe the caller's stack.
// The argument skipCaller is the number of frames to skip over.
func CallStackAt(skipCallers int) Frames {
	st := getStack(skipCallers + 3)
	ff := make(Frames, len(st))
	for i, fr := range st {
		ff[i] = fr
	}
	return ff
}

// CallStackAtMost returns a subset of Frames that describe the caller's
// stack. The argument skipCaller is the number of frames to skip over,
// and the argument maxFrames is the maximum number of frames to return
// (if the entire stack is less than maxFrames, the entireStack is
// returned). maxFrames of zero or fewer is ignored:
//
//	CallStackAtMost(0, 0) // ... returns the entire stack for the caller
func CallStackAtMost(skipCallers int, maxFrames int) Frames {
	st := getStack(skipCallers + 3)
	stackLen := len(st)
	if maxFrames > 0 && stackLen > maxFrames {
		stackLen = maxFrames
	}
	ff := make(Frames, stackLen)
	for i, fr := range st {
		if i == stackLen {
			break
		}
		ff[i] = fr
	}
	return ff
}

// getFrame translates a runtime.Frame item returned from the internal
// runtime utilities into a frame.
//
//go:noinline
func getFrame(skipCallers int) *frame {
	return &frame{pc: runtime.GetFrame(skipCallers).PC}
}

// getStack translates runtime.Frame items returned from the internal
// runtime utilities into frames.
//
//go:noinline
func getStack(skipCallers int) frames {
	st := runtime.GetStack(skipCallers)
	ff := make([]*frame, len(st))
	for i, fr := range st {
		ff[i] = &frame{pc: fr.PC}
	}
	return ff
}
