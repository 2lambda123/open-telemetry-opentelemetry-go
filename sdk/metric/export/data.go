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

//go:build go1.18
// +build go1.18

// TODO: NOTE this is a temporary space, it may be moved following the
// discussion of #2813, or #2841

package export // import "go.opentelemetry.io/otel/sdk/metric/export"

import (
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric/unit"
	"go.opentelemetry.io/otel/sdk/instrumentation"
	"go.opentelemetry.io/otel/sdk/resource"
)

// ResourceMetrics is a collection of ScopeMetrics and the associated Resource
// that created them.
type ResourceMetrics struct {
	// Resource represents the entity that collected the metrics.
	Resource *resource.Resource
	// ScopeMetrics are the collection of metrics with unique Scopes.
	ScopeMetrics []ScopeMetrics
}

// ScopeMetrics is a collection of Metrics Produces by a Meter.
type ScopeMetrics struct {
	// Scope is the Scope that the Meter was created with.
	Scope instrumentation.Scope
	// Metrics are a list of aggregations created by the Meter.
	Metrics []Metrics
}

// Metrics is a collection of one or more aggregated timeseries from an Instrument.
type Metrics struct {
	// Name is the name of the Instrument that created this data.
	Name string
	// Description is the description of the Instrument, which can be used in documentation.
	Description string
	// Unit is the unit in which the Instrument reports.
	Unit unit.Unit
	// Data is the aggregated data from an Instrument.
	Data Aggregation
}

// Aggregation is the store of data reported by an Instrument.
// It will be one of: Gauge, Sum, Histogram.
type Aggregation interface {
	privateAggregation()
}

// Gauge represents a measurement of the current value of an instrument.
type Gauge struct {
	// DataPoints reprents individual aggregated measurements with unique Attributes.
	DataPoints []DataPoint
}

func (Gauge) privateAggregation() {}

// Sum represents the sum of all measurements of values from an instrument.
type Sum struct {
	// DataPoints reprents individual aggregated measurements with unique Attributes.
	DataPoints []DataPoint
	// Temporality describes if the aggregation is reported as the change from the
	// last report time, or the cumulative changes since a fixed start time.
	Temporality Temporality
	// IsMonotonic represents if this aggregation only increases or decreases.
	IsMonotonic bool
}

func (Sum) privateAggregation() {}

// DataPoint is a single data point in a timeseries.
type DataPoint struct {
	// Attributes is the set of key value pairs that uniquely identify the timeseries.
	Attributes []attribute.KeyValue
	// StartTime is when the timeseries was started. (optional)
	StartTime time.Time
	// Time is the time when the timeseries was recorded. (optional)
	Time time.Time
	// Value is the value of this data point.
	Value Value
}

// Value is a int64 or float64. All Values created by the sdk will be either
// Int64 or Float64.
type Value interface {
	privateValue()
}

// Int64 is a container for an int64 value.
type Int64 int64

func (Int64) privateValue() {}

// Float64 is a container for a float64 value.
type Float64 float64

func (Float64) privateValue() {}

// Histogram represents the histogram of all measurements of values from an instrument.
type Histogram struct {
	// DataPoints reprents individual aggregated measurements with unique Attributes.
	DataPoints []HistogramDataPoint
	// Temporality describes if the aggregation is reported as the change from the
	// last report time, or the cumulative changes since a fixed start time.
	Temporality Temporality
}

func (Histogram) privateAggregation() {}

// HistogramDataPoint is a single histogram data point in a timeseries.
type HistogramDataPoint struct {
	// Attributes is the set of key value pairs that uniquely identify the timeseries.
	Attributes []attribute.KeyValue
	// StartTime is when the timeseries was started.
	StartTime time.Time
	// Time is the time when the timeseries was recorded.
	Time time.Time

	// Count is the number of updates this histogram has been calculated with.
	Count uint64
	// Bounds are the upper bounds of the buckets of the histogram. Because the
	// last boundary is +infinity this one is implied.
	Bounds []float64
	// BucketCounts is the count of each of the buckets.
	BucketCounts []uint64

	// Min is the minimum value recorded. (optional)
	Min *float64
	// Max is the maximum value recorded. (optional)
	Max *float64
	// Sum is the sum of the values recorded.
	Sum float64
}
