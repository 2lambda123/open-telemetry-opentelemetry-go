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

package stdout_test

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/otel/api/core"
	"go.opentelemetry.io/otel/api/key"
	"go.opentelemetry.io/otel/api/metric"
	"go.opentelemetry.io/otel/exporters/metric/stdout"
	"go.opentelemetry.io/otel/exporters/metric/test"
	export "go.opentelemetry.io/otel/sdk/export/metric"
	"go.opentelemetry.io/otel/sdk/export/metric/aggregator"
	"go.opentelemetry.io/otel/sdk/metric/aggregator/array"
	"go.opentelemetry.io/otel/sdk/metric/aggregator/ddsketch"
	"go.opentelemetry.io/otel/sdk/metric/aggregator/lastvalue"
	"go.opentelemetry.io/otel/sdk/metric/aggregator/minmaxsumcount"
	"go.opentelemetry.io/otel/sdk/metric/aggregator/sum"
	aggtest "go.opentelemetry.io/otel/sdk/metric/aggregator/test"
	"go.opentelemetry.io/otel/sdk/resource"
)

type testFixture struct {
	t        *testing.T
	ctx      context.Context
	exporter *stdout.Exporter
	resource *resource.Resource
	output   *bytes.Buffer
}

func newFixture(t *testing.T, config stdout.Config) testFixture {
	buf := &bytes.Buffer{}
	config.Writer = buf
	config.DoNotPrintTime = true
	exp, err := stdout.NewRawExporter(config)
	if err != nil {
		t.Fatal("Error building fixture: ", err)
	}
	return testFixture{
		t:        t,
		ctx:      context.Background(),
		resource: resource.New(key.String("res1", "val1"), key.String("res2", "val2")),
		exporter: exp,
		output:   buf,
	}
}

func (fix testFixture) Output() string {
	return strings.TrimSpace(fix.output.String())
}

func (fix testFixture) Export(checkpointSet export.CheckpointSet) {
	err := fix.exporter.Export(fix.ctx, fix.resource, checkpointSet)
	if err != nil {
		fix.t.Error("export failed: ", err)
	}
}

func TestStdoutInvalidQuantile(t *testing.T) {
	_, err := stdout.NewRawExporter(stdout.Config{
		Quantiles: []float64{1.1, 0.9},
	})
	require.Error(t, err, "Invalid quantile error expected")
	require.Equal(t, aggregator.ErrInvalidQuantile, err)
}

func TestStdoutTimestamp(t *testing.T) {
	var buf bytes.Buffer
	exporter, err := stdout.NewRawExporter(stdout.Config{
		Writer:         &buf,
		DoNotPrintTime: false,
	})
	if err != nil {
		t.Fatal("Invalid config: ", err)
	}

	before := time.Now()

	checkpointSet := test.NewCheckpointSet(export.NewDefaultLabelEncoder())

	ctx := context.Background()
	desc := metric.NewDescriptor("test.name", metric.ObserverKind, core.Int64NumberKind)
	lvagg := lastvalue.New()
	aggtest.CheckedUpdate(t, lvagg, core.NewInt64Number(321), &desc)
	lvagg.Checkpoint(ctx, &desc)

	checkpointSet.Add(&desc, lvagg)

	if err := exporter.Export(ctx, resource.New(), checkpointSet); err != nil {
		t.Fatal("Unexpected export error: ", err)
	}

	after := time.Now()

	var printed map[string]interface{}

	if err := json.Unmarshal(buf.Bytes(), &printed); err != nil {
		t.Fatal("JSON parse error: ", err)
	}

	updateTS := printed["time"].(string)
	updateTimestamp, err := time.Parse(time.RFC3339Nano, updateTS)
	if err != nil {
		t.Fatal("JSON parse error: ", updateTS, ": ", err)
	}

	lastValueTS := printed["updates"].([]interface{})[0].(map[string]interface{})["time"].(string)
	lastValueTimestamp, err := time.Parse(time.RFC3339Nano, lastValueTS)
	if err != nil {
		t.Fatal("JSON parse error: ", lastValueTS, ": ", err)
	}

	require.True(t, updateTimestamp.After(before))
	require.True(t, updateTimestamp.Before(after))

	require.True(t, lastValueTimestamp.After(before))
	require.True(t, lastValueTimestamp.Before(after))

	require.True(t, lastValueTimestamp.Before(updateTimestamp))
}

func TestStdoutCounterFormat(t *testing.T) {
	fix := newFixture(t, stdout.Config{})

	checkpointSet := test.NewCheckpointSet(export.NewDefaultLabelEncoder())

	desc := metric.NewDescriptor("test.name", metric.CounterKind, core.Int64NumberKind)
	cagg := sum.New()
	aggtest.CheckedUpdate(fix.t, cagg, core.NewInt64Number(123), &desc)
	cagg.Checkpoint(fix.ctx, &desc)

	checkpointSet.Add(&desc, cagg, key.String("A", "B"), key.String("C", "D"))

	fix.Export(checkpointSet)

	require.Equal(t, `{"updates":[{"name":"test.name{A=B,C=D,res1=val1,res2=val2}","sum":123}]}`, fix.Output())
}

func TestStdoutLastValueFormat(t *testing.T) {
	fix := newFixture(t, stdout.Config{})

	checkpointSet := test.NewCheckpointSet(export.NewDefaultLabelEncoder())

	desc := metric.NewDescriptor("test.name", metric.ObserverKind, core.Float64NumberKind)
	lvagg := lastvalue.New()
	aggtest.CheckedUpdate(fix.t, lvagg, core.NewFloat64Number(123.456), &desc)
	lvagg.Checkpoint(fix.ctx, &desc)

	checkpointSet.Add(&desc, lvagg, key.String("A", "B"), key.String("C", "D"))

	fix.Export(checkpointSet)

	require.Equal(t, `{"updates":[{"name":"test.name{A=B,C=D,res1=val1,res2=val2}","last":123.456}]}`, fix.Output())
}

func TestStdoutMinMaxSumCount(t *testing.T) {
	fix := newFixture(t, stdout.Config{})

	checkpointSet := test.NewCheckpointSet(export.NewDefaultLabelEncoder())

	desc := metric.NewDescriptor("test.name", metric.MeasureKind, core.Float64NumberKind)
	magg := minmaxsumcount.New(&desc)
	aggtest.CheckedUpdate(fix.t, magg, core.NewFloat64Number(123.456), &desc)
	aggtest.CheckedUpdate(fix.t, magg, core.NewFloat64Number(876.543), &desc)
	magg.Checkpoint(fix.ctx, &desc)

	checkpointSet.Add(&desc, magg, key.String("A", "B"), key.String("C", "D"))

	fix.Export(checkpointSet)

	require.Equal(t, `{"updates":[{"name":"test.name{A=B,C=D,res1=val1,res2=val2}","min":123.456,"max":876.543,"sum":999.999,"count":2}]}`, fix.Output())
}

func TestStdoutMeasureFormat(t *testing.T) {
	fix := newFixture(t, stdout.Config{
		PrettyPrint: true,
	})

	checkpointSet := test.NewCheckpointSet(export.NewDefaultLabelEncoder())

	desc := metric.NewDescriptor("test.name", metric.MeasureKind, core.Float64NumberKind)
	magg := array.New()

	for i := 0; i < 1000; i++ {
		aggtest.CheckedUpdate(fix.t, magg, core.NewFloat64Number(float64(i)+0.5), &desc)
	}

	magg.Checkpoint(fix.ctx, &desc)

	checkpointSet.Add(&desc, magg, key.String("A", "B"), key.String("C", "D"))

	fix.Export(checkpointSet)

	require.Equal(t, `{
	"updates": [
		{
			"name": "test.name{A=B,C=D,res1=val1,res2=val2}",
			"min": 0.5,
			"max": 999.5,
			"sum": 500000,
			"count": 1000,
			"quantiles": [
				{
					"q": 0.5,
					"v": 500.5
				},
				{
					"q": 0.9,
					"v": 900.5
				},
				{
					"q": 0.99,
					"v": 990.5
				}
			]
		}
	]
}`, fix.Output())
}

func TestStdoutNoData(t *testing.T) {
	desc := metric.NewDescriptor("test.name", metric.MeasureKind, core.Float64NumberKind)
	for name, tc := range map[string]export.Aggregator{
		"ddsketch":       ddsketch.New(ddsketch.NewDefaultConfig(), &desc),
		"minmaxsumcount": minmaxsumcount.New(&desc),
	} {
		tc := tc
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			fix := newFixture(t, stdout.Config{})

			checkpointSet := test.NewCheckpointSet(export.NewDefaultLabelEncoder())

			magg := tc
			magg.Checkpoint(fix.ctx, &desc)

			checkpointSet.Add(&desc, magg)

			fix.Export(checkpointSet)

			require.Equal(t, `{"updates":null}`, fix.Output())
		})
	}
}

func TestStdoutLastValueNotSet(t *testing.T) {
	fix := newFixture(t, stdout.Config{})

	checkpointSet := test.NewCheckpointSet(export.NewDefaultLabelEncoder())

	desc := metric.NewDescriptor("test.name", metric.ObserverKind, core.Float64NumberKind)
	lvagg := lastvalue.New()
	lvagg.Checkpoint(fix.ctx, &desc)

	checkpointSet.Add(&desc, lvagg, key.String("A", "B"), key.String("C", "D"))

	fix.Export(checkpointSet)

	require.Equal(t, `{"updates":null}`, fix.Output())
}
