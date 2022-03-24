package runtime

import (
	"os"
	"runtime"
	"strings"
)

func GetStack(skip int) []runtime.Frame {
	var pcs [32]uintptr
	frames, n := callers(skip, pcs[:])
	ff := make([]runtime.Frame, 0, n)
	for {
		fr, ok := frames.Next()
		if !ok {
			break
		}
		ff = append(ff, fr)
	}
	return ff
}

func GetFrame(skip int) runtime.Frame {
	var pcs [3]uintptr
	frames, _ := callers(skip, pcs[:])
	fr, ok := frames.Next()
	if !ok {
		return runtime.Frame{}
	}
	return fr
}

func FuncName(name string) string {
	i := strings.LastIndex(name, string(os.PathSeparator))
	name = name[i+1:]
	i = strings.Index(name, ".")
	return name[i+1:]
}

//go:noinline
func callers(skip int, pcs []uintptr) (frames *runtime.Frames, n int) {
	n = runtime.Callers(skip+1, pcs)
	frames = runtime.CallersFrames(pcs[:n])
	if _, ok := frames.Next(); !ok {
		return &runtime.Frames{}, 0
	}
	return
}
