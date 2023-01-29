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

package metric // import "go.opentelemetry.io/otel/sdk/metric"

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"

	"go.opentelemetry.io/otel/internal/global"
	"go.opentelemetry.io/otel/sdk/metric/aggregation"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
)

// manualReader is a simple Reader that allows an application to
// read metrics on demand.
type manualReader struct {
	sdkProducer  atomic.Value
	shutdownOnce sync.Once

	mu                sync.Mutex
	isShutdown        bool
	externalProducers atomic.Value

	temporalitySelector TemporalitySelector
	aggregationSelector AggregationSelector
}

// Compile time check the manualReader implements Reader and is comparable.
var _ = map[Reader]struct{}{&manualReader{}: {}}

// NewManualReader returns a Reader which is directly called to collect metrics.
func NewManualReader(opts ...ManualReaderOption) Reader {
	cfg := newManualReaderConfig(opts)
	r := &manualReader{
		temporalitySelector: cfg.temporalitySelector,
		aggregationSelector: cfg.aggregationSelector,
	}
	r.externalProducers.Store([]Producer{})
	return r
}

// register stores the sdkProducer which enables the caller
// to read metrics from the SDK on demand.
func (mr *manualReader) register(p sdkProducer) {
	// Only register once. If producer is already set, do nothing.
	if !mr.sdkProducer.CompareAndSwap(nil, produceHolder{produce: p.produce}) {
		msg := "did not register manual reader"
		global.Error(errDuplicateRegister, msg)
	}
}

// RegisterProducer stores the external Producer which enables the caller
// to read metrics on demand.
func (mr *manualReader) RegisterProducer(p Producer) {
	mr.mu.Lock()
	defer mr.mu.Unlock()
	if mr.isShutdown {
		return
	}
	currentProducers := mr.externalProducers.Load().([]Producer)
	newProducers := []Producer{}
	newProducers = append(newProducers, currentProducers...)
	newProducers = append(newProducers, p)
	mr.externalProducers.Store(newProducers)
}

// temporality reports the Temporality for the instrument kind provided.
func (mr *manualReader) temporality(kind InstrumentKind) metricdata.Temporality {
	return mr.temporalitySelector(kind)
}

// aggregation returns what Aggregation to use for kind.
func (mr *manualReader) aggregation(kind InstrumentKind) aggregation.Aggregation { // nolint:revive  // import-shadow for method scoped by type.
	return mr.aggregationSelector(kind)
}

// ForceFlush is a no-op, it always returns nil.
func (mr *manualReader) ForceFlush(context.Context) error {
	return nil
}

// Shutdown closes any connections and frees any resources used by the reader.
func (mr *manualReader) Shutdown(context.Context) error {
	err := ErrReaderShutdown
	mr.shutdownOnce.Do(func() {
		// Any future call to Collect will now return ErrReaderShutdown.
		mr.sdkProducer.Store(produceHolder{
			produce: shutdownProducer{}.produce,
		})
		mr.mu.Lock()
		defer mr.mu.Unlock()
		mr.isShutdown = true
		// release references to Producer(s)
		mr.externalProducers.Store([]Producer{})
		err = nil
	})
	return err
}

// Collect gathers all metrics from the SDK and other Producers, calling any
// callbacks necessary. Collect will return an error if called after shutdown.
func (mr *manualReader) Collect(ctx context.Context) (metricdata.ResourceMetrics, error) {
	p := mr.sdkProducer.Load()
	if p == nil {
		return metricdata.ResourceMetrics{}, ErrReaderNotRegistered
	}

	ph, ok := p.(produceHolder)
	if !ok {
		// The atomic.Value is entirely in the periodicReader's control so
		// this should never happen. In the unforeseen case that this does
		// happen, return an error instead of panicking so a users code does
		// not halt in the processes.
		err := fmt.Errorf("manual reader: invalid producer: %T", p)
		return metricdata.ResourceMetrics{}, err
	}

	rm, err := ph.produce(ctx)
	if err != nil {
		return metricdata.ResourceMetrics{}, err
	}
	var errs []error
	for _, producer := range mr.externalProducers.Load().([]Producer) {
		externalMetrics, err := producer.Produce(ctx)
		if err != nil {
			errs = append(errs, err)
		}
		rm.ScopeMetrics = append(rm.ScopeMetrics, externalMetrics...)
	}
	return rm, unifyErrors(errs)
}

// manualReaderConfig contains configuration options for a ManualReader.
type manualReaderConfig struct {
	temporalitySelector TemporalitySelector
	aggregationSelector AggregationSelector
}

// newManualReaderConfig returns a manualReaderConfig configured with options.
func newManualReaderConfig(opts []ManualReaderOption) manualReaderConfig {
	cfg := manualReaderConfig{
		temporalitySelector: DefaultTemporalitySelector,
		aggregationSelector: DefaultAggregationSelector,
	}
	for _, opt := range opts {
		cfg = opt.applyManual(cfg)
	}
	return cfg
}

// ManualReaderOption applies a configuration option value to a ManualReader.
type ManualReaderOption interface {
	applyManual(manualReaderConfig) manualReaderConfig
}

// WithTemporalitySelector sets the TemporalitySelector a reader will use to
// determine the Temporality of an instrument based on its kind. If this
// option is not used, the reader will use the DefaultTemporalitySelector.
func WithTemporalitySelector(selector TemporalitySelector) ManualReaderOption {
	return temporalitySelectorOption{selector: selector}
}

type temporalitySelectorOption struct {
	selector func(instrument InstrumentKind) metricdata.Temporality
}

// applyManual returns a manualReaderConfig with option applied.
func (t temporalitySelectorOption) applyManual(mrc manualReaderConfig) manualReaderConfig {
	mrc.temporalitySelector = t.selector
	return mrc
}

// WithAggregationSelector sets the AggregationSelector a reader will use to
// determine the aggregation to use for an instrument based on its kind. If
// this option is not used, the reader will use the DefaultAggregationSelector
// or the aggregation explicitly passed for a view matching an instrument.
func WithAggregationSelector(selector AggregationSelector) ManualReaderOption {
	// Deep copy and validate before using.
	wrapped := func(ik InstrumentKind) aggregation.Aggregation {
		a := selector(ik)
		cpA := a.Copy()
		if err := cpA.Err(); err != nil {
			cpA = DefaultAggregationSelector(ik)
			global.Error(
				err, "using default aggregation instead",
				"aggregation", a,
				"replacement", cpA,
			)
		}
		return cpA
	}

	return aggregationSelectorOption{selector: wrapped}
}

type aggregationSelectorOption struct {
	selector AggregationSelector
}

// applyManual returns a manualReaderConfig with option applied.
func (t aggregationSelectorOption) applyManual(c manualReaderConfig) manualReaderConfig {
	c.aggregationSelector = t.selector
	return c
}
