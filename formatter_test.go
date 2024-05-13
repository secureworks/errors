package errors

import (
	"fmt"
	"testing"

	"github.com/secureworks/errors/internal/testutils"
)

func TestErrorf(t *testing.T) {
	t.Run("wraps with message context", func(t *testing.T) {
		err := Errorf("wraps: %w", newErrorCaller())
		testutils.AssertEqual(t, `wraps: new err`, fmt.Sprint(err))
	})

	t.Run("wraps with frame context", func(t *testing.T) {
		err := Errorf("wraps: %w", newErrorCaller())
		withFrames, ok := err.(interface{ Frames() Frames })
		testutils.AssertTrue(t, ok)
		testutils.AssertLinesMatch(t, withFrames.Frames(), "%+v", []string{
			"",
			"^github.com/secureworks/errors\\.TestErrorf.func2$",
			"^\t.+/formatter_test\\.go:\\d+$",
		})
	})

	t.Run("handles variant params", func(t *testing.T) {
		err := Errorf("wraps: %[2]s (%[3]d): %[1]w", newErrorCaller(), "inner", 1)
		testutils.AssertErrorMessage(t, "wraps: inner (1): new err", err)
		_, ok := err.(interface{ Frames() Frames })
		testutils.AssertTrue(t, ok)
	})
}
