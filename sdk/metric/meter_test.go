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

package metric

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"testing"

	"github.com/go-logr/logr"
	"github.com/go-logr/logr/testr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/global"
	"go.opentelemetry.io/otel/metric/instrument"
	"go.opentelemetry.io/otel/sdk/instrumentation"
	"go.opentelemetry.io/otel/sdk/metric/aggregation"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	"go.opentelemetry.io/otel/sdk/metric/metricdata/metricdatatest"
	"go.opentelemetry.io/otel/sdk/resource"
)

// A meter should be able to make instruments concurrently.
func TestMeterInstrumentConcurrency(t *testing.T) {
	wg := &sync.WaitGroup{}
	wg.Add(12)

	m := NewMeterProvider().Meter("inst-concurrency")

	go func() {
		_, _ = m.Float64ObservableCounter("AFCounter")
		wg.Done()
	}()
	go func() {
		_, _ = m.Float64ObservableUpDownCounter("AFUpDownCounter")
		wg.Done()
	}()
	go func() {
		_, _ = m.Float64ObservableGauge("AFGauge")
		wg.Done()
	}()
	go func() {
		_, _ = m.Int64ObservableCounter("AICounter")
		wg.Done()
	}()
	go func() {
		_, _ = m.Int64ObservableUpDownCounter("AIUpDownCounter")
		wg.Done()
	}()
	go func() {
		_, _ = m.Int64ObservableGauge("AIGauge")
		wg.Done()
	}()
	go func() {
		_, _ = m.Float64Counter("SFCounter")
		wg.Done()
	}()
	go func() {
		_, _ = m.Float64UpDownCounter("SFUpDownCounter")
		wg.Done()
	}()
	go func() {
		_, _ = m.Float64Histogram("SFHistogram")
		wg.Done()
	}()
	go func() {
		_, _ = m.Int64Counter("SICounter")
		wg.Done()
	}()
	go func() {
		_, _ = m.Int64UpDownCounter("SIUpDownCounter")
		wg.Done()
	}()
	go func() {
		_, _ = m.Int64Histogram("SIHistogram")
		wg.Done()
	}()

	wg.Wait()
}

var emptyCallback metric.Callback = func(context.Context, metric.Observer) error { return nil }

// A Meter Should be able register Callbacks Concurrently.
func TestMeterCallbackCreationConcurrency(t *testing.T) {
	wg := &sync.WaitGroup{}
	wg.Add(2)

	m := NewMeterProvider().Meter("callback-concurrency")

	go func() {
		_, _ = m.RegisterCallback(emptyCallback)
		wg.Done()
	}()
	go func() {
		_, _ = m.RegisterCallback(emptyCallback)
		wg.Done()
	}()
	wg.Wait()
}

func TestNoopCallbackUnregisterConcurrency(t *testing.T) {
	m := NewMeterProvider().Meter("noop-unregister-concurrency")
	reg, err := m.RegisterCallback(emptyCallback)
	require.NoError(t, err)

	wg := &sync.WaitGroup{}
	wg.Add(2)
	go func() {
		_ = reg.Unregister()
		wg.Done()
	}()
	go func() {
		_ = reg.Unregister()
		wg.Done()
	}()
	wg.Wait()
}

func TestCallbackUnregisterConcurrency(t *testing.T) {
	reader := NewManualReader()
	provider := NewMeterProvider(WithReader(reader))
	meter := provider.Meter("unregister-concurrency")

	actr, err := meter.Float64ObservableCounter("counter")
	require.NoError(t, err)

	ag, err := meter.Int64ObservableGauge("gauge")
	require.NoError(t, err)

	regCtr, err := meter.RegisterCallback(emptyCallback, actr)
	require.NoError(t, err)

	regG, err := meter.RegisterCallback(emptyCallback, ag)
	require.NoError(t, err)

	wg := &sync.WaitGroup{}
	wg.Add(2)
	go func() {
		_ = regCtr.Unregister()
		_ = regG.Unregister()
		wg.Done()
	}()
	go func() {
		_ = regCtr.Unregister()
		_ = regG.Unregister()
		wg.Done()
	}()
	wg.Wait()
}

// Instruments should produce correct ResourceMetrics.
func TestMeterCreatesInstruments(t *testing.T) {
	attrs := []attribute.KeyValue{attribute.String("name", "alice")}
	seven := 7.0
	testCases := []struct {
		name string
		fn   func(*testing.T, metric.Meter)
		want metricdata.Metrics
	}{
		{
			name: "ObservableInt64Count",
			fn: func(t *testing.T, m metric.Meter) {
				cback := func(_ context.Context, o instrument.Int64Observer) error {
					o.Observe(4, attrs...)
					return nil
				}
				ctr, err := m.Int64ObservableCounter("aint", instrument.WithInt64Callback(cback))
				assert.NoError(t, err)
				_, err = m.RegisterCallback(func(_ context.Context, o metric.Observer) error {
					o.ObserveInt64(ctr, 3)
					return nil
				}, ctr)
				assert.NoError(t, err)
			},
			want: metricdata.Metrics{
				Name: "aint",
				Data: metricdata.Sum[int64]{
					Temporality: metricdata.CumulativeTemporality,
					IsMonotonic: true,
					DataPoints: []metricdata.DataPoint[int64]{
						{Attributes: attribute.NewSet(attrs...), Value: 4},
						{Value: 3},
					},
				},
			},
		},
		{
			name: "ObservableInt64UpDownCount",
			fn: func(t *testing.T, m metric.Meter) {
				cback := func(_ context.Context, o instrument.Int64Observer) error {
					o.Observe(4, attrs...)
					return nil
				}
				ctr, err := m.Int64ObservableUpDownCounter("aint", instrument.WithInt64Callback(cback))
				assert.NoError(t, err)
				_, err = m.RegisterCallback(func(_ context.Context, o metric.Observer) error {
					o.ObserveInt64(ctr, 11)
					return nil
				}, ctr)
				assert.NoError(t, err)
			},
			want: metricdata.Metrics{
				Name: "aint",
				Data: metricdata.Sum[int64]{
					Temporality: metricdata.CumulativeTemporality,
					IsMonotonic: false,
					DataPoints: []metricdata.DataPoint[int64]{
						{Attributes: attribute.NewSet(attrs...), Value: 4},
						{Value: 11},
					},
				},
			},
		},
		{
			name: "ObservableInt64Gauge",
			fn: func(t *testing.T, m metric.Meter) {
				cback := func(_ context.Context, o instrument.Int64Observer) error {
					o.Observe(4, attrs...)
					return nil
				}
				gauge, err := m.Int64ObservableGauge("agauge", instrument.WithInt64Callback(cback))
				assert.NoError(t, err)
				_, err = m.RegisterCallback(func(_ context.Context, o metric.Observer) error {
					o.ObserveInt64(gauge, 11)
					return nil
				}, gauge)
				assert.NoError(t, err)
			},
			want: metricdata.Metrics{
				Name: "agauge",
				Data: metricdata.Gauge[int64]{
					DataPoints: []metricdata.DataPoint[int64]{
						{Attributes: attribute.NewSet(attrs...), Value: 4},
						{Value: 11},
					},
				},
			},
		},
		{
			name: "ObservableFloat64Count",
			fn: func(t *testing.T, m metric.Meter) {
				cback := func(_ context.Context, o instrument.Float64Observer) error {
					o.Observe(4, attrs...)
					return nil
				}
				ctr, err := m.Float64ObservableCounter("afloat", instrument.WithFloat64Callback(cback))
				assert.NoError(t, err)
				_, err = m.RegisterCallback(func(_ context.Context, o metric.Observer) error {
					o.ObserveFloat64(ctr, 3)
					return nil
				}, ctr)
				assert.NoError(t, err)
			},
			want: metricdata.Metrics{
				Name: "afloat",
				Data: metricdata.Sum[float64]{
					Temporality: metricdata.CumulativeTemporality,
					IsMonotonic: true,
					DataPoints: []metricdata.DataPoint[float64]{
						{Attributes: attribute.NewSet(attrs...), Value: 4},
						{Value: 3},
					},
				},
			},
		},
		{
			name: "ObservableFloat64UpDownCount",
			fn: func(t *testing.T, m metric.Meter) {
				cback := func(_ context.Context, o instrument.Float64Observer) error {
					o.Observe(4, attrs...)
					return nil
				}
				ctr, err := m.Float64ObservableUpDownCounter("afloat", instrument.WithFloat64Callback(cback))
				assert.NoError(t, err)
				_, err = m.RegisterCallback(func(_ context.Context, o metric.Observer) error {
					o.ObserveFloat64(ctr, 11)
					return nil
				}, ctr)
				assert.NoError(t, err)
			},
			want: metricdata.Metrics{
				Name: "afloat",
				Data: metricdata.Sum[float64]{
					Temporality: metricdata.CumulativeTemporality,
					IsMonotonic: false,
					DataPoints: []metricdata.DataPoint[float64]{
						{Attributes: attribute.NewSet(attrs...), Value: 4},
						{Value: 11},
					},
				},
			},
		},
		{
			name: "ObservableFloat64Gauge",
			fn: func(t *testing.T, m metric.Meter) {
				cback := func(_ context.Context, o instrument.Float64Observer) error {
					o.Observe(4, attrs...)
					return nil
				}
				gauge, err := m.Float64ObservableGauge("agauge", instrument.WithFloat64Callback(cback))
				assert.NoError(t, err)
				_, err = m.RegisterCallback(func(_ context.Context, o metric.Observer) error {
					o.ObserveFloat64(gauge, 11)
					return nil
				}, gauge)
				assert.NoError(t, err)
			},
			want: metricdata.Metrics{
				Name: "agauge",
				Data: metricdata.Gauge[float64]{
					DataPoints: []metricdata.DataPoint[float64]{
						{Attributes: attribute.NewSet(attrs...), Value: 4},
						{Value: 11},
					},
				},
			},
		},

		{
			name: "SyncInt64Count",
			fn: func(t *testing.T, m metric.Meter) {
				ctr, err := m.Int64Counter("sint")
				assert.NoError(t, err)

				ctr.Add(context.Background(), 3)
			},
			want: metricdata.Metrics{
				Name: "sint",
				Data: metricdata.Sum[int64]{
					Temporality: metricdata.CumulativeTemporality,
					IsMonotonic: true,
					DataPoints: []metricdata.DataPoint[int64]{
						{Value: 3},
					},
				},
			},
		},
		{
			name: "SyncInt64UpDownCount",
			fn: func(t *testing.T, m metric.Meter) {
				ctr, err := m.Int64UpDownCounter("sint")
				assert.NoError(t, err)

				ctr.Add(context.Background(), 11)
			},
			want: metricdata.Metrics{
				Name: "sint",
				Data: metricdata.Sum[int64]{
					Temporality: metricdata.CumulativeTemporality,
					IsMonotonic: false,
					DataPoints: []metricdata.DataPoint[int64]{
						{Value: 11},
					},
				},
			},
		},
		{
			name: "SyncInt64Histogram",
			fn: func(t *testing.T, m metric.Meter) {
				gauge, err := m.Int64Histogram("histogram")
				assert.NoError(t, err)

				gauge.Record(context.Background(), 7)
			},
			want: metricdata.Metrics{
				Name: "histogram",
				Data: metricdata.Histogram{
					Temporality: metricdata.CumulativeTemporality,
					DataPoints: []metricdata.HistogramDataPoint{
						{
							Attributes:   attribute.Set{},
							Count:        1,
							Bounds:       []float64{0, 5, 10, 25, 50, 75, 100, 250, 500, 750, 1000, 2500, 5000, 7500, 10000},
							BucketCounts: []uint64{0, 0, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
							Min:          &seven,
							Max:          &seven,
							Sum:          7.0,
						},
					},
				},
			},
		},
		{
			name: "SyncFloat64Count",
			fn: func(t *testing.T, m metric.Meter) {
				ctr, err := m.Float64Counter("sfloat")
				assert.NoError(t, err)

				ctr.Add(context.Background(), 3)
			},
			want: metricdata.Metrics{
				Name: "sfloat",
				Data: metricdata.Sum[float64]{
					Temporality: metricdata.CumulativeTemporality,
					IsMonotonic: true,
					DataPoints: []metricdata.DataPoint[float64]{
						{Value: 3},
					},
				},
			},
		},
		{
			name: "SyncFloat64UpDownCount",
			fn: func(t *testing.T, m metric.Meter) {
				ctr, err := m.Float64UpDownCounter("sfloat")
				assert.NoError(t, err)

				ctr.Add(context.Background(), 11)
			},
			want: metricdata.Metrics{
				Name: "sfloat",
				Data: metricdata.Sum[float64]{
					Temporality: metricdata.CumulativeTemporality,
					IsMonotonic: false,
					DataPoints: []metricdata.DataPoint[float64]{
						{Value: 11},
					},
				},
			},
		},
		{
			name: "SyncFloat64Histogram",
			fn: func(t *testing.T, m metric.Meter) {
				gauge, err := m.Float64Histogram("histogram")
				assert.NoError(t, err)

				gauge.Record(context.Background(), 7)
			},
			want: metricdata.Metrics{
				Name: "histogram",
				Data: metricdata.Histogram{
					Temporality: metricdata.CumulativeTemporality,
					DataPoints: []metricdata.HistogramDataPoint{
						{
							Attributes:   attribute.Set{},
							Count:        1,
							Bounds:       []float64{0, 5, 10, 25, 50, 75, 100, 250, 500, 750, 1000, 2500, 5000, 7500, 10000},
							BucketCounts: []uint64{0, 0, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
							Min:          &seven,
							Max:          &seven,
							Sum:          7.0,
						},
					},
				},
			},
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			rdr := NewManualReader()
			m := NewMeterProvider(WithReader(rdr)).Meter("testInstruments")

			tt.fn(t, m)

			rm, err := rdr.Collect(context.Background())
			assert.NoError(t, err)

			require.Len(t, rm.ScopeMetrics, 1)
			sm := rm.ScopeMetrics[0]
			require.Len(t, sm.Metrics, 1)
			got := sm.Metrics[0]
			metricdatatest.AssertEqual(t, tt.want, got, metricdatatest.IgnoreTimestamp())
		})
	}
}

func TestRegisterNonSDKObserverErrors(t *testing.T) {
	rdr := NewManualReader()
	mp := NewMeterProvider(WithReader(rdr))
	meter := mp.Meter("scope")

	type obsrv struct{ instrument.Asynchronous }
	o := obsrv{}

	_, err := meter.RegisterCallback(
		func(context.Context, metric.Observer) error { return nil },
		o,
	)
	assert.ErrorContains(
		t,
		err,
		"invalid observable: from different implementation",
		"External instrument registred",
	)
}

func TestMeterMixingOnRegisterErrors(t *testing.T) {
	rdr := NewManualReader()
	mp := NewMeterProvider(WithReader(rdr))

	m1 := mp.Meter("scope1")
	m2 := mp.Meter("scope2")
	iCtr, err := m2.Int64ObservableCounter("int64 ctr")
	require.NoError(t, err)
	fCtr, err := m2.Float64ObservableCounter("float64 ctr")
	require.NoError(t, err)
	_, err = m1.RegisterCallback(
		func(context.Context, metric.Observer) error { return nil },
		iCtr, fCtr,
	)
	assert.ErrorContains(
		t,
		err,
		`invalid registration: observable "int64 ctr" from Meter "scope2", registered with Meter "scope1"`,
		"Instrument registred with non-creation Meter",
	)
	assert.ErrorContains(
		t,
		err,
		`invalid registration: observable "float64 ctr" from Meter "scope2", registered with Meter "scope1"`,
		"Instrument registred with non-creation Meter",
	)
}

func TestCallbackObserverNonRegistered(t *testing.T) {
	rdr := NewManualReader()
	mp := NewMeterProvider(WithReader(rdr))

	m1 := mp.Meter("scope1")
	valid, err := m1.Int64ObservableCounter("ctr")
	require.NoError(t, err)

	m2 := mp.Meter("scope2")
	iCtr, err := m2.Int64ObservableCounter("int64 ctr")
	require.NoError(t, err)
	fCtr, err := m2.Float64ObservableCounter("float64 ctr")
	require.NoError(t, err)

	// Panics if Observe is called.
	type int64Obsrv struct{ instrument.Int64Observable }
	int64Foreign := int64Obsrv{}
	type float64Obsrv struct{ instrument.Float64Observable }
	float64Foreign := float64Obsrv{}

	_, err = m1.RegisterCallback(
		func(_ context.Context, o metric.Observer) error {
			o.ObserveInt64(valid, 1)
			o.ObserveInt64(iCtr, 1)
			o.ObserveFloat64(fCtr, 1)
			o.ObserveInt64(int64Foreign, 1)
			o.ObserveFloat64(float64Foreign, 1)
			return nil
		},
		valid,
	)
	require.NoError(t, err)

	var got metricdata.ResourceMetrics
	assert.NotPanics(t, func() {
		got, err = rdr.Collect(context.Background())
	})

	assert.NoError(t, err)
	want := metricdata.ResourceMetrics{
		Resource: resource.Default(),
		ScopeMetrics: []metricdata.ScopeMetrics{
			{
				Scope: instrumentation.Scope{
					Name: "scope1",
				},
				Metrics: []metricdata.Metrics{
					{
						Name: "ctr",
						Data: metricdata.Sum[int64]{
							Temporality: metricdata.CumulativeTemporality,
							IsMonotonic: true,
							DataPoints: []metricdata.DataPoint[int64]{
								{
									Value: 1,
								},
							},
						},
					},
				},
			},
		},
	}
	metricdatatest.AssertEqual(t, want, got, metricdatatest.IgnoreTimestamp())
}

type logSink struct {
	logr.LogSink

	messages []string
}

func newLogSink(t *testing.T) *logSink {
	return &logSink{LogSink: testr.New(t).GetSink()}
}

func (l *logSink) Info(level int, msg string, keysAndValues ...interface{}) {
	l.messages = append(l.messages, msg)
	l.LogSink.Info(level, msg, keysAndValues...)
}

func (l *logSink) Error(err error, msg string, keysAndValues ...interface{}) {
	l.messages = append(l.messages, fmt.Sprintf("%s: %s", err, msg))
	l.LogSink.Error(err, msg, keysAndValues...)
}

func (l *logSink) String() string {
	out := make([]string, len(l.messages))
	for i := range l.messages {
		out[i] = "\t-" + l.messages[i]
	}
	return strings.Join(out, "\n")
}

func TestGlobalInstRegisterCallback(t *testing.T) {
	l := newLogSink(t)
	otel.SetLogger(logr.New(l))

	const mtrName = "TestGlobalInstRegisterCallback"
	preMtr := global.Meter(mtrName)
	preInt64Ctr, err := preMtr.Int64ObservableCounter("pre.int64.counter")
	require.NoError(t, err)
	preFloat64Ctr, err := preMtr.Float64ObservableCounter("pre.float64.counter")
	require.NoError(t, err)

	rdr := NewManualReader()
	mp := NewMeterProvider(WithReader(rdr), WithResource(resource.Empty()))
	global.SetMeterProvider(mp)

	postMtr := global.Meter(mtrName)
	postInt64Ctr, err := postMtr.Int64ObservableCounter("post.int64.counter")
	require.NoError(t, err)
	postFloat64Ctr, err := postMtr.Float64ObservableCounter("post.float64.counter")
	require.NoError(t, err)

	cb := func(_ context.Context, o metric.Observer) error {
		o.ObserveInt64(preInt64Ctr, 1)
		o.ObserveFloat64(preFloat64Ctr, 2)
		o.ObserveInt64(postInt64Ctr, 3)
		o.ObserveFloat64(postFloat64Ctr, 4)
		return nil
	}

	_, err = preMtr.RegisterCallback(cb, preInt64Ctr, preFloat64Ctr, postInt64Ctr, postFloat64Ctr)
	assert.NoError(t, err)

	_, err = preMtr.RegisterCallback(cb, preInt64Ctr, preFloat64Ctr, postInt64Ctr, postFloat64Ctr)
	assert.NoError(t, err)

	got, err := rdr.Collect(context.Background())
	assert.NoError(t, err)
	assert.Lenf(t, l.messages, 0, "Warnings and errors logged:\n%s", l)
	metricdatatest.AssertEqual(t, metricdata.ResourceMetrics{
		ScopeMetrics: []metricdata.ScopeMetrics{
			{
				Scope: instrumentation.Scope{Name: "TestGlobalInstRegisterCallback"},
				Metrics: []metricdata.Metrics{
					{
						Name: "pre.int64.counter",
						Data: metricdata.Sum[int64]{
							Temporality: metricdata.CumulativeTemporality,
							IsMonotonic: true,
							DataPoints:  []metricdata.DataPoint[int64]{{Value: 1}},
						},
					},
					{
						Name: "pre.float64.counter",
						Data: metricdata.Sum[float64]{
							DataPoints:  []metricdata.DataPoint[float64]{{Value: 2}},
							Temporality: metricdata.CumulativeTemporality,
							IsMonotonic: true,
						},
					},
					{
						Name: "post.int64.counter",
						Data: metricdata.Sum[int64]{
							Temporality: metricdata.CumulativeTemporality,
							IsMonotonic: true,
							DataPoints:  []metricdata.DataPoint[int64]{{Value: 3}},
						},
					},
					{
						Name: "post.float64.counter",
						Data: metricdata.Sum[float64]{
							DataPoints:  []metricdata.DataPoint[float64]{{Value: 4}},
							Temporality: metricdata.CumulativeTemporality,
							IsMonotonic: true,
						},
					},
				},
			},
		},
	}, got, metricdatatest.IgnoreTimestamp())
}

func TestMetersProvideScope(t *testing.T) {
	rdr := NewManualReader()
	mp := NewMeterProvider(WithReader(rdr))

	m1 := mp.Meter("scope1")
	ctr1, err := m1.Float64ObservableCounter("ctr1")
	assert.NoError(t, err)
	_, err = m1.RegisterCallback(func(_ context.Context, o metric.Observer) error {
		o.ObserveFloat64(ctr1, 5)
		return nil
	}, ctr1)
	assert.NoError(t, err)

	m2 := mp.Meter("scope2")
	ctr2, err := m2.Int64ObservableCounter("ctr2")
	assert.NoError(t, err)
	_, err = m2.RegisterCallback(func(_ context.Context, o metric.Observer) error {
		o.ObserveInt64(ctr2, 7)
		return nil
	}, ctr2)
	assert.NoError(t, err)

	want := metricdata.ResourceMetrics{
		Resource: resource.Default(),
		ScopeMetrics: []metricdata.ScopeMetrics{
			{
				Scope: instrumentation.Scope{
					Name: "scope1",
				},
				Metrics: []metricdata.Metrics{
					{
						Name: "ctr1",
						Data: metricdata.Sum[float64]{
							Temporality: metricdata.CumulativeTemporality,
							IsMonotonic: true,
							DataPoints: []metricdata.DataPoint[float64]{
								{
									Value: 5,
								},
							},
						},
					},
				},
			},
			{
				Scope: instrumentation.Scope{
					Name: "scope2",
				},
				Metrics: []metricdata.Metrics{
					{
						Name: "ctr2",
						Data: metricdata.Sum[int64]{
							Temporality: metricdata.CumulativeTemporality,
							IsMonotonic: true,
							DataPoints: []metricdata.DataPoint[int64]{
								{
									Value: 7,
								},
							},
						},
					},
				},
			},
		},
	}

	got, err := rdr.Collect(context.Background())
	assert.NoError(t, err)
	metricdatatest.AssertEqual(t, want, got, metricdatatest.IgnoreTimestamp())
}

func TestUnregisterUnregisters(t *testing.T) {
	r := NewManualReader()
	mp := NewMeterProvider(WithReader(r))
	m := mp.Meter("TestUnregisterUnregisters")

	int64Counter, err := m.Int64ObservableCounter("int64.counter")
	require.NoError(t, err)

	int64UpDownCounter, err := m.Int64ObservableUpDownCounter("int64.up_down_counter")
	require.NoError(t, err)

	int64Gauge, err := m.Int64ObservableGauge("int64.gauge")
	require.NoError(t, err)

	floag64Counter, err := m.Float64ObservableCounter("floag64.counter")
	require.NoError(t, err)

	floag64UpDownCounter, err := m.Float64ObservableUpDownCounter("floag64.up_down_counter")
	require.NoError(t, err)

	floag64Gauge, err := m.Float64ObservableGauge("floag64.gauge")
	require.NoError(t, err)

	var called bool
	reg, err := m.RegisterCallback(
		func(context.Context, metric.Observer) error {
			called = true
			return nil
		},
		int64Counter,
		int64UpDownCounter,
		int64Gauge,
		floag64Counter,
		floag64UpDownCounter,
		floag64Gauge,
	)
	require.NoError(t, err)

	ctx := context.Background()
	_, err = r.Collect(ctx)
	require.NoError(t, err)
	assert.True(t, called, "callback not called for registered callback")

	called = false
	require.NoError(t, reg.Unregister(), "unregister")

	_, err = r.Collect(ctx)
	require.NoError(t, err)
	assert.False(t, called, "callback called for unregistered callback")
}

func TestRegisterCallbackDropAggregations(t *testing.T) {
	aggFn := func(InstrumentKind) aggregation.Aggregation {
		return aggregation.Drop{}
	}
	r := NewManualReader(WithAggregationSelector(aggFn))
	mp := NewMeterProvider(WithReader(r))
	m := mp.Meter("testRegisterCallbackDropAggregations")

	int64Counter, err := m.Int64ObservableCounter("int64.counter")
	require.NoError(t, err)

	int64UpDownCounter, err := m.Int64ObservableUpDownCounter("int64.up_down_counter")
	require.NoError(t, err)

	int64Gauge, err := m.Int64ObservableGauge("int64.gauge")
	require.NoError(t, err)

	floag64Counter, err := m.Float64ObservableCounter("floag64.counter")
	require.NoError(t, err)

	floag64UpDownCounter, err := m.Float64ObservableUpDownCounter("floag64.up_down_counter")
	require.NoError(t, err)

	floag64Gauge, err := m.Float64ObservableGauge("floag64.gauge")
	require.NoError(t, err)

	var called bool
	_, err = m.RegisterCallback(
		func(context.Context, metric.Observer) error {
			called = true
			return nil
		},
		int64Counter,
		int64UpDownCounter,
		int64Gauge,
		floag64Counter,
		floag64UpDownCounter,
		floag64Gauge,
	)
	require.NoError(t, err)

	data, err := r.Collect(context.Background())
	require.NoError(t, err)

	assert.False(t, called, "callback called for all drop instruments")
	assert.Len(t, data.ScopeMetrics, 0, "metrics exported for drop instruments")
}

func TestAttributeFilter(t *testing.T) {
	one := 1.0
	two := 2.0
	testcases := []struct {
		name       string
		register   func(t *testing.T, mtr metric.Meter) error
		wantMetric metricdata.Metrics
	}{
		{
			name: "ObservableFloat64Counter",
			register: func(t *testing.T, mtr metric.Meter) error {
				ctr, err := mtr.Float64ObservableCounter("afcounter")
				if err != nil {
					return err
				}
				_, err = mtr.RegisterCallback(func(_ context.Context, o metric.Observer) error {
					o.ObserveFloat64(ctr, 1.0, attribute.String("foo", "bar"), attribute.Int("version", 1))
					o.ObserveFloat64(ctr, 2.0, attribute.String("foo", "bar"), attribute.Int("version", 2))
					return nil
				}, ctr)
				return err
			},
			wantMetric: metricdata.Metrics{
				Name: "afcounter",
				Data: metricdata.Sum[float64]{
					DataPoints: []metricdata.DataPoint[float64]{
						{
							Attributes: attribute.NewSet(attribute.String("foo", "bar")),
							Value:      2.0, // TODO (#3439): This should be 3.0.
						},
					},
					Temporality: metricdata.CumulativeTemporality,
					IsMonotonic: true,
				},
			},
		},
		{
			name: "ObservableFloat64UpDownCounter",
			register: func(t *testing.T, mtr metric.Meter) error {
				ctr, err := mtr.Float64ObservableUpDownCounter("afupdowncounter")
				if err != nil {
					return err
				}
				_, err = mtr.RegisterCallback(func(_ context.Context, o metric.Observer) error {
					o.ObserveFloat64(ctr, 1.0, attribute.String("foo", "bar"), attribute.Int("version", 1))
					o.ObserveFloat64(ctr, 2.0, attribute.String("foo", "bar"), attribute.Int("version", 2))
					return nil
				}, ctr)
				return err
			},
			wantMetric: metricdata.Metrics{
				Name: "afupdowncounter",
				Data: metricdata.Sum[float64]{
					DataPoints: []metricdata.DataPoint[float64]{
						{
							Attributes: attribute.NewSet(attribute.String("foo", "bar")),
							Value:      2.0, // TODO (#3439): This should be 3.0.
						},
					},
					Temporality: metricdata.CumulativeTemporality,
					IsMonotonic: false,
				},
			},
		},
		{
			name: "ObservableFloat64Gauge",
			register: func(t *testing.T, mtr metric.Meter) error {
				ctr, err := mtr.Float64ObservableGauge("afgauge")
				if err != nil {
					return err
				}
				_, err = mtr.RegisterCallback(func(_ context.Context, o metric.Observer) error {
					o.ObserveFloat64(ctr, 1.0, attribute.String("foo", "bar"), attribute.Int("version", 1))
					o.ObserveFloat64(ctr, 2.0, attribute.String("foo", "bar"), attribute.Int("version", 2))
					return nil
				}, ctr)
				return err
			},
			wantMetric: metricdata.Metrics{
				Name: "afgauge",
				Data: metricdata.Gauge[float64]{
					DataPoints: []metricdata.DataPoint[float64]{
						{
							Attributes: attribute.NewSet(attribute.String("foo", "bar")),
							Value:      2.0,
						},
					},
				},
			},
		},
		{
			name: "ObservableInt64Counter",
			register: func(t *testing.T, mtr metric.Meter) error {
				ctr, err := mtr.Int64ObservableCounter("aicounter")
				if err != nil {
					return err
				}
				_, err = mtr.RegisterCallback(func(_ context.Context, o metric.Observer) error {
					o.ObserveInt64(ctr, 10, attribute.String("foo", "bar"), attribute.Int("version", 1))
					o.ObserveInt64(ctr, 20, attribute.String("foo", "bar"), attribute.Int("version", 2))
					return nil
				}, ctr)
				return err
			},
			wantMetric: metricdata.Metrics{
				Name: "aicounter",
				Data: metricdata.Sum[int64]{
					DataPoints: []metricdata.DataPoint[int64]{
						{
							Attributes: attribute.NewSet(attribute.String("foo", "bar")),
							Value:      20, // TODO (#3439): This should be 30.
						},
					},
					Temporality: metricdata.CumulativeTemporality,
					IsMonotonic: true,
				},
			},
		},
		{
			name: "ObservableInt64UpDownCounter",
			register: func(t *testing.T, mtr metric.Meter) error {
				ctr, err := mtr.Int64ObservableUpDownCounter("aiupdowncounter")
				if err != nil {
					return err
				}
				_, err = mtr.RegisterCallback(func(_ context.Context, o metric.Observer) error {
					o.ObserveInt64(ctr, 10, attribute.String("foo", "bar"), attribute.Int("version", 1))
					o.ObserveInt64(ctr, 20, attribute.String("foo", "bar"), attribute.Int("version", 2))
					return nil
				}, ctr)
				return err
			},
			wantMetric: metricdata.Metrics{
				Name: "aiupdowncounter",
				Data: metricdata.Sum[int64]{
					DataPoints: []metricdata.DataPoint[int64]{
						{
							Attributes: attribute.NewSet(attribute.String("foo", "bar")),
							Value:      20, // TODO (#3439): This should be 30.
						},
					},
					Temporality: metricdata.CumulativeTemporality,
					IsMonotonic: false,
				},
			},
		},
		{
			name: "ObservableInt64Gauge",
			register: func(t *testing.T, mtr metric.Meter) error {
				ctr, err := mtr.Int64ObservableGauge("aigauge")
				if err != nil {
					return err
				}
				_, err = mtr.RegisterCallback(func(_ context.Context, o metric.Observer) error {
					o.ObserveInt64(ctr, 10, attribute.String("foo", "bar"), attribute.Int("version", 1))
					o.ObserveInt64(ctr, 20, attribute.String("foo", "bar"), attribute.Int("version", 2))
					return nil
				}, ctr)
				return err
			},
			wantMetric: metricdata.Metrics{
				Name: "aigauge",
				Data: metricdata.Gauge[int64]{
					DataPoints: []metricdata.DataPoint[int64]{
						{
							Attributes: attribute.NewSet(attribute.String("foo", "bar")),
							Value:      20,
						},
					},
				},
			},
		},
		{
			name: "SyncFloat64Counter",
			register: func(t *testing.T, mtr metric.Meter) error {
				ctr, err := mtr.Float64Counter("sfcounter")
				if err != nil {
					return err
				}

				ctr.Add(context.Background(), 1.0, attribute.String("foo", "bar"), attribute.Int("version", 1))
				ctr.Add(context.Background(), 2.0, attribute.String("foo", "bar"), attribute.Int("version", 2))
				return nil
			},
			wantMetric: metricdata.Metrics{
				Name: "sfcounter",
				Data: metricdata.Sum[float64]{
					DataPoints: []metricdata.DataPoint[float64]{
						{
							Attributes: attribute.NewSet(attribute.String("foo", "bar")),
							Value:      3.0,
						},
					},
					Temporality: metricdata.CumulativeTemporality,
					IsMonotonic: true,
				},
			},
		},
		{
			name: "SyncFloat64UpDownCounter",
			register: func(t *testing.T, mtr metric.Meter) error {
				ctr, err := mtr.Float64UpDownCounter("sfupdowncounter")
				if err != nil {
					return err
				}

				ctr.Add(context.Background(), 1.0, attribute.String("foo", "bar"), attribute.Int("version", 1))
				ctr.Add(context.Background(), 2.0, attribute.String("foo", "bar"), attribute.Int("version", 2))
				return nil
			},
			wantMetric: metricdata.Metrics{
				Name: "sfupdowncounter",
				Data: metricdata.Sum[float64]{
					DataPoints: []metricdata.DataPoint[float64]{
						{
							Attributes: attribute.NewSet(attribute.String("foo", "bar")),
							Value:      3.0,
						},
					},
					Temporality: metricdata.CumulativeTemporality,
					IsMonotonic: false,
				},
			},
		},
		{
			name: "SyncFloat64Histogram",
			register: func(t *testing.T, mtr metric.Meter) error {
				ctr, err := mtr.Float64Histogram("sfhistogram")
				if err != nil {
					return err
				}

				ctr.Record(context.Background(), 1.0, attribute.String("foo", "bar"), attribute.Int("version", 1))
				ctr.Record(context.Background(), 2.0, attribute.String("foo", "bar"), attribute.Int("version", 2))
				return nil
			},
			wantMetric: metricdata.Metrics{
				Name: "sfhistogram",
				Data: metricdata.Histogram{
					DataPoints: []metricdata.HistogramDataPoint{
						{
							Attributes:   attribute.NewSet(attribute.String("foo", "bar")),
							Bounds:       []float64{0, 5, 10, 25, 50, 75, 100, 250, 500, 750, 1000, 2500, 5000, 7500, 10000},
							BucketCounts: []uint64{0, 2, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
							Count:        2,
							Min:          &one,
							Max:          &two,
							Sum:          3.0,
						},
					},
					Temporality: metricdata.CumulativeTemporality,
				},
			},
		},
		{
			name: "SyncInt64Counter",
			register: func(t *testing.T, mtr metric.Meter) error {
				ctr, err := mtr.Int64Counter("sicounter")
				if err != nil {
					return err
				}

				ctr.Add(context.Background(), 10, attribute.String("foo", "bar"), attribute.Int("version", 1))
				ctr.Add(context.Background(), 20, attribute.String("foo", "bar"), attribute.Int("version", 2))
				return nil
			},
			wantMetric: metricdata.Metrics{
				Name: "sicounter",
				Data: metricdata.Sum[int64]{
					DataPoints: []metricdata.DataPoint[int64]{
						{
							Attributes: attribute.NewSet(attribute.String("foo", "bar")),
							Value:      30,
						},
					},
					Temporality: metricdata.CumulativeTemporality,
					IsMonotonic: true,
				},
			},
		},
		{
			name: "SyncInt64UpDownCounter",
			register: func(t *testing.T, mtr metric.Meter) error {
				ctr, err := mtr.Int64UpDownCounter("siupdowncounter")
				if err != nil {
					return err
				}

				ctr.Add(context.Background(), 10, attribute.String("foo", "bar"), attribute.Int("version", 1))
				ctr.Add(context.Background(), 20, attribute.String("foo", "bar"), attribute.Int("version", 2))
				return nil
			},
			wantMetric: metricdata.Metrics{
				Name: "siupdowncounter",
				Data: metricdata.Sum[int64]{
					DataPoints: []metricdata.DataPoint[int64]{
						{
							Attributes: attribute.NewSet(attribute.String("foo", "bar")),
							Value:      30,
						},
					},
					Temporality: metricdata.CumulativeTemporality,
					IsMonotonic: false,
				},
			},
		},
		{
			name: "SyncInt64Histogram",
			register: func(t *testing.T, mtr metric.Meter) error {
				ctr, err := mtr.Int64Histogram("sihistogram")
				if err != nil {
					return err
				}

				ctr.Record(context.Background(), 1, attribute.String("foo", "bar"), attribute.Int("version", 1))
				ctr.Record(context.Background(), 2, attribute.String("foo", "bar"), attribute.Int("version", 2))
				return nil
			},
			wantMetric: metricdata.Metrics{
				Name: "sihistogram",
				Data: metricdata.Histogram{
					DataPoints: []metricdata.HistogramDataPoint{
						{
							Attributes:   attribute.NewSet(attribute.String("foo", "bar")),
							Bounds:       []float64{0, 5, 10, 25, 50, 75, 100, 250, 500, 750, 1000, 2500, 5000, 7500, 10000},
							BucketCounts: []uint64{0, 2, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
							Count:        2,
							Min:          &one,
							Max:          &two,
							Sum:          3.0,
						},
					},
					Temporality: metricdata.CumulativeTemporality,
				},
			},
		},
	}

	for _, tt := range testcases {
		t.Run(tt.name, func(t *testing.T) {
			rdr := NewManualReader()
			mtr := NewMeterProvider(
				WithReader(rdr),
				WithView(NewView(
					Instrument{Name: "*"},
					Stream{AttributeFilter: func(kv attribute.KeyValue) bool {
						return kv.Key == attribute.Key("foo")
					}},
				)),
			).Meter("TestAttributeFilter")
			require.NoError(t, tt.register(t, mtr))

			m, err := rdr.Collect(context.Background())
			assert.NoError(t, err)

			require.Len(t, m.ScopeMetrics, 1)
			require.Len(t, m.ScopeMetrics[0].Metrics, 1)

			metricdatatest.AssertEqual(t, tt.wantMetric, m.ScopeMetrics[0].Metrics[0], metricdatatest.IgnoreTimestamp())
		})
	}
}

var (
	aiCounter       instrument.Int64ObservableCounter
	aiUpDownCounter instrument.Int64ObservableUpDownCounter
	aiGauge         instrument.Int64ObservableGauge

	afCounter       instrument.Float64ObservableCounter
	afUpDownCounter instrument.Float64ObservableUpDownCounter
	afGauge         instrument.Float64ObservableGauge

	siCounter       instrument.Int64Counter
	siUpDownCounter instrument.Int64UpDownCounter
	siHistogram     instrument.Int64Histogram

	sfCounter       instrument.Float64Counter
	sfUpDownCounter instrument.Float64UpDownCounter
	sfHistogram     instrument.Float64Histogram
)

func BenchmarkInstrumentCreation(b *testing.B) {
	provider := NewMeterProvider(WithReader(NewManualReader()))
	meter := provider.Meter("BenchmarkInstrumentCreation")

	b.ReportAllocs()
	b.ResetTimer()

	for n := 0; n < b.N; n++ {
		aiCounter, _ = meter.Int64ObservableCounter("observable.int64.counter")
		aiUpDownCounter, _ = meter.Int64ObservableUpDownCounter("observable.int64.up.down.counter")
		aiGauge, _ = meter.Int64ObservableGauge("observable.int64.gauge")

		afCounter, _ = meter.Float64ObservableCounter("observable.float64.counter")
		afUpDownCounter, _ = meter.Float64ObservableUpDownCounter("observable.float64.up.down.counter")
		afGauge, _ = meter.Float64ObservableGauge("observable.float64.gauge")

		siCounter, _ = meter.Int64Counter("sync.int64.counter")
		siUpDownCounter, _ = meter.Int64UpDownCounter("sync.int64.up.down.counter")
		siHistogram, _ = meter.Int64Histogram("sync.int64.histogram")

		sfCounter, _ = meter.Float64Counter("sync.float64.counter")
		sfUpDownCounter, _ = meter.Float64UpDownCounter("sync.float64.up.down.counter")
		sfHistogram, _ = meter.Float64Histogram("sync.float64.histogram")
	}
}
