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

package metrictransform

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric/metrictest"
	"go.opentelemetry.io/otel/metric/number"
	"go.opentelemetry.io/otel/metric/sdkapi"
	export "go.opentelemetry.io/otel/sdk/export/metric"
	"go.opentelemetry.io/otel/sdk/export/metric/aggregation"
	"go.opentelemetry.io/otel/sdk/metric/aggregator/exponential"
	"go.opentelemetry.io/otel/sdk/metric/aggregator/lastvalue"
	"go.opentelemetry.io/otel/sdk/metric/aggregator/minmaxsumcount"
	"go.opentelemetry.io/otel/sdk/metric/aggregator/sum"
	commonpb "go.opentelemetry.io/proto/otlp/common/v1"
	metricpb "go.opentelemetry.io/proto/otlp/metrics/v1"
)

var (
	// Timestamps used in this test:

	intervalStart = time.Now()
	intervalEnd   = intervalStart.Add(time.Hour)
)

const (
	otelCumulative = metricpb.AggregationTemporality_AGGREGATION_TEMPORALITY_CUMULATIVE
	otelDelta      = metricpb.AggregationTemporality_AGGREGATION_TEMPORALITY_DELTA
)

func TestStringKeyValues(t *testing.T) {
	tests := []struct {
		kvs      []attribute.KeyValue
		expected []*commonpb.KeyValue
	}{
		{
			nil,
			nil,
		},
		{
			[]attribute.KeyValue{},
			nil,
		},
		{
			[]attribute.KeyValue{
				attribute.Bool("true", true),
				attribute.Int64("one", 1),
				attribute.Int64("two", 2),
				attribute.Float64("three", 3),
				attribute.Int("four", 4),
				attribute.Int("five", 5),
				attribute.Float64("six", 6),
				attribute.Int("seven", 7),
				attribute.Int("eight", 8),
				attribute.String("the", "final word"),
			},
			[]*commonpb.KeyValue{
				{Key: "eight", Value: &commonpb.AnyValue{Value: &commonpb.AnyValue_IntValue{IntValue: 8}}},
				{Key: "five", Value: &commonpb.AnyValue{Value: &commonpb.AnyValue_IntValue{IntValue: 5}}},
				{Key: "four", Value: &commonpb.AnyValue{Value: &commonpb.AnyValue_IntValue{IntValue: 4}}},
				{Key: "one", Value: &commonpb.AnyValue{Value: &commonpb.AnyValue_IntValue{IntValue: 1}}},
				{Key: "seven", Value: &commonpb.AnyValue{Value: &commonpb.AnyValue_IntValue{IntValue: 7}}},
				{Key: "six", Value: &commonpb.AnyValue{Value: &commonpb.AnyValue_DoubleValue{DoubleValue: 6.0}}},
				{Key: "the", Value: &commonpb.AnyValue{Value: &commonpb.AnyValue_StringValue{StringValue: "final word"}}},
				{Key: "three", Value: &commonpb.AnyValue{Value: &commonpb.AnyValue_DoubleValue{DoubleValue: 3.0}}},
				{Key: "true", Value: &commonpb.AnyValue{Value: &commonpb.AnyValue_BoolValue{BoolValue: true}}},
				{Key: "two", Value: &commonpb.AnyValue{Value: &commonpb.AnyValue_IntValue{IntValue: 2}}},
			},
		},
	}

	for _, test := range tests {
		labels := attribute.NewSet(test.kvs...)
		assert.Equal(t, test.expected, Iterator(labels.Iter()))
	}
}

func TestMinMaxSumCountValue(t *testing.T) {
	mmscs := minmaxsumcount.New(2, &sdkapi.Descriptor{})
	mmsc, ckpt := &mmscs[0], &mmscs[1]

	assert.NoError(t, mmsc.Update(context.Background(), 1, &sdkapi.Descriptor{}))
	assert.NoError(t, mmsc.Update(context.Background(), 10, &sdkapi.Descriptor{}))

	// Prior to checkpointing ErrNoData should be returned.
	_, _, _, _, err := minMaxSumCountValues(ckpt)
	assert.EqualError(t, err, aggregation.ErrNoData.Error())

	// Checkpoint to set non-zero values
	require.NoError(t, mmsc.SynchronizedMove(ckpt, &sdkapi.Descriptor{}))
	min, max, sum, count, err := minMaxSumCountValues(ckpt)
	if assert.NoError(t, err) {
		assert.Equal(t, min, number.NewInt64Number(1))
		assert.Equal(t, max, number.NewInt64Number(10))
		assert.Equal(t, sum, number.NewInt64Number(11))
		assert.Equal(t, count, uint64(2))
	}
}

func TestMinMaxSumCountDatapoints(t *testing.T) {
	desc := metrictest.NewDescriptor("", sdkapi.HistogramInstrumentKind, number.Int64Kind)
	labels := attribute.NewSet(attribute.String("one", "1"))
	mmscs := minmaxsumcount.New(2, &sdkapi.Descriptor{})
	mmsc, ckpt := &mmscs[0], &mmscs[1]

	assert.NoError(t, mmsc.Update(context.Background(), 1, &desc))
	assert.NoError(t, mmsc.Update(context.Background(), 10, &desc))
	require.NoError(t, mmsc.SynchronizedMove(ckpt, &desc))
	expected := []*metricpb.SummaryDataPoint{
		{
			Count:             2,
			Sum:               11,
			StartTimeUnixNano: uint64(intervalStart.UnixNano()),
			TimeUnixNano:      uint64(intervalEnd.UnixNano()),
			Attributes: []*commonpb.KeyValue{
				{
					Key:   "one",
					Value: &commonpb.AnyValue{Value: &commonpb.AnyValue_StringValue{StringValue: "1"}},
				},
			},
			QuantileValues: []*metricpb.SummaryDataPoint_ValueAtQuantile{
				{
					Quantile: 0.0,
					Value:    1.0,
				},
				{
					Quantile: 1.0,
					Value:    10.0,
				},
			},
		},
	}
	record := export.NewRecord(&desc, &labels, ckpt.Aggregation(), intervalStart, intervalEnd)
	m, err := minMaxSumCount(record, ckpt)
	if assert.NoError(t, err) {
		assert.Nil(t, m.GetGauge())
		assert.Nil(t, m.GetSum())
		assert.Nil(t, m.GetHistogram())
		assert.Equal(t, expected, m.GetSummary().DataPoints)
		assert.Nil(t, m.GetIntGauge())     // nolint
		assert.Nil(t, m.GetIntSum())       // nolint
		assert.Nil(t, m.GetIntHistogram()) // nolint
	}
}

func TestMinMaxSumCountPropagatesErrors(t *testing.T) {
	// ErrNoData should be returned by both the Min and Max values of
	// a MinMaxSumCount Aggregator. Use this fact to check the error is
	// correctly returned.
	mmsc := &minmaxsumcount.New(1, &sdkapi.Descriptor{})[0]
	_, _, _, _, err := minMaxSumCountValues(mmsc)
	assert.Error(t, err)
	assert.Equal(t, aggregation.ErrNoData, err)
}

func TestSumIntDataPoints(t *testing.T) {
	desc := metrictest.NewDescriptor("", sdkapi.HistogramInstrumentKind, number.Int64Kind)
	labels := attribute.NewSet(attribute.String("one", "1"))
	sums := sum.New(2)
	s, ckpt := &sums[0], &sums[1]

	assert.NoError(t, s.Update(context.Background(), number.Number(1), &desc))
	require.NoError(t, s.SynchronizedMove(ckpt, &desc))
	record := export.NewRecord(&desc, &labels, ckpt.Aggregation(), intervalStart, intervalEnd)

	value, err := ckpt.Sum()
	require.NoError(t, err)

	if m, err := sumPoint(record, value, record.StartTime(), record.EndTime(), aggregation.CumulativeTemporality, true); assert.NoError(t, err) {
		assert.Nil(t, m.GetGauge())
		assert.Equal(t, &metricpb.Sum{
			AggregationTemporality: otelCumulative,
			IsMonotonic:            true,
			DataPoints: []*metricpb.NumberDataPoint{{
				StartTimeUnixNano: uint64(intervalStart.UnixNano()),
				TimeUnixNano:      uint64(intervalEnd.UnixNano()),
				Attributes: []*commonpb.KeyValue{
					{
						Key:   "one",
						Value: &commonpb.AnyValue{Value: &commonpb.AnyValue_StringValue{StringValue: "1"}},
					},
				},
				Value: &metricpb.NumberDataPoint_AsInt{
					AsInt: 1,
				},
			}},
		}, m.GetSum())
		assert.Nil(t, m.GetHistogram())
		assert.Nil(t, m.GetSummary())
		assert.Nil(t, m.GetIntGauge())     // nolint
		assert.Nil(t, m.GetIntSum())       // nolint
		assert.Nil(t, m.GetIntHistogram()) // nolint
	}
}

func TestSumFloatDataPoints(t *testing.T) {
	desc := metrictest.NewDescriptor("", sdkapi.HistogramInstrumentKind, number.Float64Kind)
	labels := attribute.NewSet(attribute.String("one", "1"))
	sums := sum.New(2)
	s, ckpt := &sums[0], &sums[1]

	assert.NoError(t, s.Update(context.Background(), number.NewFloat64Number(1), &desc))
	require.NoError(t, s.SynchronizedMove(ckpt, &desc))
	record := export.NewRecord(&desc, &labels, ckpt.Aggregation(), intervalStart, intervalEnd)
	value, err := ckpt.Sum()
	require.NoError(t, err)

	if m, err := sumPoint(record, value, record.StartTime(), record.EndTime(), aggregation.DeltaTemporality, false); assert.NoError(t, err) {
		assert.Nil(t, m.GetGauge())
		assert.Equal(t, &metricpb.Sum{
			IsMonotonic:            false,
			AggregationTemporality: otelDelta,
			DataPoints: []*metricpb.NumberDataPoint{{
				Value: &metricpb.NumberDataPoint_AsDouble{
					AsDouble: 1.0,
				},
				StartTimeUnixNano: uint64(intervalStart.UnixNano()),
				TimeUnixNano:      uint64(intervalEnd.UnixNano()),
				Attributes: []*commonpb.KeyValue{
					{
						Key:   "one",
						Value: &commonpb.AnyValue{Value: &commonpb.AnyValue_StringValue{StringValue: "1"}},
					},
				},
			}}}, m.GetSum())
		assert.Nil(t, m.GetHistogram())
		assert.Nil(t, m.GetSummary())
		assert.Nil(t, m.GetIntGauge())     // nolint
		assert.Nil(t, m.GetIntSum())       // nolint
		assert.Nil(t, m.GetIntHistogram()) // nolint
	}
}

func TestLastValueIntDataPoints(t *testing.T) {
	desc := metrictest.NewDescriptor("", sdkapi.HistogramInstrumentKind, number.Int64Kind)
	labels := attribute.NewSet(attribute.String("one", "1"))
	lvs := lastvalue.New(2)
	lv, ckpt := &lvs[0], &lvs[1]

	assert.NoError(t, lv.Update(context.Background(), number.Number(100), &desc))
	require.NoError(t, lv.SynchronizedMove(ckpt, &desc))
	record := export.NewRecord(&desc, &labels, ckpt.Aggregation(), intervalStart, intervalEnd)
	value, timestamp, err := ckpt.LastValue()
	require.NoError(t, err)

	if m, err := gaugePoint(record, value, time.Time{}, timestamp); assert.NoError(t, err) {
		assert.Equal(t, []*metricpb.NumberDataPoint{{
			StartTimeUnixNano: 0,
			TimeUnixNano:      uint64(timestamp.UnixNano()),
			Attributes: []*commonpb.KeyValue{
				{
					Key:   "one",
					Value: &commonpb.AnyValue{Value: &commonpb.AnyValue_StringValue{StringValue: "1"}},
				},
			},
			Value: &metricpb.NumberDataPoint_AsInt{
				AsInt: 100,
			},
		}}, m.GetGauge().DataPoints)
		assert.Nil(t, m.GetSum())
		assert.Nil(t, m.GetHistogram())
		assert.Nil(t, m.GetSummary())
		assert.Nil(t, m.GetIntGauge())     // nolint
		assert.Nil(t, m.GetIntSum())       // nolint
		assert.Nil(t, m.GetIntHistogram()) // nolint
	}
}

func TestSumErrUnknownValueType(t *testing.T) {
	desc := metrictest.NewDescriptor("", sdkapi.HistogramInstrumentKind, number.Kind(-1))
	labels := attribute.NewSet()
	s := &sum.New(1)[0]
	record := export.NewRecord(&desc, &labels, s, intervalStart, intervalEnd)
	value, err := s.Sum()
	require.NoError(t, err)

	_, err = sumPoint(record, value, record.StartTime(), record.EndTime(), aggregation.CumulativeTemporality, true)
	assert.Error(t, err)
	if !errors.Is(err, ErrUnknownValueType) {
		t.Errorf("expected ErrUnknownValueType, got %v", err)
	}
}

type testAgg struct {
	kind aggregation.Kind
	agg  aggregation.Aggregation
}

func (t *testAgg) Kind() aggregation.Kind {
	return t.kind
}

func (t *testAgg) Aggregation() aggregation.Aggregation {
	return t.agg
}

// None of these three are used:

func (t *testAgg) Update(ctx context.Context, number number.Number, descriptor *sdkapi.Descriptor) error {
	return nil
}
func (t *testAgg) SynchronizedMove(destination export.Aggregator, descriptor *sdkapi.Descriptor) error {
	return nil
}
func (t *testAgg) Merge(aggregator export.Aggregator, descriptor *sdkapi.Descriptor) error {
	return nil
}

type testErrSum struct {
	err error
}

type testErrLastValue struct {
	err error
}

type testErrMinMaxSumCount struct {
	testErrSum
}

func (te *testErrLastValue) LastValue() (number.Number, time.Time, error) {
	return 0, time.Time{}, te.err
}
func (te *testErrLastValue) Kind() aggregation.Kind {
	return aggregation.LastValueKind
}

func (te *testErrSum) Sum() (number.Number, error) {
	return 0, te.err
}
func (te *testErrSum) Kind() aggregation.Kind {
	return aggregation.SumKind
}

func (te *testErrMinMaxSumCount) Min() (number.Number, error) {
	return 0, te.err
}

func (te *testErrMinMaxSumCount) Max() (number.Number, error) {
	return 0, te.err
}

func (te *testErrMinMaxSumCount) Count() (uint64, error) {
	return 0, te.err
}

var _ export.Aggregator = &testAgg{}
var _ aggregation.Aggregation = &testAgg{}
var _ aggregation.Sum = &testErrSum{}
var _ aggregation.LastValue = &testErrLastValue{}
var _ aggregation.MinMaxSumCount = &testErrMinMaxSumCount{}

func TestRecordAggregatorIncompatibleErrors(t *testing.T) {
	makeMpb := func(kind aggregation.Kind, agg aggregation.Aggregation) (*metricpb.Metric, error) {
		desc := metrictest.NewDescriptor("things", sdkapi.CounterInstrumentKind, number.Int64Kind)
		labels := attribute.NewSet()
		test := &testAgg{
			kind: kind,
			agg:  agg,
		}
		return Record(aggregation.CumulativeTemporalitySelector(), export.NewRecord(&desc, &labels, test, intervalStart, intervalEnd))
	}

	mpb, err := makeMpb(aggregation.SumKind, &lastvalue.New(1)[0])

	require.Error(t, err)
	require.Nil(t, mpb)
	require.True(t, errors.Is(err, ErrIncompatibleAgg))

	mpb, err = makeMpb(aggregation.LastValueKind, &sum.New(1)[0])

	require.Error(t, err)
	require.Nil(t, mpb)
	require.True(t, errors.Is(err, ErrIncompatibleAgg))

	mpb, err = makeMpb(aggregation.MinMaxSumCountKind, &lastvalue.New(1)[0])

	require.Error(t, err)
	require.Nil(t, mpb)
	require.True(t, errors.Is(err, ErrIncompatibleAgg))
}

func TestRecordAggregatorUnexpectedErrors(t *testing.T) {
	makeMpb := func(kind aggregation.Kind, agg aggregation.Aggregation) (*metricpb.Metric, error) {
		desc := metrictest.NewDescriptor("things", sdkapi.CounterInstrumentKind, number.Int64Kind)
		labels := attribute.NewSet()
		return Record(aggregation.CumulativeTemporalitySelector(), export.NewRecord(&desc, &labels, agg, intervalStart, intervalEnd))
	}

	errEx := fmt.Errorf("timeout")

	mpb, err := makeMpb(aggregation.SumKind, &testErrSum{errEx})

	require.Error(t, err)
	require.Nil(t, mpb)
	require.True(t, errors.Is(err, errEx))

	mpb, err = makeMpb(aggregation.LastValueKind, &testErrLastValue{errEx})

	require.Error(t, err)
	require.Nil(t, mpb)
	require.True(t, errors.Is(err, errEx))

	mpb, err = makeMpb(aggregation.MinMaxSumCountKind, &testErrMinMaxSumCount{testErrSum{errEx}})

	require.Error(t, err)
	require.Nil(t, mpb)
	require.True(t, errors.Is(err, errEx))
}

func TestExponentialHistogramDataPoints(t *testing.T) {
	type testCase struct {
		name        string
		values      []float64
		temporality aggregation.Temporality
		numberKind  number.Kind
		expectSum   number.Number
		expect      *metricpb.ExponentialHistogram
	}
	useAttrs := []attribute.KeyValue{
		attribute.String("one", "1"),
	}
	expectAttrs := []*commonpb.KeyValue{
		{Key: "one", Value: &commonpb.AnyValue{Value: &commonpb.AnyValue_StringValue{StringValue: "1"}}},
	}

	for _, test := range []testCase{
		{
			"empty",
			[]float64{},
			aggregation.DeltaTemporality,
			number.Float64Kind,
			0,
			&metricpb.ExponentialHistogram{
				AggregationTemporality: otelDelta,
				DataPoints: []*metricpb.ExponentialHistogramDataPoint{{
					Attributes:        expectAttrs,
					StartTimeUnixNano: uint64(intervalStart.UnixNano()),
					TimeUnixNano:      uint64(intervalEnd.UnixNano()),
					Count:             0,
					ZeroCount:         0,
					Sum:               0,
				}},
			},
		},
		{
			"positive",
			[]float64{1, 2, 4, 8},
			aggregation.DeltaTemporality,
			number.Float64Kind,
			0,
			&metricpb.ExponentialHistogram{
				AggregationTemporality: otelDelta,
				DataPoints: []*metricpb.ExponentialHistogramDataPoint{{
					Attributes:        expectAttrs,
					StartTimeUnixNano: uint64(intervalStart.UnixNano()),
					TimeUnixNano:      uint64(intervalEnd.UnixNano()),

					// 1..8 spans 3 orders of magnitide, max-size 2, thus scale=-1
					Scale:     -1,
					Count:     4,
					ZeroCount: 0,
					Sum:       15,
					Positive: &metricpb.ExponentialHistogramDataPoint_Buckets{
						Offset:       0,
						BucketCounts: []uint64{2, 2},
					},
				}},
			},
		},
		{
			"positive_and_negative",
			[]float64{2, 3, -100},
			aggregation.DeltaTemporality,
			number.Float64Kind,
			0,
			&metricpb.ExponentialHistogram{
				AggregationTemporality: otelDelta,
				DataPoints: []*metricpb.ExponentialHistogramDataPoint{{
					Attributes:        expectAttrs,
					StartTimeUnixNano: uint64(intervalStart.UnixNano()),
					TimeUnixNano:      uint64(intervalEnd.UnixNano()),

					// Scale 1 has boundaries at 1, sqrt(2), 2, 2*sqrt(2), ...
					Scale:     1,
					Count:     3,
					ZeroCount: 0,
					Sum:       -95,
					Positive: &metricpb.ExponentialHistogramDataPoint_Buckets{
						// Index 2 => 2, Index 3 => 2*sqrt(2)
						Offset:       2,
						BucketCounts: []uint64{1, 1},
					},
					Negative: &metricpb.ExponentialHistogramDataPoint_Buckets{
						// Index 13 = 2^floor(13/2) * sqrt(2) ~= 90
						Offset:       13,
						BucketCounts: []uint64{1},
					},
				}},
			},
		},
		{
			// Note: (2**(2**-10))**-100 = 0.9345
			// Note: (2**(2**-10))**-101 = 0.9339
			// Note: (2**(2**-10))**-102 = 0.9333
			"negative and zero",
			[]float64{-0.9343, -0.9342, -0.9341, -0.9338, -0.9337, -0.9336, 0, 0, 0, 0},
			aggregation.DeltaTemporality,
			number.Float64Kind,
			0,
			&metricpb.ExponentialHistogram{
				AggregationTemporality: otelDelta,
				DataPoints: []*metricpb.ExponentialHistogramDataPoint{{
					Attributes:        expectAttrs,
					StartTimeUnixNano: uint64(intervalStart.UnixNano()),
					TimeUnixNano:      uint64(intervalEnd.UnixNano()),

					// Scale 1 has boundaries at 1, sqrt(2), 2, 2*sqrt(2), ...
					Scale:     10,
					Count:     10,
					ZeroCount: 4,
					Sum:       -0.9343 + -0.9342 + -0.9341 + -0.9338 + -0.9337 + -0.9336,
					Negative: &metricpb.ExponentialHistogramDataPoint_Buckets{
						Offset:       -102,
						BucketCounts: []uint64{3, 3},
					},
				}},
			},
		},
		{
			"integers cumulative",
			// Scale=-1 has base 4,
			// index 0 holds values [1, 4), has count 2
			// index 1 holds values [4, 15), has count 12
			[]float64{2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15},
			aggregation.CumulativeTemporality,
			number.Float64Kind,
			0,
			&metricpb.ExponentialHistogram{
				AggregationTemporality: otelCumulative,
				DataPoints: []*metricpb.ExponentialHistogramDataPoint{{
					Attributes:        expectAttrs,
					StartTimeUnixNano: uint64(intervalStart.UnixNano()),
					TimeUnixNano:      uint64(intervalEnd.UnixNano()),
					Count:             14,
					ZeroCount:         0,
					Sum:               119,
					Scale:             -1,
					Positive: &metricpb.ExponentialHistogramDataPoint_Buckets{
						Offset:       0,
						BucketCounts: []uint64{2, 12},
					},
				}},
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			desc := metrictest.NewDescriptor("ignore", sdkapi.HistogramInstrumentKind, test.numberKind)
			labels := attribute.NewSet(useAttrs...)
			agg := &exponential.New(1, &desc, exponential.WithMaxSize(2))[0]

			for _, value := range test.values {
				var num number.Number
				if test.numberKind == number.Float64Kind {
					num = number.NewFloat64Number(value)
				} else {
					num = number.NewInt64Number(int64(value))
				}
				assert.NoError(t, agg.Update(context.Background(), num, &desc))
			}

			record := export.NewRecord(&desc, &labels, agg, intervalStart, intervalEnd)

			if m, err := exponentialHistogramPoint(record, test.temporality, agg); assert.NoError(t, err) {
				assert.Equal(t, test.expect, m.GetExponentialHistogram())
			}
		})
	}
}
