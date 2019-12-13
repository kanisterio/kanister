package caller

import (
	"runtime"
	"strings"
)

// Frame contains information about the caller. This is a subset of the fields
// in runtime.Frame.
type Frame struct {
	Function string
	File     string
	Line     int
}

// GetFrame returns information about a caller function at the specified depth
// above this function in the the call stack.
func GetFrame(depth int) Frame {
	// we get the callers as uintptrs - but we just need 1
	fpcs := make([]uintptr, 1)

	// Skip depth + 1 frames to get to the desired caller.
	num := runtime.Callers(depth+1, fpcs)
	if num != 1 {
		// Failure potentially due to wrongly specified depth
		return Frame{Function: "Unknown", File: "Unknown", Line: 0}
	}
	frames := runtime.CallersFrames(fpcs[:num])

	var frame runtime.Frame
	frame, _ = frames.Next()
	filename := frame.File
	if paths := strings.SplitAfterN(frame.File, "/go/src/", 2); len(paths) > 1 {
		filename = paths[1]
	} else if paths := strings.SplitAfterN(frame.File, "/kanister/", 2); len(paths) > 1 {
		filename = paths[1]
	}
	return Frame{Function: frame.Function, File: filename, Line: frame.Line}
}
