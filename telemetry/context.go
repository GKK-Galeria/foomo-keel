package telemetry

import (
	"context"
	"path"
	"runtime/pprof"
	"time"

	goruntime "github.com/foomo/go/runtime"
	keelsemconv "github.com/foomo/keel/semconv"
	foomosemconv "github.com/foomo/opentelemetry-go/semconv"
	"github.com/grafana/pyroscope-go"
	"github.com/pkg/errors"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	semconv "go.opentelemetry.io/otel/semconv/v1.41.0"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap/zapcore"
)

type Context struct {
	ctx context.Context
}

// Ctx returns a new Context from the given context.Context.
func Ctx(ctx context.Context) Context {
	return Context{ctx}
}

// ------------------------------------------------------------------------------------------------
// ~ Public methods
// ------------------------------------------------------------------------------------------------

// LogDebug logs a message at `debug` level.
func (c Context) LogDebug(msg string, kv ...attribute.KeyValue) {
	c.log(c.ctx, zapcore.DebugLevel, msg, 1, kv...)
}

// LogInfo logs a message at `info` level.
func (c Context) LogInfo(msg string, kv ...attribute.KeyValue) {
	c.log(c.ctx, zapcore.InfoLevel, msg, 1, kv...)
}

// LogWarn logs a message at `warn` level.
func (c Context) LogWarn(msg string, kv ...attribute.KeyValue) {
	c.log(c.ctx, zapcore.WarnLevel, msg, 1, kv...)
}

// LogError logs a message at `error` level.
func (c Context) LogError(msg string, kv ...attribute.KeyValue) {
	c.log(c.ctx, zapcore.ErrorLevel, msg, 1, kv...)
}

// Context returns the underlying context.Context.
func (c Context) Context() context.Context {
	return c.ctx
}

// ContextWithCancel returns a copy of the context with a new Done channel.
// The returned context's Done channel is closed when the returned cancel function
// is called or when the parent context's Done channel is closed, whichever happens first.
func (c Context) ContextWithCancel() (context.Context, context.CancelFunc) {
	return context.WithCancel(c.ctx)
}

// ContextWithCancelCause returns a copy of the context with a new Done channel and
// a CancelCauseFunc instead of a CancelFunc. Calling cancel with a non-nil error
// (the "cause") records that error in the context; it can then be retrieved using Cause(ctx).
func (c Context) ContextWithCancelCause() (context.Context, context.CancelCauseFunc) {
	return context.WithCancelCause(c.ctx)
}

// ContextWith returns a copy of the context with a deadline.
// The returned context's Done channel is closed when the deadline expires, when the
// returned cancel function is called, or when the parent context's Done channel is
// closed, whichever happens first.
func (c Context) ContextWith(deadline time.Time) (context.Context, context.CancelFunc) {
	return context.WithDeadline(c.ctx, deadline)
}

// ContextWithTimeout returns a copy of the context with a timeout.
// It is equivalent to ContextWith(time.Now().Add(timeout)).
func (c Context) ContextWithTimeout(timeout time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(c.ctx, timeout)
}

// ContextWithValue returns a copy of the context with the key-value pair associated.
func (c Context) ContextWithValue(key, val any) context.Context {
	return context.WithValue(c.ctx, key, val)
}

// ContextWithoutCancel returns a copy of the context that is not canceled when
// parent is canceled. The returned context returns no Deadline or Err, and its
// Done channel is nil.
func (c Context) ContextWithoutCancel() context.Context {
	return context.WithoutCancel(c.ctx)
}

// Value returns the value associated with this context for key, or nil
// if no value is associated with key.
func (c Context) Value(key any) any {
	return c.ctx.Value(key)
}

// Deadline returns the time when work done on behalf of this context should be
// canceled. Deadline returns ok==false when no deadline is set.
func (c Context) Deadline() (deadline time.Time, ok bool) {
	return c.ctx.Deadline()
}

// Done returns a channel that's closed when work done on behalf of this context
// should be canceled. Done may return nil if this context can never be canceled.
func (c Context) Done() <-chan struct{} {
	return c.ctx.Done()
}

// Err returns a non-nil error value after Done is closed. Err returns Canceled
// if the context was canceled or DeadlineExceeded if the context's deadline passed.
func (c Context) Err() error {
	return c.ctx.Err()
}

// Span returns the span from the context.
func (c Context) Span() trace.Span {
	return trace.SpanFromContext(c.ctx)
}

// SetSpanDebug sets the span debug attribute.
func (c Context) SetSpanDebug() {
	c.Span().SetAttributes(keelsemconv.DebugEnabled(true))
}

// EndSpan ends the span.
func (c Context) EndSpan(err error, opts ...trace.SpanEndOption) {
	sp := c.Span()
	if sp.IsRecording() {
		if err != nil {
			sp.RecordError(err, trace.WithAttributes(CodeStacktrace(3, 0)))
			sp.SetStatus(codes.Error, errors.Cause(err).Error())
		} else {
			c.SetSpanStatusOK()
		}

		sp.End(opts...)
	}
}

// DeferEndSpan is a helper so you can do `defer ctx.DeferEndSpan(&err)` instead of `defer func(){ ctx.EndSpan(err) }()`
func (c Context) DeferEndSpan(err *error, opts ...trace.SpanEndOption) {
	e := *err

	sp := c.Span()
	if sp.IsRecording() {
		if e != nil {
			sp.RecordError(e, trace.WithAttributes(CodeStacktrace(5, 2)))
			sp.SetStatus(codes.Error, errors.Cause(e).Error())
		} else {
			c.SetSpanStatusOK()
		}

		sp.End(opts...)
	}
}

// SetSpanStatusOK sets the status of the span to ok.
func (c Context) SetSpanStatusOK() {
	c.Span().SetStatus(codes.Ok, "")
}

// SetSpanStatusError sets the status of the span to error.
func (c Context) SetSpanStatusError(description string) {
	c.Span().SetStatus(codes.Error, description)
}

// SetSpanName sets the name of the span.
func (c Context) SetSpanName(name string) {
	c.Span().SetName(name)
}

// SetSpanAttributes sets the attributes of the span.
func (c Context) SetSpanAttributes(kv ...attribute.KeyValue) {
	c.Span().SetAttributes(kv...)
}

// RecordError records an error on the span and logs it.
func (c Context) RecordError(err error, kv ...attribute.KeyValue) {
	sp := c.Span()
	if sp.IsRecording() {
		sp.RecordError(err,
			trace.WithAttributes(kv...),
			trace.WithAttributes(CodeStacktrace(5, 1)),
		)
		sp.SetStatus(codes.Error, errors.Cause(err).Error())
	}
}

// RecordSpanError records an error on the span.
func (c Context) RecordSpanError(err error, kv ...attribute.KeyValue) {
	sp := c.Span()
	if sp.IsRecording() {
		sp.RecordError(err,
			trace.WithAttributes(append(kv, CodeStacktrace(5, 1))...),
		)
	}
}

// AddSpanEvent adds an event to the span.
func (c Context) AddSpanEvent(name string, kv ...attribute.KeyValue) {
	c.Span().AddEvent(name, trace.WithAttributes(kv...))
}

// StartSpan starts a span.
func (c Context) StartSpan(opts ...trace.SpanStartOption) Context {
	ctx, _ := c.startSpan("", 1, opts...)
	return ctx
}

// StartSpanWithNewRoot sets the name of the span.
func (c Context) StartSpanWithNewRoot(opts ...trace.SpanStartOption) Context {
	ctx, _ := c.startSpan("", 1, append(opts, trace.WithNewRoot(), trace.WithLinks(trace.LinkFromContext(c.ctx)))...)
	return ctx
}

// StartSpanWithProfile starts a span and profiles the handler.
func (c Context) StartSpanWithProfile(name string, handler func(ctx Context), kv ...attribute.KeyValue) {
	ctx, span := c.startSpan("", 1, trace.WithAttributes(kv...))
	defer span.End()

	ctx.StartProfile(name, handler, kv...)
}

// StartProfile starts a profile for the handler.
func (c Context) StartProfile(name string, handler func(ctx Context), kv ...attribute.KeyValue) {
	attr := foomosemconv.ProfileName(name)
	c.Span().SetAttributes(attr)
	pyroscope.TagWrapper(c.ctx, PyroscopeLabels(append(kv, attr)...), func(ctx context.Context) {
		handler(Ctx(ctx))
	})
}

// SetProfileAttributes sets the labels for the profile.
func (c Context) SetProfileAttributes(kv ...attribute.KeyValue) Context {
	ctx := pprof.WithLabels(c.ctx, PyroscopeLabels(kv...))
	pprof.SetGoroutineLabels(ctx)

	return Ctx(ctx)
}

// ------------------------------------------------------------------------------------------------
// ~ Private methods
// ------------------------------------------------------------------------------------------------

func (c Context) startSpan(name string, skip int, opts ...trace.SpanStartOption) (Context, trace.Span) {
	if name == "" {
		if fr := goruntime.CallFrame(skip + 1); !fr.Zero() {
			name = path.Base(fr.Pkg) + "." + fr.Short()
			opts = append(opts, trace.WithAttributes(
				semconv.CodeFunctionName(fr.Name()),
				semconv.CodeLineNumber(fr.Line),
				semconv.CodeFilePath(fr.File),
			))
		} else {
			name = "runtime.go"
		}
	}

	ctx, span := Tracer().Start(c.ctx, name, opts...)

	return Ctx(ctx), span
}

func (c Context) log(ctx context.Context, lvl zapcore.Level, msg string, skip int, kv ...attribute.KeyValue) {
	Log(ctx, lvl, msg, skip+1, kv...)
}
