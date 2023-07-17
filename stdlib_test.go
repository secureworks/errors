package errors_test

import (
	stderrors "errors"
	"fmt"
	"reflect"
	"testing"

	"github.com/secureworks/errors"
	"github.com/secureworks/errors/internal/testutils"
)

type customErr struct {
	msg string
}

func (c customErr) Error() string { return c.msg }

var (
	stackFramer = reflect.TypeOf((*interface {
		Frames() errors.Frames
	})(nil)).Elem()
	stackTracer = reflect.TypeOf((*interface {
		StackTrace() []uintptr
	})(nil)).Elem()
)

func TestNew(t *testing.T) {
	libErr := errors.New("new err")
	stdErr := stderrors.New("new err")

	testutils.AssertNotNil(t, libErr)
	testutils.AssertEqual(t, stdErr, libErr)
	testutils.AssertEqual(t, stdErr.Error(), libErr.Error())
}

func TestNewWith(t *testing.T) {
	cases := []struct {
		name string
		err  error
		wrap bool
		impl []reflect.Type
	}{
		{
			name: "Stack",
			err:  errors.NewWithStackTrace("new err"),
			wrap: true,
			impl: []reflect.Type{
				stackFramer,
				stackTracer,
			},
		},
		{
			name: "Frame",
			err:  errors.NewWithFrame("new err"),
			wrap: true,
			impl: []reflect.Type{
				stackFramer,
			},
		},
		{
			name: "FrameAt",
			err:  errors.NewWithFrameAt("new err", 0),
			wrap: true,
			impl: []reflect.Type{
				stackFramer,
			},
		},
		{
			name: "Frames",
			err:  errors.NewWithFrames("new err", errors.Frames{}),
			wrap: true,
			impl: []reflect.Type{
				stackFramer,
			},
		},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			// Unwraps.
			baseErr := errors.Unwrap(tt.err)
			if tt.wrap {
				testutils.AssertEqual(t, "new err", baseErr.Error())
			} else {
				testutils.AssertNil(t, baseErr)
			}

			// Implements.
			for _, iface := range tt.impl {
				testutils.AssertTrue(t, reflect.TypeOf(tt.err).Implements(iface))
			}
		})
	}
}

func TestUnwrap(t *testing.T) {
	err := errors.New("new err")

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
			args: args{err: errors.WithStackTrace(err)},
			want: err,
		},
		{
			name: "with frame",
			args: args{err: errors.WithFrame(err)},
			want: err,
		},
		{
			name: "with frames",
			args: args{err: errors.WithFrames(err, errors.Frames{})},
			want: err,
		},
		{
			name: "with message",
			args: args{err: errors.WithMessage(err, "replace err")},
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
			testutils.AssertEqual(t, tt.want, errors.Unwrap(tt.args.err))
		})
	}
}

func TestIs(t *testing.T) {
	err := errors.New("new err")

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
				err:    errors.WithStackTrace(err),
				target: err,
			},
			want: true,
		},
		{
			name: "with frame",
			args: args{
				err:    errors.WithFrame(err),
				target: err,
			},
			want: true,
		},
		{
			name: "with frames",
			args: args{
				err:    errors.WithFrames(err, errors.Frames{}),
				target: err,
			},
			want: true,
		},
		{
			name: "with message",
			args: args{
				err:    errors.WithMessage(err, "replace err"),
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
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			testutils.AssertEqual(t, tt.want, errors.Is(tt.args.err, tt.args.target))
		})
	}
}

func TestAs(t *testing.T) {
	err := customErr{msg: "test message"}

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
				err:    errors.WithStackTrace(err),
				target: new(customErr),
			},
			want: true,
		},
		{
			name: "with frame",
			args: args{
				err:    errors.WithFrame(err),
				target: new(customErr),
			},
			want: true,
		},
		{
			name: "with frame",
			args: args{
				err:    errors.WithFrames(err, errors.Frames{}),
				target: new(customErr),
			},
			want: true,
		},
		{
			name: "with message",
			args: args{
				err:    errors.WithMessage(err, "replace err"),
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
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			matches := errors.As(tt.args.err, tt.args.target)
			testutils.AssertEqual(t, tt.want, matches)

			if matches {
				//goland:noinspection GoTypeAssertionOnErrors
				ce := tt.args.target.(*customErr)
				testutils.AssertEqual(t, err, *ce, "target set to new value")
			}
		})
	}
}
