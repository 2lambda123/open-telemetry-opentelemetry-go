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
	"errors"
	"fmt"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/internal/global"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/instrument"
	"go.opentelemetry.io/otel/metric/unit"
	"go.opentelemetry.io/otel/sdk/instrumentation"
	"go.opentelemetry.io/otel/sdk/metric/internal"
)

// meter handles the creation and coordination of all metric instruments. A
// meter represents a single instrumentation scope; all metric telemetry
// produced by an instrumentation scope will use metric instruments from a
// single meter.
type meter struct {
	scope instrumentation.Scope
	pipes pipelines

	int64IP   *instProvider[int64]
	float64IP *instProvider[float64]
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
		scope:     s,
		pipes:     p,
		int64IP:   newInstProvider(s, p, ic),
		float64IP: newInstProvider(s, p, fc),
	}
}

// Compile-time check meter implements metric.Meter.
var _ metric.Meter = (*meter)(nil)

// Int64Counter returns a new instrument identified by name and configured with
// options. The instrument is used to synchronously record increasing int64
// measurements during a computational operation.
func (m *meter) Int64Counter(name string, options ...instrument.Int64Option) (instrument.Int64Counter, error) {
	cfg := instrument.NewInt64Config(options...)
	const kind = InstrumentKindCounter
	return m.int64IP.lookup(kind, name, cfg.Description(), cfg.Unit())
}

// Int64UpDownCounter returns a new instrument identified by name and
// configured with options. The instrument is used to synchronously record
// int64 measurements during a computational operation.
func (m *meter) Int64UpDownCounter(name string, options ...instrument.Int64Option) (instrument.Int64UpDownCounter, error) {
	cfg := instrument.NewInt64Config(options...)
	const kind = InstrumentKindUpDownCounter
	return m.int64IP.lookup(kind, name, cfg.Description(), cfg.Unit())
}

// Int64Histogram returns a new instrument identified by name and configured
// with options. The instrument is used to synchronously record the
// distribution of int64 measurements during a computational operation.
func (m *meter) Int64Histogram(name string, options ...instrument.Int64Option) (instrument.Int64Histogram, error) {
	cfg := instrument.NewInt64Config(options...)
	const kind = InstrumentKindHistogram
	return m.int64IP.lookup(kind, name, cfg.Description(), cfg.Unit())
}

// Int64ObservableCounter returns a new instrument identified by name and
// configured with options. The instrument is used to asynchronously record
// increasing int64 measurements once per a measurement collection cycle.
func (m *meter) Int64ObservableCounter(name string, options ...instrument.Int64ObserverOption) (instrument.Int64ObservableCounter, error) {
	cfg := instrument.NewInt64ObserverConfig(options...)
	const kind = InstrumentKindObservableCounter
	p := int64ObservProvider{m.int64IP}
	inst, err := p.lookup(kind, name, cfg.Description(), cfg.Unit())
	if err != nil {
		return nil, err
	}
	p.registerCallbacks(inst, cfg.Callbacks())
	return inst, nil
}

// Int64ObservableUpDownCounter returns a new instrument identified by name and
// configured with options. The instrument is used to asynchronously record
// int64 measurements once per a measurement collection cycle.
func (m *meter) Int64ObservableUpDownCounter(name string, options ...instrument.Int64ObserverOption) (instrument.Int64ObservableUpDownCounter, error) {
	cfg := instrument.NewInt64ObserverConfig(options...)
	const kind = InstrumentKindObservableUpDownCounter
	p := int64ObservProvider{m.int64IP}
	inst, err := p.lookup(kind, name, cfg.Description(), cfg.Unit())
	if err != nil {
		return nil, err
	}
	p.registerCallbacks(inst, cfg.Callbacks())
	return inst, nil
}

// Int64ObservableGauge returns a new instrument identified by name and
// configured with options. The instrument is used to asynchronously record
// instantaneous int64 measurements once per a measurement collection cycle.
func (m *meter) Int64ObservableGauge(name string, options ...instrument.Int64ObserverOption) (instrument.Int64ObservableGauge, error) {
	cfg := instrument.NewInt64ObserverConfig(options...)
	const kind = InstrumentKindObservableGauge
	p := int64ObservProvider{m.int64IP}
	inst, err := p.lookup(kind, name, cfg.Description(), cfg.Unit())
	if err != nil {
		return nil, err
	}
	p.registerCallbacks(inst, cfg.Callbacks())
	return inst, nil
}

// Float64Counter returns a new instrument identified by name and configured
// with options. The instrument is used to synchronously record increasing
// float64 measurements during a computational operation.
func (m *meter) Float64Counter(name string, options ...instrument.Float64Option) (instrument.Float64Counter, error) {
	cfg := instrument.NewFloat64Config(options...)
	const kind = InstrumentKindCounter
	return m.float64IP.lookup(kind, name, cfg.Description(), cfg.Unit())
}

// Float64UpDownCounter returns a new instrument identified by name and
// configured with options. The instrument is used to synchronously record
// float64 measurements during a computational operation.
func (m *meter) Float64UpDownCounter(name string, options ...instrument.Float64Option) (instrument.Float64UpDownCounter, error) {
	cfg := instrument.NewFloat64Config(options...)
	const kind = InstrumentKindUpDownCounter
	return m.float64IP.lookup(kind, name, cfg.Description(), cfg.Unit())
}

// Float64Histogram returns a new instrument identified by name and configured
// with options. The instrument is used to synchronously record the
// distribution of float64 measurements during a computational operation.
func (m *meter) Float64Histogram(name string, options ...instrument.Float64Option) (instrument.Float64Histogram, error) {
	cfg := instrument.NewFloat64Config(options...)
	const kind = InstrumentKindHistogram
	return m.float64IP.lookup(kind, name, cfg.Description(), cfg.Unit())
}

// Float64ObservableCounter returns a new instrument identified by name and
// configured with options. The instrument is used to asynchronously record
// increasing float64 measurements once per a measurement collection cycle.
func (m *meter) Float64ObservableCounter(name string, options ...instrument.Float64ObserverOption) (instrument.Float64ObservableCounter, error) {
	cfg := instrument.NewFloat64ObserverConfig(options...)
	const kind = InstrumentKindObservableCounter
	p := float64ObservProvider{m.float64IP}
	inst, err := p.lookup(kind, name, cfg.Description(), cfg.Unit())
	if err != nil {
		return nil, err
	}
	p.registerCallbacks(inst, cfg.Callbacks())
	return inst, nil
}

// Float64ObservableUpDownCounter returns a new instrument identified by name
// and configured with options. The instrument is used to asynchronously record
// float64 measurements once per a measurement collection cycle.
func (m *meter) Float64ObservableUpDownCounter(name string, options ...instrument.Float64ObserverOption) (instrument.Float64ObservableUpDownCounter, error) {
	cfg := instrument.NewFloat64ObserverConfig(options...)
	const kind = InstrumentKindObservableUpDownCounter
	p := float64ObservProvider{m.float64IP}
	inst, err := p.lookup(kind, name, cfg.Description(), cfg.Unit())
	if err != nil {
		return nil, err
	}
	p.registerCallbacks(inst, cfg.Callbacks())
	return inst, nil
}

// Float64ObservableGauge returns a new instrument identified by name and
// configured with options. The instrument is used to asynchronously record
// instantaneous float64 measurements once per a measurement collection cycle.
func (m *meter) Float64ObservableGauge(name string, options ...instrument.Float64ObserverOption) (instrument.Float64ObservableGauge, error) {
	cfg := instrument.NewFloat64ObserverConfig(options...)
	const kind = InstrumentKindObservableGauge
	p := float64ObservProvider{m.float64IP}
	inst, err := p.lookup(kind, name, cfg.Description(), cfg.Unit())
	if err != nil {
		return nil, err
	}
	p.registerCallbacks(inst, cfg.Callbacks())
	return inst, nil
}

// RegisterCallback registers the function f to be called when any of the
// insts Collect method is called.
func (m *meter) RegisterCallback(insts []instrument.Asynchronous, f metric.Callback) (metric.Registration, error) {
	if len(insts) == 0 {
		// Don't allocate a multiObserver if not needed.
		return noopRegister{}, nil
	}

	reg := newMultiObserver()
	var errs multierror
	for _, inst := range insts {
		switch o := inst.(type) {
		case *observer[int64]:
			if err := o.registerable(m.scope); err != nil {
				if !errors.Is(err, errEmptyAgg) {
					errs.append(err)
				}
				continue
			}
			reg.registerInt64(o.observerID)
		case *observer[float64]:
			if err := o.registerable(m.scope); err != nil {
				if !errors.Is(err, errEmptyAgg) {
					errs.append(err)
				}
				continue
			}
			reg.registerFloat64(o.observerID)
		default:
			// Instrument external to the SDK.
			return nil, fmt.Errorf("invalid observer: from different implementation")
		}
	}

	if err := errs.errorOrNil(); err != nil {
		return nil, err
	}

	if reg.len() == 0 {
		// All insts use drop aggregation.
		return noopRegister{}, nil
	}

	cback := func(ctx context.Context) error {
		return f(ctx, reg)
	}
	return m.pipes.registerMultiCallback(cback), nil
}

type multiObserver struct {
	float64 map[observerID[float64]]struct{}
	int64   map[observerID[int64]]struct{}
}

func newMultiObserver() multiObserver {
	return multiObserver{
		float64: make(map[observerID[float64]]struct{}),
		int64:   make(map[observerID[int64]]struct{}),
	}
}

func (r multiObserver) len() int {
	return len(r.float64) + len(r.int64)
}

func (r multiObserver) registerFloat64(id observerID[float64]) {
	r.float64[id] = struct{}{}
}

func (r multiObserver) registerInt64(id observerID[int64]) {
	r.int64[id] = struct{}{}
}

var (
	errUnknownObserver = errors.New("unknown observer")
	errUnregObserver   = errors.New("observer not registered for callback")
)

func (r multiObserver) ObserveFloat64(o instrument.Float64Observer, v float64, a ...attribute.KeyValue) {
	oImpl, ok := o.(*observer[float64])
	if !ok {
		global.Error(errUnknownObserver, "failed to record")
		return
	}
	if _, registered := r.float64[oImpl.observerID]; !registered {
		global.Error(errUnregObserver, "failed to record",
			"name", oImpl.name,
			"description", oImpl.description,
			"unit", oImpl.unit,
			"number", fmt.Sprintf("%T", float64(0)),
		)
		return
	}
	oImpl.observe(v, a)
}

func (r multiObserver) ObserveInt64(o instrument.Int64Observer, v int64, a ...attribute.KeyValue) {
	oImpl, ok := o.(*observer[int64])
	if !ok {
		global.Error(errUnknownObserver, "failed to record")
		return
	}
	if _, registered := r.int64[oImpl.observerID]; !registered {
		global.Error(errUnregObserver, "failed to record",
			"name", oImpl.name,
			"description", oImpl.description,
			"unit", oImpl.unit,
			"number", fmt.Sprintf("%T", int64(0)),
		)
		return
	}
	oImpl.observe(v, a)
}

type noopRegister struct{}

func (noopRegister) Unregister() error {
	return nil
}

// instProvider provides all OpenTelemetry instruments.
type instProvider[N int64 | float64] struct {
	scope   instrumentation.Scope
	pipes   pipelines
	resolve resolver[N]
}

func newInstProvider[N int64 | float64](s instrumentation.Scope, p pipelines, c instrumentCache[N]) *instProvider[N] {
	return &instProvider[N]{scope: s, pipes: p, resolve: newResolver(p, c)}
}

func (p *instProvider[N]) aggs(kind InstrumentKind, name, desc string, u unit.Unit) ([]internal.Aggregator[N], error) {
	inst := Instrument{
		Name:        name,
		Description: desc,
		Unit:        u,
		Kind:        kind,
		Scope:       p.scope,
	}
	return p.resolve.Aggregators(inst)
}

// lookup returns the resolved instrumentImpl.
func (p *instProvider[N]) lookup(kind InstrumentKind, name, desc string, u unit.Unit) (*instrumentImpl[N], error) {
	aggs, err := p.aggs(kind, name, desc, u)
	return &instrumentImpl[N]{aggregators: aggs}, err
}

type int64ObservProvider struct{ *instProvider[int64] }

func (p int64ObservProvider) lookup(kind InstrumentKind, name, desc string, u unit.Unit) (*observer[int64], error) {
	aggs, err := p.aggs(kind, name, desc, u)
	return newObserver(p.scope, kind, name, desc, u, aggs), err
}

func (p int64ObservProvider) registerCallbacks(inst *observer[int64], cBacks []instrument.Int64Callback) {
	if inst == nil {
		// Drop aggregator.
		return
	}

	for _, cBack := range cBacks {
		p.pipes.registerCallback(p.callback(inst, cBack))
	}
}

func (p int64ObservProvider) callback(i *observer[int64], f instrument.Int64Callback) func(context.Context) error {
	inst := callbackObserver[int64]{i}
	return func(ctx context.Context) error { return f(ctx, inst) }
}

type float64ObservProvider struct{ *instProvider[float64] }

func (p float64ObservProvider) lookup(kind InstrumentKind, name, desc string, u unit.Unit) (*observer[float64], error) {
	aggs, err := p.aggs(kind, name, desc, u)
	return newObserver(p.scope, kind, name, desc, u, aggs), err
}

func (p float64ObservProvider) registerCallbacks(inst *observer[float64], cBacks []instrument.Float64Callback) {
	if inst == nil {
		// Drop aggregator.
		return
	}

	for _, cBack := range cBacks {
		p.pipes.registerCallback(p.callback(inst, cBack))
	}
}

func (p float64ObservProvider) callback(i *observer[float64], f instrument.Float64Callback) func(context.Context) error {
	inst := callbackObserver[float64]{i}
	return func(ctx context.Context) error { return f(ctx, inst) }
}

// callbackObserver is passed to a callback where a users is expected to call
// Observe directly to record a measurement.
type callbackObserver[N int64 | float64] struct {
	*observer[N]
}

func (o callbackObserver[N]) Observe(_ context.Context, val N, attrs ...attribute.KeyValue) {
	o.observe(val, attrs)
}
