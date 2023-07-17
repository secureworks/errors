package errors_test

import (
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

func TestUnwrap(t *testing.T) {
	err := errors.New("root")
	cases := []struct {
		name string
		args error
		want error
	}{
		{
			name: "unwrapped is nil for root",
			args: err,
			want: nil,
		},
		{
			name: "wrapper",
			args: errors.New("wrapper: %w", err),
			want: err,
		},
		{
			name: "std errors compatibility",
			args: fmt.Errorf("wrap: %w", err),
			want: err,
		},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			testutils.AssertEqual(t, tt.want, errors.Unwrap(tt.args))
		})
	}
}

func TestIs(t *testing.T) {
	err := errors.New("root")

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
			name: "different error",
			args: args{
				err:    errors.New("new"),
				target: err,
			},
			want: false,
		},
		{
			name: "wrapper",
			args: args{
				err:    errors.New("wrapper: %w", err),
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
			name: "unrelated",
			args: args{
				err:    errors.New("new1"),
				target: new(customErr),
			},
			want: false,
		},
		{
			name: "wrapper",
			args: args{
				err:    errors.New("wrapper: %w", err),
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
