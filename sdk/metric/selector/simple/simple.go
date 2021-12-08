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

package simple // import "go.opentelemetry.io/otel/sdk/metric/selector/simple"

import (
	"go.opentelemetry.io/otel/metric/sdkapi"
	export "go.opentelemetry.io/otel/sdk/export/metric"
	"go.opentelemetry.io/otel/sdk/metric/aggregator/exponential"
	"go.opentelemetry.io/otel/sdk/metric/aggregator/histogram"
	"go.opentelemetry.io/otel/sdk/metric/aggregator/lastvalue"
	"go.opentelemetry.io/otel/sdk/metric/aggregator/sum"
)

type (
	selectorInexpensive struct{}
	selectorHistogram   struct {
		options []histogram.Option
	}
	selectorExponentialHistogram struct {
		options []exponential.Option
	}
)

var (
	_ export.AggregatorSelector = selectorInexpensive{}
	_ export.AggregatorSelector = selectorHistogram{}
	_ export.AggregatorSelector = selectorExponentialHistogram{}
)

// NewWithInexpensiveDistribution returns a simple aggregator selector
// that uses minmaxsumcount aggregators for `Histogram`
// instruments.  This selector is faster and uses less memory than the
// others in this package because minmaxsumcount aggregators maintain
// the least information about the distribution among these choices.
func NewWithInexpensiveDistribution() export.AggregatorSelector {
	return selectorInexpensive{}
}

// NewWithHistogramDistribution returns a simple aggregator selector
// that uses histogram aggregators for `Histogram` instruments.
// This selector is a good default choice for most metric exporters.
func NewWithHistogramDistribution(options ...histogram.Option) export.AggregatorSelector {
	return selectorHistogram{options: options}
}

func NewWithExponentialHistogramDistribution(options ...exponential.Option) export.AggregatorSelector {
	return selectorExponentialHistogram{options: options}
}

func sumAggs(aggPtrs []*export.Aggregator) {
	aggs := sum.New(len(aggPtrs))
	for i := range aggPtrs {
		*aggPtrs[i] = &aggs[i]
	}
}

func lastValueAggs(aggPtrs []*export.Aggregator) {
	aggs := lastvalue.New(len(aggPtrs))
	for i := range aggPtrs {
		*aggPtrs[i] = &aggs[i]
	}
}

func (selectorInexpensive) AggregatorFor(descriptor *sdkapi.Descriptor, aggPtrs ...*export.Aggregator) {
	switch descriptor.InstrumentKind() {
	case sdkapi.GaugeObserverInstrumentKind:
		lastValueAggs(aggPtrs)
	case sdkapi.HistogramInstrumentKind:
		aggs := sum.New(len(aggPtrs))
		for i := range aggPtrs {
			*aggPtrs[i] = &aggs[i]
		}
	default:
		sumAggs(aggPtrs)
	}
}

func (s selectorHistogram) AggregatorFor(descriptor *sdkapi.Descriptor, aggPtrs ...*export.Aggregator) {
	switch descriptor.InstrumentKind() {
	case sdkapi.GaugeObserverInstrumentKind:
		lastValueAggs(aggPtrs)
	case sdkapi.HistogramInstrumentKind:
		aggs := histogram.New(len(aggPtrs), descriptor, s.options...)
		for i := range aggPtrs {
			*aggPtrs[i] = &aggs[i]
		}
	default:
		sumAggs(aggPtrs)
	}
}

func (s selectorExponentialHistogram) AggregatorFor(descriptor *sdkapi.Descriptor, aggPtrs ...*export.Aggregator) {
	switch descriptor.InstrumentKind() {
	case sdkapi.GaugeObserverInstrumentKind:
		lastValueAggs(aggPtrs)
	case sdkapi.HistogramInstrumentKind:
		aggs := exponential.New(len(aggPtrs), descriptor, s.options...)
		for i := range aggPtrs {
			*aggPtrs[i] = &aggs[i]
		}
	default:
		sumAggs(aggPtrs)
	}
}
