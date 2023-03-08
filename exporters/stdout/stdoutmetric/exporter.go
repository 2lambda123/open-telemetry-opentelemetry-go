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

package stdoutmetric // import "go.opentelemetry.io/otel/exporters/stdout/stdoutmetric"

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"

	"go.opentelemetry.io/otel/internal/global"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/aggregation"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
)

// exporter is an OpenTelemetry metric exporter.
type exporter struct {
	encVal atomic.Value // encoderHolder

	shutdownOnce sync.Once

	temporalitySelector metric.TemporalitySelector
	aggregationSelector metric.AggregationSelector

	redactTimestamps bool
}

// New returns a configured metric exporter.
//
// If no options are passed, the default exporter returned will use a JSON
// encoder with tab indentations that output to STDOUT.
func New(options ...Option) (metric.Exporter, error) {
	cfg := newConfig(options...)
	exp := &exporter{
		temporalitySelector: cfg.temporalitySelector,
		aggregationSelector: cfg.aggregationSelector,
		redactTimestamps:    cfg.redactTimestamps,
	}
	exp.encVal.Store(*cfg.encoder)
	return exp, nil
}

func (e *exporter) Temporality(k metric.InstrumentKind) metricdata.Temporality {
	return e.temporalitySelector(k)
}

func (e *exporter) Aggregation(k metric.InstrumentKind) aggregation.Aggregation {
	return e.aggregationSelector(k)
}

func (e *exporter) Export(ctx context.Context, data metricdata.ResourceMetrics) error {
	select {
	case <-ctx.Done():
		// Don't do anything if the context has already timed out.
		return ctx.Err()
	default:
		// Context is still valid, continue.
	}
	if e.redactTimestamps {
		data = redactTimestamps(data)
	}
	return e.encVal.Load().(encoderHolder).Encode(data)
}

func (e *exporter) ForceFlush(ctx context.Context) error {
	// exporter holds no state, nothing to flush.
	return ctx.Err()
}

func (e *exporter) Shutdown(ctx context.Context) error {
	e.shutdownOnce.Do(func() {
		e.encVal.Store(encoderHolder{
			encoder: shutdownEncoder{},
		})
	})
	return ctx.Err()
}

func redactTimestamps(orig metricdata.ResourceMetrics) metricdata.ResourceMetrics {
	rm := metricdata.ResourceMetrics{
		Resource:     orig.Resource,
		ScopeMetrics: make([]metricdata.ScopeMetrics, len(orig.ScopeMetrics)),
	}
	for i, sm := range orig.ScopeMetrics {
		rm.ScopeMetrics[i] = metricdata.ScopeMetrics{
			Scope:   sm.Scope,
			Metrics: make([]metricdata.Metrics, len(sm.Metrics)),
		}
		for j, m := range sm.Metrics {
			rm.ScopeMetrics[i].Metrics[j] = metricdata.Metrics{
				Name:        m.Name,
				Description: m.Description,
				Unit:        m.Unit,
				Data:        redactAggregationTimestamps(m.Data),
			}
		}
	}
	return rm
}

func redactAggregationTimestamps(orig metricdata.Aggregation) metricdata.Aggregation {
	switch a := orig.(type) {
	case metricdata.Sum[float64]:
		return metricdata.Sum[float64]{
			Temporality: a.Temporality,
			DataPoints:  redactDataPointTimestamps(a.DataPoints),
			IsMonotonic: a.IsMonotonic,
		}
	case metricdata.Sum[int64]:
		return metricdata.Sum[int64]{
			Temporality: a.Temporality,
			DataPoints:  redactDataPointTimestamps(a.DataPoints),
			IsMonotonic: a.IsMonotonic,
		}
	case metricdata.Gauge[float64]:
		return metricdata.Gauge[float64]{
			DataPoints: redactDataPointTimestamps(a.DataPoints),
		}
	case metricdata.Gauge[int64]:
		return metricdata.Gauge[int64]{
			DataPoints: redactDataPointTimestamps(a.DataPoints),
		}
	case metricdata.Histogram:
		return metricdata.Histogram{
			Temporality: a.Temporality,
			DataPoints:  redactHistogramTimestamps(a.DataPoints),
		}
	default:
		global.Debug("unknown aggregation type", "type", fmt.Sprintf("%T", a))
		return orig
	}
}

func redactHistogramTimestamps(hdp []metricdata.HistogramDataPoint) []metricdata.HistogramDataPoint {
	out := make([]metricdata.HistogramDataPoint, len(hdp))
	for i, dp := range hdp {
		out[i] = metricdata.HistogramDataPoint{
			Attributes:   dp.Attributes,
			Count:        dp.Count,
			Sum:          dp.Sum,
			Bounds:       dp.Bounds,
			BucketCounts: dp.BucketCounts,
			Min:          dp.Min,
			Max:          dp.Max,
		}
	}
	return out
}

func redactDataPointTimestamps[T int64 | float64](sdp []metricdata.DataPoint[T]) []metricdata.DataPoint[T] {
	out := make([]metricdata.DataPoint[T], len(sdp))
	for i, dp := range sdp {
		out[i] = metricdata.DataPoint[T]{
			Attributes: dp.Attributes,
			Value:      dp.Value,
		}
	}
	return out
}
