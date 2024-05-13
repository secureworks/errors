package syncerr

import (
	"context"
	"fmt"
	"sort"
	"testing"
	"time"

	"github.com/secureworks/errors"
	"github.com/secureworks/errors/internal/testutils"
)

type errorType struct{}

func (m errorType) Error() string { return "i'm an error" }

type errorList []errorType

func (m errorList) Error() string {
	return fmt.Sprintf("i'm an error list x %d", len(m))
}

func TestCoordinatedGroup(t *testing.T) {
	err := errors.New("new err")

	// Problematic nils.
	var nilErrList errorList
	nilErr := (*errorType)(nil)

	cases := []struct {
		errs     []error
		expected error
	}{
		{expected: nil},
		{errs: []error{nil}, expected: nil},
		{errs: []error{nilErrList, nilErr}, expected: nil},
		{errs: []error{err}, expected: err},
		{errs: []error{err, nil}, expected: err},
		{errs: []error{nilErrList, err, nilErr}, expected: err},
	}

	for i, tc := range cases {
		octx := context.Background()
		group, ictx := NewCoordinatedGroup(octx)
		for _, err := range tc.errs {
			err := err
			group.Go(func() error { return err })
		}

		// Returns an error when one is found.
		err := group.Wait()
		testutils.AssertEqual(t, tc.expected, err, fmt.Sprintf("case %d", i))

		cancelled := false
		select {
		case <-ictx.Done():
			cancelled = true
		default:
		}
		testutils.AssertTrue(t, cancelled,
			fmt.Sprintf("case %d: inner context was not cancelled", i))

		cancelled = false
		select {
		case <-octx.Done():
			cancelled = true
		default:
		}
		testutils.AssertFalse(t, cancelled,
			fmt.Sprintf("case %d: outer context was cancelled", i))
	}
}

func TestCoordinatedGroup_ZeroValue(t *testing.T) {
	err1 := errors.New("new err: 1")
	err2 := errors.New("new err: 2")

	// Problematic nils.
	var nilErrList errorList
	nilErr := (*errorType)(nil)

	cases := []struct {
		errs   []error
		hasErr bool
	}{
		{errs: []error{}},
		{errs: []error{nil}},
		{errs: []error{nilErr, nilErr}},
		{errs: []error{err1}, hasErr: true},
		{errs: []error{err1, nil}, hasErr: true},
		{errs: []error{err1, nil, err2}, hasErr: true},
		{errs: []error{nil, err1, err2}, hasErr: true},
		{errs: []error{err1, nilErrList, err2}, hasErr: true},
		{errs: []error{nilErr, nilErrList, err1}, hasErr: true},
	}

	for i, tc := range cases {
		group := new(CoordinatedGroup)

		for j, err := range tc.errs {
			err := err
			time.Sleep(time.Duration(int64(time.Millisecond) * int64(j)))
			group.Go(func() error { return err })
		}

		gErr := group.Wait()
		if tc.hasErr {
			testutils.AssertEqual(t, err1, gErr, fmt.Sprintf("case %d:", i))
		} else {
			testutils.AssertEqual(t, nil, gErr, fmt.Sprintf("case %d:", i))
		}

	}
}

func TestParallelGroup(t *testing.T) {
	err1 := errors.New("new err: 1")
	err2 := errors.New("new err: 2")

	// Problematic nils.
	var nilErrList errorList
	nilErr := (*errorType)(nil)

	cases := []struct {
		errs []error
	}{
		{errs: []error{}},
		{errs: []error{nil}},
		{errs: []error{nilErrList}},
		{errs: []error{err1}},
		{errs: []error{err1, nil}},
		{errs: []error{err1, nil, err2, nil}},
		{errs: []error{err1, nilErr, err2, nilErr}},
	}

	for i, tc := range cases {
		group := new(ParallelGroup)

		var taskErrors []error
		for _, err := range tc.errs {
			err := err
			group.Go(func() error { return err })
			if err == err1 || err == err2 {
				taskErrors = append(taskErrors, err)
			}
		}

		err := group.Wait()
		merr := group.WaitForMultiError()
		testutils.AssertEqual(t, err, merr,
			fmt.Sprintf("case %d: Wait == WaitForMultiError", i))

		expected := sortedMessages(taskErrors)
		actual := sortedMessages(merr.Errors())
		testutils.AssertEqual(t,
			expected, actual, fmt.Sprintf("case %d: expected errors", i))
	}
}

func TestCoordinatedGroup_WrapName(t *testing.T) {
	err1 := errors.New("new err")

	group := new(CoordinatedGroup)

	for _, err := range []error{nil, err1, nil} {
		err := err
		group.Go(func() error { return err }, "worker")
	}

	err := group.Wait()
	testutils.AssertEqual(t, "worker: new err", err.Error())
}

func TestParallelGroup_WrapName(t *testing.T) {
	err1 := errors.New("new err: 1")
	err2 := errors.New("new err: 2")

	group := new(ParallelGroup)

	for i, err := range []error{err1, nil, nil, err2} {
		err := err
		group.Go(func() error { return err }, fmt.Sprintf("worker %d", i))
	}

	merr := group.WaitForMultiError()
	testutils.AssertEqual(t,
		[]string{"worker 0: new err: 1", "worker 3: new err: 2"}, sortedMessages(merr.Errors()))
}

func sortedMessages(errs []error) (msgs []string) {
	msgs = make([]string, len(errs))
	for i, err := range errs {
		msgs[i] = err.Error()
	}
	sort.Strings(msgs)
	return
}
