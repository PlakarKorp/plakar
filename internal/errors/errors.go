package errors

import (
	"errors"
	"fmt"
	"runtime"
	"sync"
)

// Code represents a typed error category.
type Code string

const (
	// CodeUnknown is used when no specific code is provided.
	CodeUnknown Code = "unknown"
)

// Error exposes a typed error enriched with stack information and optional cause.
type Error struct {
	code    Code
	msg     string
	cause   error
	stack   []uintptr
	context map[string]any
}

// Option mutates an error during construction.
type Option func(*Error)

// WithContext attaches contextual key/values to the error.
func WithContext(key string, value any) Option {
	return func(e *Error) {
		if e.context == nil {
			e.context = make(map[string]any)
		}
		e.context[key] = value
	}
}

// WithCause sets the underlying cause.
func WithCause(err error) Option {
	return func(e *Error) {
		e.cause = err
	}
}

// New instantiates a typed error while capturing the current stack.
func New(code Code, message string, opts ...Option) *Error {
	err := &Error{
		code:  code,
		msg:   message,
		stack: captureStack(3),
	}
	for _, opt := range opts {
		opt(err)
	}
	return err
}

// Wrap wraps the provided cause with an optional message. When message is empty
// the cause message is reused.
func Wrap(code Code, err error, message string, opts ...Option) *Error {
	if err == nil {
		return nil
	}
	if message == "" {
		message = err.Error()
	}
	opts = append(opts, WithCause(err))
	return New(code, message, opts...)
}

// From attempts to extract an *Error; unknown errors are wrapped in CodeUnknown.
func From(err error) *Error {
	if err == nil {
		return nil
	}
	var target *Error
	if errors.As(err, &target) {
		return target
	}
	return Wrap(CodeUnknown, err, err.Error())
}

// Error implements the error interface.
func (e *Error) Error() string {
	if e == nil {
		return ""
	}
	return e.msg
}

// Unwrap returns the underlying cause.
func (e *Error) Unwrap() error {
	return e.cause
}

// Code returns the error code.
func (e *Error) Code() Code {
	if e == nil {
		return CodeUnknown
	}
	return e.code
}

// Stack returns the captured call stack as program counters.
func (e *Error) Stack() []uintptr {
	if e == nil {
		return nil
	}
	stack := make([]uintptr, len(e.stack))
	copy(stack, e.stack)
	return stack
}

// Context returns a copy of the contextual metadata.
func (e *Error) Context() map[string]any {
	if e == nil || e.context == nil {
		return nil
	}
	clone := make(map[string]any, len(e.context))
	for k, v := range e.context {
		clone[k] = v
	}
	return clone
}

func captureStack(skip int) []uintptr {
	pcs := make([]uintptr, 32)
	n := runtime.Callers(skip, pcs)
	return pcs[:n]
}

// Manager coordinates error observers.
type Manager struct {
	mu       sync.RWMutex
	handlers []Handler
}

// Handler receives emitted errors.
type Handler func(*Error)

// NewManager constructs an empty manager.
func NewManager() *Manager {
	return &Manager{}
}

// Register adds a handler that will be invoked on every Emit call.
func (m *Manager) Register(handler Handler) {
	if handler == nil {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.handlers = append(m.handlers, handler)
}

// Emit normalises the provided error, notifies subscribers, and returns the
// resulting *Error instance.
func (m *Manager) Emit(err error) *Error {
	e := From(err)
	if e == nil {
		return nil
	}

	m.mu.RLock()
	handlers := make([]Handler, len(m.handlers))
	copy(handlers, m.handlers)
	m.mu.RUnlock()

	for _, handler := range handlers {
		handler(e)
	}
	return e
}

// Format returns a string including code and message for quick diagnostics.
func (e *Error) Format() string {
	if e == nil {
		return ""
	}
	return fmt.Sprintf("[%s] %s", e.code, e.msg)
}
