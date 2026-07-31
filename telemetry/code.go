package telemetry

import (
	goruntime "github.com/foomo/go/runtime"
	"go.opentelemetry.io/otel/attribute"
	semconv "go.opentelemetry.io/otel/semconv/v1.41.0"
)

func CodeStacktrace(num, skip int) attribute.KeyValue {
	return semconv.CodeStacktrace(goruntime.StackTrace(num, skip+1))
}
