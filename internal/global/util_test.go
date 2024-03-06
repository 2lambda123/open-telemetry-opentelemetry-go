// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package global

import (
	"sync"
	"testing"
)

// ResetForTest configures the test to restores the initial global state during
// its Cleanup step.
func ResetForTest(t testing.TB) {
	t.Cleanup(func() {
		globalTracer = defaultTracerValue()
		globalPropagators = defaultPropagatorsValue()
		globalMeterProvider = defaultMeterProvider()
		delegateTraceOnce = sync.Once{}
		delegateTextMapPropagatorOnce = sync.Once{}
		delegateMeterOnce = sync.Once{}
	})
}
