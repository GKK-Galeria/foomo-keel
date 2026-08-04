package telemetry

import (
	"context"
	"path"

	goruntime "github.com/foomo/go/runtime"
	keelsemconv "github.com/foomo/keel/semconv"
	"github.com/pkg/errors"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	semconv "go.opentelemetry.io/otel/semconv/v1.41.0"
	"go.opentelemetry.io/otel/trace"
)

// Deprecated: use StartSpan instead.
func Start(ctx context.Context, spanName string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	return startSpan(ctx, 1, opts...)
}

// StartSpan starts a new span with the given name and options.
func StartSpan(ctx context.Context, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	return startSpan(ctx, 1, opts...)
}

// StartDebugSpan starts a new span with the given name and options and adds the debug attr.
func StartDebugSpan(ctx context.Context, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	return startSpan(ctx, 1, append(opts, trace.WithAttributes(keelsemconv.DebugEnabled(true)))...)
}

func SpanFromContext(ctx context.Context) trace.Span {
	return trace.SpanFromContext(ctx)
}

// AddSpanEvent adds an event to the span.
func AddSpanEvent(ctx context.Context, name string, opts ...trace.EventOption) {
	SpanFromContext(ctx).AddEvent(name, opts...)
}

// SetSpanAttributes sets attributes on the span.
func SetSpanAttributes(ctx context.Context, attrs ...attribute.KeyValue) {
	SpanFromContext(ctx).SetAttributes(attrs...)
}

// SetSpanName sets the name of the span.
func SetSpanName(ctx context.Context, name string) {
	SpanFromContext(ctx).SetName(name)
}

// IsSpanRecording returns true if the span is recording.
func IsSpanRecording(ctx context.Context) bool {
	return SpanFromContext(ctx).IsRecording()
}

// SetSpanDebug sets the span debug attribute.
func SetSpanDebug(ctx context.Context) {
	SpanFromContext(ctx).SetAttributes(keelsemconv.DebugEnabled(true))
}

// SetSpanStatusOK sets the status of the span to ok.
func SetSpanStatusOK(ctx context.Context) {
	SpanFromContext(ctx).SetStatus(codes.Ok, "")
}

// SetSpanStatusError sets the status of the span to error.
func SetSpanStatusError(ctx context.Context, description string) {
	SpanFromContext(ctx).SetStatus(codes.Error, description)
}

// Deprecated: use EndSpan instead.
func End(sp trace.Span, err error) {
	if err != nil {
		sp.RecordError(err, trace.WithAttributes(CodeStacktrace(3, 0)))
		sp.SetStatus(codes.Error, errors.Cause(err).Error())
	} else {
		sp.SetStatus(codes.Ok, "")
	}

	sp.End()
}

// EndSpan ends the span.
func EndSpan(ctx context.Context, err error, opts ...trace.SpanEndOption) {
	sp := SpanFromContext(ctx)
	if err != nil {
		sp.RecordError(err, trace.WithAttributes(CodeStacktrace(3, 0)))
		sp.SetStatus(codes.Error, errors.Cause(err).Error())
	}

	sp.End(opts...)
}

// DeferEndSpan is a helper, so you can do `defer ctx.DeferEndSpan(&err)` instead of `defer func(){ ctx.EndSpan(err) }()`
func DeferEndSpan(ctx context.Context, err *error, opts ...trace.SpanEndOption) {
	EndSpan(ctx, *err)
}

func startSpan(ctx context.Context, skip int, opts ...trace.SpanStartOption) (Context, trace.Span) {
	name := "runtime.go"

	if fr := goruntime.CallFrame(skip + 1); !fr.Zero() {
		name = path.Base(fr.Pkg)
		opts = append(opts, trace.WithAttributes(
			semconv.CodeFunctionName(fr.Name()),
			semconv.CodeLineNumber(fr.Line),
			semconv.CodeFilePath(fr.File),
		))
	}

	ctx, span := Tracer().Start(ctx, name, opts...) //nolint:spancheck

	return Ctx(ctx), span //nolint:spancheck
}
