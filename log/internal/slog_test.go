// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal

import (
	"testing"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/log"

	"github.com/stretchr/testify/assert"
	"golang.org/x/exp/slog"
)

func TestSlogHandler(t *testing.T) {
	spy := &spyLogger{}
	l := slog.New(&slogHandler{spy})

	l.InfoContext(ctx, testBody, "string", testString)

	want := log.Record{}
	want.SetBody(testBody)
	want.SetSeverity(log.SeverityInfo)
	want.AddAttributes(attribute.String("string", testString))

	assert.Equal(t, testBody, spy.Record.Body())
	assert.Equal(t, log.SeverityInfo, spy.Record.Severity())
	assert.Equal(t, 1, spy.Record.AttributesLen())
	spy.Record.WalkAttributes(func(kv attribute.KeyValue) bool {
		assert.Equal(t, "string", string(kv.Key))
		assert.Equal(t, testString, kv.Value.AsString())
		return true
	})
}
