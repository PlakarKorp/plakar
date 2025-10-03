package errors

import (
	"errors"
	"runtime"
	"sync/atomic"
	"testing"
)

func TestNewCapturesStack(t *testing.T) {
	err := New(Code("test"), "boom")
	if err.Code() != Code("test") {
		t.Fatalf("unexpected code: %s", err.Code())
	}

	stack := err.Stack()
	if len(stack) == 0 {
		t.Fatal("expected stack trace to be captured")
	}

	frame, _ := runtime.CallersFrames(stack).Next()
	if frame.Function == "" {
		t.Fatal("expected stack frame to include function name")
	}
}

func TestWrapRetainsCause(t *testing.T) {
	cause := errors.New("root cause")
	err := Wrap(Code("wrap"), cause, "")
	if !errors.Is(err, cause) {
		t.Fatalf("wrap should retain cause")
	}
	if err.Error() != cause.Error() {
		t.Fatalf("expected message to fallback to cause message")
	}
}

func TestManagerNotifiesHandlers(t *testing.T) {
	manager := NewManager()

	var hits atomic.Int32
	manager.Register(func(e *Error) {
		if e.Code() != Code("emit") {
			t.Fatalf("unexpected code: %s", e.Code())
		}
		hits.Add(1)
	})

	manager.Register(func(e *Error) {
		ctx := e.Context()
		if ctx["key"] != "value" {
			t.Fatalf("context not propagated: %v", ctx)
		}
		hits.Add(1)
	})

	err := New(Code("emit"), "handler", WithContext("key", "value"))
	if got := manager.Emit(err); got != err {
		t.Fatalf("emit should return original error instance")
	}

	if hits.Load() != 2 {
		t.Fatalf("expected both handlers to fire, got %d", hits.Load())
	}
}

func TestFromUnknownError(t *testing.T) {
	err := fromHelper()
	if err.Code() != CodeUnknown {
		t.Fatalf("expected unknown code, got %s", err.Code())
	}
	if err.Unwrap() == nil {
		t.Fatalf("expected cause to be populated")
	}
}

func fromHelper() *Error {
	return From(errors.New("arbitrary"))
}
