// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package oc2otel // import "go.opentelemetry.io/otel/bridge/opencensus/internal/oc2otel"

import (
	octrace "go.opencensus.io/trace"

	"go.opentelemetry.io/otel/attribute"
)

func Attributes(attr []octrace.Attribute) []attribute.KeyValue {
	otelAttr := make([]attribute.KeyValue, len(attr))
	for i, a := range attr {
		otelAttr[i] = attribute.KeyValue{
			Key:   attribute.Key(a.Key()),
			Value: AttributeValue(a.Value()),
		}
	}
	return otelAttr
}

func AttributeValue(ocval interface{}) attribute.Value {
	switch v := ocval.(type) {
	case bool:
		return attribute.BoolValue(v)
	case int64:
		return attribute.Int64Value(v)
	case float64:
		return attribute.Float64Value(v)
	case string:
		return attribute.StringValue(v)
	default:
		return attribute.StringValue("unknown")
	}
}
