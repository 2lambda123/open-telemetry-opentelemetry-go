// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

//go:build go1.17
// +build go1.17

package metric_test

import (
	"context"
	"fmt"
	"math/rand"
	"testing"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/global"
	"go.opentelemetry.io/otel/metric/sdkapi"
	sdk "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/export"
	"go.opentelemetry.io/otel/sdk/metric/processor/processortest"
)

type benchFixture struct {
	meter       metric.Meter
	accumulator *sdk.Accumulator
	B           *testing.B
	export.AggregatorSelector
}

func newFixture(b *testing.B) *benchFixture {
	b.ReportAllocs()
	bf := &benchFixture{
		B:                  b,
		AggregatorSelector: processortest.AggregatorSelector(),
	}

	bf.accumulator = sdk.NewAccumulator(bf)
	bf.meter = metric.WrapMeterImpl(bf.accumulator)
	return bf
}

func (f *benchFixture) Process(export.Accumulation) error {
	return nil
}

func (f *benchFixture) Meter(_ string, _ ...metric.MeterOption) metric.Meter {
	return f.meter
}

func (f *benchFixture) meterMust() metric.MeterMust {
	return metric.Must(f.meter)
}

func makeLabels(n int) []attribute.KeyValue {
	used := map[string]bool{}
	l := make([]attribute.KeyValue, n)
	for i := 0; i < n; i++ {
		var k string
		for {
			k = fmt.Sprint("k", rand.Intn(1000000000))
			if !used[k] {
				used[k] = true
				break
			}
		}
		l[i] = attribute.String(k, fmt.Sprint("v", rand.Intn(1000000000)))
	}
	return l
}

func benchmarkLabels(b *testing.B, n int) {
	ctx := context.Background()
	fix := newFixture(b)
	labs := makeLabels(n)
	cnt := fix.meterMust().NewInt64Counter("int64.sum")

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		cnt.Add(ctx, 1, labs...)
	}
}

func BenchmarkInt64CounterAddWithLabels_1(b *testing.B) {
	benchmarkLabels(b, 1)
}

func BenchmarkInt64CounterAddWithLabels_2(b *testing.B) {
	benchmarkLabels(b, 2)
}

func BenchmarkInt64CounterAddWithLabels_4(b *testing.B) {
	benchmarkLabels(b, 4)
}

func BenchmarkInt64CounterAddWithLabels_8(b *testing.B) {
	benchmarkLabels(b, 8)
}

func BenchmarkInt64CounterAddWithLabels_16(b *testing.B) {
	benchmarkLabels(b, 16)
}

// Note: performance does not depend on label set size for the
// benchmarks below--all are benchmarked for a single attribute.

// Iterators

var benchmarkIteratorVar attribute.KeyValue

func benchmarkIterator(b *testing.B, n int) {
	labels := attribute.NewSet(makeLabels(n)...)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		iter := labels.Iter()
		for iter.Next() {
			benchmarkIteratorVar = iter.Label()
		}
	}
}

func BenchmarkIterator_0(b *testing.B) {
	benchmarkIterator(b, 0)
}

func BenchmarkIterator_1(b *testing.B) {
	benchmarkIterator(b, 1)
}

func BenchmarkIterator_2(b *testing.B) {
	benchmarkIterator(b, 2)
}

func BenchmarkIterator_4(b *testing.B) {
	benchmarkIterator(b, 4)
}

func BenchmarkIterator_8(b *testing.B) {
	benchmarkIterator(b, 8)
}

func BenchmarkIterator_16(b *testing.B) {
	benchmarkIterator(b, 16)
}

// Counters

func BenchmarkGlobalInt64CounterAddWithSDK(b *testing.B) {
	// Compare with BenchmarkInt64CounterAdd() to see overhead of global
	// package. This is in the SDK to avoid the API from depending on the
	// SDK.
	ctx := context.Background()
	fix := newFixture(b)

	sdk := global.Meter("test")
	global.SetMeterProvider(fix)

	labs := []attribute.KeyValue{attribute.String("A", "B")}
	cnt := Must(sdk).NewInt64Counter("int64.sum")

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		cnt.Add(ctx, 1, labs...)
	}
}

func BenchmarkInt64CounterAdd(b *testing.B) {
	ctx := context.Background()
	fix := newFixture(b)
	labs := makeLabels(1)
	cnt := fix.meterMust().NewInt64Counter("int64.sum")

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		cnt.Add(ctx, 1, labs...)
	}
}

func BenchmarkFloat64CounterAdd(b *testing.B) {
	ctx := context.Background()
	fix := newFixture(b)
	labs := makeLabels(1)
	cnt := fix.meterMust().NewFloat64Counter("float64.sum")

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		cnt.Add(ctx, 1.1, labs...)
	}
}

// LastValue

func BenchmarkInt64LastValueAdd(b *testing.B) {
	ctx := context.Background()
	fix := newFixture(b)
	labs := makeLabels(1)
	mea := fix.meterMust().NewInt64Histogram("int64.lastvalue")

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		mea.Record(ctx, int64(i), labs...)
	}
}

func BenchmarkFloat64LastValueAdd(b *testing.B) {
	ctx := context.Background()
	fix := newFixture(b)
	labs := makeLabels(1)
	mea := fix.meterMust().NewFloat64Histogram("float64.lastvalue")

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		mea.Record(ctx, float64(i), labs...)
	}
}

// Histograms

func BenchmarkInt64HistogramAdd(b *testing.B) {
	ctx := context.Background()
	fix := newFixture(b)
	labs := makeLabels(1)
	mea := fix.meterMust().NewInt64Histogram("int64.histogram")

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		mea.Record(ctx, int64(i), labs...)
	}
}

func BenchmarkFloat64HistogramAdd(b *testing.B) {
	ctx := context.Background()
	fix := newFixture(b)
	labs := makeLabels(1)
	mea := fix.meterMust().NewFloat64Histogram("float64.histogram")

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		mea.Record(ctx, float64(i), labs...)
	}
}

// Observers

func BenchmarkObserverRegistration(b *testing.B) {
	fix := newFixture(b)
	names := make([]string, 0, b.N)
	for i := 0; i < b.N; i++ {
		names = append(names, fmt.Sprintf("test.%d.lastvalue", i))
	}
	cb := func(_ context.Context, result metric.Int64ObserverResult) {}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		fix.meterMust().NewInt64GaugeObserver(names[i], cb)
	}
}

func BenchmarkGaugeObserverObservationInt64(b *testing.B) {
	ctx := context.Background()
	fix := newFixture(b)
	labs := makeLabels(1)
	_ = fix.meterMust().NewInt64GaugeObserver("test.lastvalue", func(_ context.Context, result metric.Int64ObserverResult) {
		for i := 0; i < b.N; i++ {
			result.Observe((int64)(i), labs...)
		}
	})

	b.ResetTimer()

	fix.accumulator.Collect(ctx)
}

func BenchmarkGaugeObserverObservationFloat64(b *testing.B) {
	ctx := context.Background()
	fix := newFixture(b)
	labs := makeLabels(1)
	_ = fix.meterMust().NewFloat64GaugeObserver("test.lastvalue", func(_ context.Context, result metric.Float64ObserverResult) {
		for i := 0; i < b.N; i++ {
			result.Observe((float64)(i), labs...)
		}
	})

	b.ResetTimer()

	fix.accumulator.Collect(ctx)
}

// BatchRecord

func benchmarkBatchRecord8Labels(b *testing.B, numInst int) {
	const numLabels = 8
	ctx := context.Background()
	fix := newFixture(b)
	labs := makeLabels(numLabels)
	var meas []sdkapi.Measurement

	for i := 0; i < numInst; i++ {
		inst := fix.meterMust().NewInt64Counter(fmt.Sprintf("int64.%d.sum", i))
		meas = append(meas, inst.Measurement(1))
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		fix.accumulator.RecordBatch(ctx, labs, meas...)
	}
}

func BenchmarkBatchRecord8Labels_1Instrument(b *testing.B) {
	benchmarkBatchRecord8Labels(b, 1)
}

func BenchmarkBatchRecord_8Labels_2Instruments(b *testing.B) {
	benchmarkBatchRecord8Labels(b, 2)
}

func BenchmarkBatchRecord_8Labels_4Instruments(b *testing.B) {
	benchmarkBatchRecord8Labels(b, 4)
}

func BenchmarkBatchRecord_8Labels_8Instruments(b *testing.B) {
	benchmarkBatchRecord8Labels(b, 8)
}

// Record creation

func BenchmarkRepeatedDirectCalls(b *testing.B) {
	ctx := context.Background()
	fix := newFixture(b)

	c := fix.meterMust().NewInt64Counter("int64.sum")
	k := attribute.String("bench", "true")

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		c.Add(ctx, 1, k)
		fix.accumulator.Collect(ctx)
	}
}
