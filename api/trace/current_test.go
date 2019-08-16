package trace_test

import (
	"context"
	"testing"

	"google.golang.org/grpc/codes"

	"go.opentelemetry.io/api/core"
	"go.opentelemetry.io/api/tag"
	"go.opentelemetry.io/api/trace"
)

func TestSetCurrentSpanOverridesPreviouslySetSpan(t *testing.T) {
	originalSpan := trace.NoopSpan{}
	expectedSpan := mockSpan{}

	ctx := context.Background()

	ctx = trace.SetCurrentSpan(ctx, originalSpan)
	ctx = trace.SetCurrentSpan(ctx, expectedSpan)

	if span := trace.CurrentSpan(ctx); span != expectedSpan {
		t.Errorf("Want: %v, but have: %v", expectedSpan, span)
	}
}

func TestCurrentSpan(t *testing.T) {
	for _, testcase := range []struct {
		name string
		ctx  context.Context
		want trace.Span
	}{
		{
			name: "CurrentSpan() returns a NoopSpan{} from an empty context",
			ctx:  context.Background(),
			want: trace.NoopSpan{},
		},
		{
			name: "CurrentSpan() returns current span if set",
			ctx:  trace.SetCurrentSpan(context.Background(), mockSpan{}),
			want: mockSpan{},
		},
	} {
		t.Run(testcase.name, func(t *testing.T) {
			// proto: CurrentSpan(ctx context.Context) trace.Span
			have := trace.CurrentSpan(testcase.ctx)
			if have != testcase.want {
				t.Errorf("Want: %v, but have: %v", testcase.want, have)
			}
		})
	}
}

// a duplicate of trace.NoopSpan for testing
type mockSpan struct{}

var _ trace.Span = mockSpan{}

// SpanContext returns an invalid span context.
func (mockSpan) SpanContext() core.SpanContext {
	return core.EmptySpanContext()
}

// IsRecordingEvents always returns false for mockSpan.
func (mockSpan) IsRecordingEvents() bool {
	return false
}

// SetStatus does nothing.
func (mockSpan) SetStatus(status codes.Code) {
}

// SetName does nothing.
func (mockSpan) SetName(name string) {
}

// SetError does nothing.
func (mockSpan) SetError(v bool) {
}

// SetAttribute does nothing.
func (mockSpan) SetAttribute(attribute core.KeyValue) {
}

// SetAttributes does nothing.
func (mockSpan) SetAttributes(attributes ...core.KeyValue) {
}

// ModifyAttribute does nothing.
func (mockSpan) ModifyAttribute(mutator tag.Mutator) {
}

// ModifyAttributes does nothing.
func (mockSpan) ModifyAttributes(mutators ...tag.Mutator) {
}

// Finish does nothing.
func (mockSpan) Finish() {
}

// Tracer returns noop implementation of Tracer.
func (mockSpan) Tracer() trace.Tracer {
	return trace.NoopTracer{}
}

// Event does nothing.
func (mockSpan) AddEvent(ctx context.Context, msg string, attrs ...core.KeyValue) {
}
