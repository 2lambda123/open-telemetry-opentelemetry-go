// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package log // import "go.opentelemetry.io/otel/sdk/log"
import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/sdk/instrumentation"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/trace"
)

func TestLoggerEmit(t *testing.T) {
	p0, p1, p2WithError := newProcessor("0"), newProcessor("1"), newProcessor("2")
	p2WithError.err = errors.New("error")

	r := log.Record{}
	r.SetTimestamp(time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC))
	r.SetBody(log.StringValue("testing body value"))
	r.SetSeverity(log.SeverityInfo)
	r.SetSeverityText("testing text")
	r.AddAttributes(
		log.String("k1", "str"),
		log.Float64("k2", 1.0),
	)
	r.SetObservedTimestamp(time.Date(2001, time.January, 1, 0, 0, 0, 0, time.UTC))

	contextWithSpanContext := trace.ContextWithSpanContext(context.Background(), trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    trace.TraceID{0o1},
		SpanID:     trace.SpanID{0o2},
		TraceFlags: 0x1,
	}))

	testCases := []struct {
		name            string
		logger          *logger
		ctx             context.Context
		record          log.Record
		expectedRecords []Record
	}{
		{
			name:   "NoProcessors",
			logger: newLogger(NewLoggerProvider(), instrumentation.Scope{}),
			ctx:    context.Background(),
			record: r,
		},
		{
			name: "WithProcessors",
			logger: newLogger(NewLoggerProvider(
				WithProcessor(p0),
				WithProcessor(p1),
				WithAttributeValueLengthLimit(3),
				WithAttributeCountLimit(2),
				WithResource(resource.NewSchemaless(attribute.String("key", "value"))),
			), instrumentation.Scope{Name: "scope"}),
			ctx:    context.Background(),
			record: r,
			expectedRecords: []Record{
				{
					timestamp:                 r.Timestamp(),
					body:                      r.Body(),
					severity:                  r.Severity(),
					severityText:              r.SeverityText(),
					observedTimestamp:         r.ObservedTimestamp(),
					resource:                  resource.NewSchemaless(attribute.String("key", "value")),
					attributeValueLengthLimit: 3,
					attributeCountLimit:       2,
					scope:                     &instrumentation.Scope{Name: "scope"},
					front: [attributesInlineCount]log.KeyValue{
						log.String("k1", "str"),
						log.Float64("k2", 1.0),
					},
					nFront: 2,
				},
			},
		},
		{
			name: "WithProcessorsWithError",
			logger: newLogger(NewLoggerProvider(
				WithProcessor(p2WithError),
				WithAttributeValueLengthLimit(3),
				WithAttributeCountLimit(2),
				WithResource(resource.NewSchemaless(attribute.String("key", "value"))),
			), instrumentation.Scope{Name: "scope"}),
			ctx: context.Background(),
		},
		{
			name: "WithTraceSpanInContext",
			logger: newLogger(NewLoggerProvider(
				WithProcessor(p0),
				WithProcessor(p1),
				WithAttributeValueLengthLimit(3),
				WithAttributeCountLimit(2),
				WithResource(resource.NewSchemaless(attribute.String("key", "value"))),
			), instrumentation.Scope{Name: "scope"}),
			ctx:    contextWithSpanContext,
			record: r,
			expectedRecords: []Record{
				{
					timestamp:                 r.Timestamp(),
					body:                      r.Body(),
					severity:                  r.Severity(),
					severityText:              r.SeverityText(),
					observedTimestamp:         r.ObservedTimestamp(),
					resource:                  resource.NewSchemaless(attribute.String("key", "value")),
					attributeValueLengthLimit: 3,
					attributeCountLimit:       2,
					scope:                     &instrumentation.Scope{Name: "scope"},
					front: [attributesInlineCount]log.KeyValue{
						log.String("k1", "str"),
						log.Float64("k2", 1.0),
					},
					nFront:     2,
					traceID:    trace.TraceID{0o1},
					spanID:     trace.SpanID{0o2},
					traceFlags: 0x1,
				},
			},
		},
		{
			name: "WithNilContext",
			logger: newLogger(NewLoggerProvider(
				WithProcessor(p0),
				WithProcessor(p1),
				WithAttributeValueLengthLimit(3),
				WithAttributeCountLimit(2),
				WithResource(resource.NewSchemaless(attribute.String("key", "value"))),
			), instrumentation.Scope{Name: "scope"}),
			ctx:    context.Background(),
			record: r,
			expectedRecords: []Record{
				{
					timestamp:                 r.Timestamp(),
					body:                      r.Body(),
					severity:                  r.Severity(),
					severityText:              r.SeverityText(),
					observedTimestamp:         r.ObservedTimestamp(),
					resource:                  resource.NewSchemaless(attribute.String("key", "value")),
					attributeValueLengthLimit: 3,
					attributeCountLimit:       2,
					scope:                     &instrumentation.Scope{Name: "scope"},
					front: [attributesInlineCount]log.KeyValue{
						log.String("k1", "str"),
						log.Float64("k2", 1.0),
					},
					nFront: 2,
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Clean up the records before the test.
			p0.records = nil
			p1.records = nil

			tc.logger.Emit(tc.ctx, tc.record)

			assert.Equal(t, tc.expectedRecords, p0.records)
			assert.Equal(t, tc.expectedRecords, p1.records)
		})
	}
}

func TestLoggerEnabled(t *testing.T) {
	l := newLogger(NewLoggerProvider(), instrumentation.Scope{})

	assert.True(t, l.Enabled(context.Background(), log.Record{}))
}
