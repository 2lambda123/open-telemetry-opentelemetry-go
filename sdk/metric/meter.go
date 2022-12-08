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

	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/instrument"
	"go.opentelemetry.io/otel/metric/instrument/asyncfloat64"
	"go.opentelemetry.io/otel/metric/instrument/asyncint64"
	"go.opentelemetry.io/otel/metric/instrument/syncfloat64"
	"go.opentelemetry.io/otel/metric/instrument/syncint64"
	"go.opentelemetry.io/otel/sdk/instrumentation"
)

// meter handles the creation and coordination of all metric instruments. A
// meter represents a single instrumentation scope; all metric telemetry
// produced by an instrumentation scope will use metric instruments from a
// single meter.
type meter struct {
	pipes pipelines

	instProviderInt64   *instProvider[int64]
	instProviderFloat64 *instProvider[float64]
}

func newMeter(s instrumentation.Scope, p pipelines) *meter {
	// viewCache ensures instrument conflicts, including number conflicts, this
	// meter is asked to create are logged to the user.
	var viewCache cache[string, instrumentID]

	// Passing nil as the ac parameter to newInstrumentCache will have each
	// create its own aggregator cache.
	ic := newInstrumentCache[int64](nil, &viewCache)
	fc := newInstrumentCache[float64](nil, &viewCache)

	return &meter{
		pipes:               p,
		instProviderInt64:   newInstProvider(s, p, ic),
		instProviderFloat64: newInstProvider(s, p, fc),
	}
}

// Compile-time check meter implements metric.Meter.
var _ metric.Meter = (*meter)(nil)

// Int64Counter returns a new instrument identified by name and configured with
// options. The instrument is used to synchronously record increasing int64
// measurements during a computational operation.
func (m *meter) Int64Counter(name string, options ...instrument.Option) (syncint64.Counter, error) {
	return m.instProviderInt64.lookup(InstrumentKindSyncCounter, name, options)
}

// Int64UpDownCounter returns a new instrument identified by name and
// configured with options. The instrument is used to synchronously record
// int64 measurements during a computational operation.
func (m *meter) Int64UpDownCounter(name string, options ...instrument.Option) (syncint64.UpDownCounter, error) {
	return m.instProviderInt64.lookup(InstrumentKindSyncUpDownCounter, name, options)
}

// Int64Histogram returns a new instrument identified by name and configured
// with options. The instrument is used to synchronously record the
// distribution of int64 measurements during a computational operation.
func (m *meter) Int64Histogram(name string, options ...instrument.Option) (syncint64.Histogram, error) {
	return m.instProviderInt64.lookup(InstrumentKindSyncHistogram, name, options)
}

// Int64ObservableCounter returns a new instrument identified by name and
// configured with options. The instrument is used to asynchronously record
// increasing int64 measurements once per a measurement collection cycle.
func (m *meter) Int64ObservableCounter(name string, options ...instrument.Option) (asyncint64.Counter, error) {
	return m.instProviderInt64.lookup(InstrumentKindAsyncCounter, name, options)
}

// Int64ObservableUpDownCounter returns a new instrument identified by name and
// configured with options. The instrument is used to asynchronously record
// int64 measurements once per a measurement collection cycle.
func (m *meter) Int64ObservableUpDownCounter(name string, options ...instrument.Option) (asyncint64.UpDownCounter, error) {
	return m.instProviderInt64.lookup(InstrumentKindAsyncUpDownCounter, name, options)
}

// Int64ObservableGauge returns a new instrument identified by name and
// configured with options. The instrument is used to asynchronously record
// instantaneous int64 measurements once per a measurement collection cycle.
func (m *meter) Int64ObservableGauge(name string, options ...instrument.Option) (asyncint64.Gauge, error) {
	return m.instProviderInt64.lookup(InstrumentKindAsyncGauge, name, options)
}

// Float64Counter returns a new instrument identified by name and configured
// with options. The instrument is used to synchronously record increasing
// float64 measurements during a computational operation.
func (m *meter) Float64Counter(name string, options ...instrument.Option) (syncfloat64.Counter, error) {
	return m.instProviderFloat64.lookup(InstrumentKindSyncCounter, name, options)
}

// Float64UpDownCounter returns a new instrument identified by name and
// configured with options. The instrument is used to synchronously record
// float64 measurements during a computational operation.
func (m *meter) Float64UpDownCounter(name string, options ...instrument.Option) (syncfloat64.UpDownCounter, error) {
	return m.instProviderFloat64.lookup(InstrumentKindSyncUpDownCounter, name, options)
}

// Float64Histogram returns a new instrument identified by name and configured
// with options. The instrument is used to synchronously record the
// distribution of float64 measurements during a computational operation.
func (m *meter) Float64Histogram(name string, options ...instrument.Option) (syncfloat64.Histogram, error) {
	return m.instProviderFloat64.lookup(InstrumentKindSyncHistogram, name, options)
}

// Float64ObservableCounter returns a new instrument identified by name and
// configured with options. The instrument is used to asynchronously record
// increasing float64 measurements once per a measurement collection cycle.
func (m *meter) Float64ObservableCounter(name string, options ...instrument.Option) (asyncfloat64.Counter, error) {
	return m.instProviderFloat64.lookup(InstrumentKindAsyncCounter, name, options)
}

// Float64ObservableUpDownCounter returns a new instrument identified by name
// and configured with options. The instrument is used to asynchronously record
// float64 measurements once per a measurement collection cycle.
func (m *meter) Float64ObservableUpDownCounter(name string, options ...instrument.Option) (asyncfloat64.UpDownCounter, error) {
	return m.instProviderFloat64.lookup(InstrumentKindAsyncUpDownCounter, name, options)
}

// Float64ObservableGauge returns a new instrument identified by name and
// configured with options. The instrument is used to asynchronously record
// instantaneous float64 measurements once per a measurement collection cycle.
func (m *meter) Float64ObservableGauge(name string, options ...instrument.Option) (asyncfloat64.Gauge, error) {
	return m.instProviderFloat64.lookup(InstrumentKindAsyncGauge, name, options)
}

// RegisterCallback registers the function f to be called when any of the
// insts Collect method is called.
func (m *meter) RegisterCallback(insts []instrument.Asynchronous, f func(context.Context)) error {
	for _, inst := range insts {
		// Only register if at least one instrument has a non-drop aggregation.
		// Otherwise, calling f during collection will be wasted computation.
		switch t := inst.(type) {
		case *instrumentImpl[int64]:
			if len(t.aggregators) > 0 {
				return m.registerCallback(f)
			}
		case *instrumentImpl[float64]:
			if len(t.aggregators) > 0 {
				return m.registerCallback(f)
			}
		default:
			// Instrument external to the SDK. For example, an instrument from
			// the "go.opentelemetry.io/otel/metric/internal/global" package.
			//
			// Fail gracefully here, assume a valid instrument.
			return m.registerCallback(f)
		}
	}
	// All insts use drop aggregation.
	return nil
}

func (m *meter) registerCallback(f func(context.Context)) error {
	m.pipes.registerCallback(f)
	return nil
}

// instProvider provides all OpenTelemetry instruments.
type instProvider[N int64 | float64] struct {
	scope   instrumentation.Scope
	resolve resolver[N]
}

func newInstProvider[N int64 | float64](s instrumentation.Scope, p pipelines, c instrumentCache[N]) *instProvider[N] {
	return &instProvider[N]{scope: s, resolve: newResolver(p, c)}
}

// lookup returns the resolved instrumentImpl.
func (p *instProvider[N]) lookup(kind InstrumentKind, name string, opts []instrument.Option) (*instrumentImpl[N], error) {
	cfg := instrument.NewConfig(opts...)
	i := Instrument{
		Name:        name,
		Description: cfg.Description(),
		Unit:        cfg.Unit(),
		Kind:        kind,
		Scope:       p.scope,
	}
	aggs, err := p.resolve.Aggregators(i)
	return &instrumentImpl[N]{aggregators: aggs}, err
}
