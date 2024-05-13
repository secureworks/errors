package errors

import (
	"fmt"
	"regexp"
)

var errorfFormatMatcher = regexp.MustCompile(`%(\[\d+])?w`)

// Errorf is a shorthand for:
//
//	errors.WithFrame(fmt.Errorf("some msg: %w", err))
//
// It is made available to support the best practice of adding a call
// stack frame to the error context alongside a message when building a
// chain. When possible, prefer using the full syntax instead of this
// shorthand for clarity.
//
// Using an invalid format string (one that does not wrap the given
// error) causes this method to panic.
func Errorf(format string, values ...interface{}) error {
	if !errorfFormatMatcher.MatchString(format) {
		panic(NewWithStackTrace(fmt.Sprintf("invalid use of errors.Errorf: "+
			"format string must wrap an error, but \"%%w\" not found: %q", format)))
	}
	return &withFrames{
		error:  fmt.Errorf(format, values...),
		frames: frames{getFrame(3)},
	}
}
