package errors

import (
	stderrors "errors"
	"fmt"
	"testing"

	"github.com/secureworks/errors/internal/testutils"
)

type customErr struct {
	msg string
}

func (c customErr) Error() string { return c.msg }

func TestNew(t *testing.T) {
	libErr := New("new err")
	stdErr := stderrors.New("new err")

	testutils.AssertNotNil(t, libErr)
	testutils.AssertEqual(t, stdErr, libErr)
	testutils.AssertEqual(t, stdErr.Error(), libErr.Error())
}

func TestUnwrap(t *testing.T) {
	err := New("new err")

	type args struct {
		err error
	}
	cases := []struct {
		name string
		args args
		want error
	}{
		{
			name: "with stack",
			args: args{err: WithStackTrace(err)},
			want: err,
		},
		{
			name: "with frame",
			args: args{err: WithFrame(err)},
			want: err,
		},
		{
			name: "with frames",
			args: args{err: WithFrames(err, Frames{})},
			want: err,
		},
		{
			name: "with message",
			args: args{err: WithMessage(err, "replace err")},
			want: err,
		},
		{
			name: "std errors compatibility",
			args: args{err: fmt.Errorf("wrap: %w", err)},
			want: err,
		},
		{
			name: "unwrapped is nil",
			args: args{err: err},
			want: nil,
		},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			testutils.AssertEqual(t, tt.want, Unwrap(tt.args.err))
		})
	}
}

func TestIs(t *testing.T) {
	err := New("new err")
	err2 := New("signal error")

	type args struct {
		err    error
		target error
	}
	cases := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "with stack",
			args: args{
				err:    WithStackTrace(err),
				target: err,
			},
			want: true,
		},
		{
			name: "with frame",
			args: args{
				err:    WithFrame(err),
				target: err,
			},
			want: true,
		},
		{
			name: "with frames",
			args: args{
				err:    WithFrames(err, Frames{}),
				target: err,
			},
			want: true,
		},
		{
			name: "with message",
			args: args{
				err:    WithMessage(err, "replace err"),
				target: err,
			},
			want: true,
		},
		{
			name: "std errors compatibility",
			args: args{
				err:    fmt.Errorf("wrap: %w", err),
				target: err,
			},
			want: true,
		},
		{
			name: "std errors compatibility (false)",
			args: args{
				err:    fmt.Errorf("not wrap: %s", err),
				target: err,
			},
			want: false,
		},
		{
			name: "std errors multierror compatibility",
			args: args{
				err:    fmt.Errorf("wrap: %w; %w", err, err2),
				target: err2,
			},
			want: true,
		},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			testutils.AssertEqual(t, tt.want, Is(tt.args.err, tt.args.target))
		})
	}
}

func TestAs(t *testing.T) {
	err := customErr{msg: "test message"}
	err2 := New("signal error")

	type args struct {
		err    error
		target interface{}
	}
	cases := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "with stack",
			args: args{
				err:    WithStackTrace(err),
				target: new(customErr),
			},
			want: true,
		},
		{
			name: "with frame",
			args: args{
				err:    WithFrame(err),
				target: new(customErr),
			},
			want: true,
		},
		{
			name: "with frame",
			args: args{
				err:    WithFrames(err, Frames{}),
				target: new(customErr),
			},
			want: true,
		},
		{
			name: "with message",
			args: args{
				err:    WithMessage(err, "replace err"),
				target: new(customErr),
			},
			want: true,
		},
		{
			name: "std errors compatibility",
			args: args{
				err:    fmt.Errorf("wrap: %w", err),
				target: new(customErr),
			},
			want: true,
		},
		{
			name: "std errors compatibility (false)",
			args: args{
				err:    fmt.Errorf("not wrap: %s", err),
				target: new(customErr),
			},
			want: false,
		},
		{
			name: "std errors multierror compatibility",
			args: args{
				err:    fmt.Errorf("wrap: %w; %w", err, err2),
				target: new(customErr),
			},
			want: true,
		},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			matches := As(tt.args.err, tt.args.target)
			testutils.AssertEqual(t, tt.want, matches)

			if matches {
				ce := tt.args.target.(*customErr)
				testutils.AssertEqual(t, err, *ce, "target set to new value")
			}
		})
	}
}

func TestJoin(t *testing.T) {
	err1 := New("new err 1")
	err2 := New("new err 2")
	err3 := New("new err 3")
	errs := []error{err1, err2, nil, err3}

	merr := Join(errs...)
	testutils.AssertEqual(t, "new err 1; new err 2; new err 3", merr.Error())
	testutils.AssertEqual(t, []error{err1, err2, err3}, merr.(interface{ Unwrap() []error }).Unwrap())
	testutils.AssertNil(t, Join(nil, nil, nil))
}
