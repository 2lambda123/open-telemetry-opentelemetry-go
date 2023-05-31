// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package metric

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/go-logr/logr/funcr"
	"github.com/stretchr/testify/assert"

	"go.opentelemetry.io/otel"
)

func TestMeterConcurrentSafe(t *testing.T) {
	const name = "TestMeterConcurrentSafe meter"
	mp := NewMeterProvider()

	go func() {
		_ = mp.Meter(name)
	}()

	_ = mp.Meter(name)
}

func TestForceFlushConcurrentSafe(t *testing.T) {
	mp := NewMeterProvider()

	go func() {
		_ = mp.ForceFlush(context.Background())
	}()

	_ = mp.ForceFlush(context.Background())
}

func TestShutdownConcurrentSafe(t *testing.T) {
	mp := NewMeterProvider()

	go func() {
		_ = mp.Shutdown(context.Background())
	}()

	_ = mp.Shutdown(context.Background())
}

func TestMeterDoesNotPanicForEmptyMeterProvider(t *testing.T) {
	mp := MeterProvider{}
	assert.NotPanics(t, func() { _ = mp.Meter("") })
}

func TestForceFlushDoesNotPanicForEmptyMeterProvider(t *testing.T) {
	mp := MeterProvider{}
	assert.NotPanics(t, func() { _ = mp.ForceFlush(context.Background()) })
}

func TestShutdownDoesNotPanicForEmptyMeterProvider(t *testing.T) {
	mp := MeterProvider{}
	assert.NotPanics(t, func() { _ = mp.Shutdown(context.Background()) })
}

func TestMeterProviderReturnsSameMeter(t *testing.T) {
	mp := MeterProvider{}
	mtr := mp.Meter("")

	assert.Same(t, mtr, mp.Meter(""))
	assert.NotSame(t, mtr, mp.Meter("diff"))
}

func TestEmptyMeterName(t *testing.T) {
	var buf strings.Builder
	warnLevel := 1
	l := funcr.New(func(prefix, args string) {
		_, _ = buf.WriteString(fmt.Sprint(prefix, args))
	}, funcr.Options{Verbosity: warnLevel})
	otel.SetLogger(l)
	mp := NewMeterProvider()

	mp.Meter("")

	assert.Contains(t, buf.String(), `"level"=1 "msg"="Meter name should not be empty. Empty meter name is specified as invalid."`)
}
