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

package reducer_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/otel/api/kv"
	"go.opentelemetry.io/otel/api/label"
	"go.opentelemetry.io/otel/api/metric"
	export "go.opentelemetry.io/otel/sdk/export/metric"
	metricsdk "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/exportertest"
	"go.opentelemetry.io/otel/sdk/metric/processor/basic"
	"go.opentelemetry.io/otel/sdk/metric/processor/reducer"
	processortest "go.opentelemetry.io/otel/sdk/metric/processor/test"
	"go.opentelemetry.io/otel/sdk/resource"
)

var (
	kvs1 = []kv.KeyValue{
		kv.Int("A", 1),
		kv.Int("B", 2),
		kv.Int("C", 3),
	}
	kvs2 = []kv.KeyValue{
		kv.Int("A", 1),
		kv.Int("B", 0),
		kv.Int("C", 3),
	}
)

type testFilter struct{}

func (testFilter) LabelFilterFor(_ *metric.Descriptor) label.Filter {
	return func(k string) bool {
		return k == "A" || k == "C"
	}
}

func TestFilterProcessor(t *testing.T) {
	ctx := context.Background()
	testProc := processortest.NewProcessor(
		processortest.AggregatorSelector(),
		label.DefaultEncoder(),
	)
	accum := metricsdk.NewAccumulator(
		reducer.New(testFilter{}, processortest.SingleCheckpointer(testProc)),
		metricsdk.WithResource(
			resource.New(kv.String("R", "V")),
		),
	)
	meter := metric.WrapMeterImpl(accum, "testing")

	counter := metric.Must(meter).NewFloat64Counter("counter.sum")

	_ = metric.Must(meter).NewInt64SumObserver("observer.sum",
		func(_ context.Context, result metric.Int64ObserverResult) {
			result.Observe(10, kvs1...)
			result.Observe(10, kvs2...)
		},
	)

	counter.Add(ctx, 100, kvs1...)
	counter.Add(ctx, 100, kvs2...)

	accum.Collect(ctx)

	require.EqualValues(t, map[string]float64{
		"counter.sum/A=1,C=3/R=V":  200,
		"observer.sum/A=1,C=3/R=V": 20,
	}, testProc.Values())
}

// Test a filter with the ../basic Processor.
func TestFilterBasicProcessor(t *testing.T) {
	ctx := context.Background()
	eselector := processortest.ExportKindSelector(export.CumulativeExporter)
	basicProc := basic.New(processortest.AggregatorSelector(), eselector)
	accum := metricsdk.NewAccumulator(
		reducer.New(testFilter{}, basicProc),
		metricsdk.WithResource(
			resource.New(kv.String("R", "V")),
		),
	)
	exporter := exportertest.NewExporter(basicProc, eselector)

	meter := metric.WrapMeterImpl(accum, "testing")

	counter := metric.Must(meter).NewFloat64Counter("counter.sum")

	_ = metric.Must(meter).NewInt64SumObserver("observer.sum",
		func(_ context.Context, result metric.Int64ObserverResult) {
			result.Observe(10, kvs1...)
			result.Observe(10, kvs2...)
		},
	)

	counter.Add(ctx, 100, kvs1...)
	counter.Add(ctx, 100, kvs2...)

	basicProc.StartCollection()
	accum.Collect(ctx)
	if err := basicProc.FinishCollection(); err != nil {
		t.Error(err)
	}

	require.EqualValues(t, map[string]float64{
		"counter.sum/A=1,C=3/R=V":  200,
		"observer.sum/A=1,C=3/R=V": 20,
	}, exporter.Values(basicProc.CheckpointSet(), label.DefaultEncoder()))
}
