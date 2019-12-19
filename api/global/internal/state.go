package internal

import (
	"sync"
	"sync/atomic"

	"go.opentelemetry.io/otel/api/metric"
	"go.opentelemetry.io/otel/api/trace"
)

type (
	traceProviderHolder struct {
		tp trace.Provider
	}

	meterProviderHolder struct {
		mp metric.Provider
	}
)

var (
	globalTracer = defaultTracerValue()
	globalMeter  = defaultMeterValue()

	delegateMeterOnce sync.Once
)

// TraceProvider is the internal implementation for global.TraceProvider.
func TraceProvider() trace.Provider {
	return globalTracer.Load().(traceProviderHolder).tp
}

// SetTraceProvider is the internal implementation for global.SetTraceProvider.
func SetTraceProvider(tp trace.Provider) {
	globalTracer.Store(traceProviderHolder{tp: tp})
}

// MeterProvider is the internal implementation for global.MeterProvider.
func MeterProvider() metric.Provider {
	return globalMeter.Load().(meterProviderHolder).mp
}

// SetMeterProvider is the internal implementation for global.SetMeterProvider.
func SetMeterProvider(mp metric.Provider) {
	delegateMeterOnce.Do(func() {
		current := MeterProvider()

		if current == mp {
			// Setting the provider to the prior default
			// is nonsense, set it to a noop.
			mp = metric.NoopProvider{}
		} else if def, ok := current.(*meterProvider); ok {
			def.setDelegate(mp)
		}
	})
	globalMeter.Store(meterProviderHolder{mp: mp})
}

func defaultTracerValue() *atomic.Value {
	v := &atomic.Value{}
	v.Store(traceProviderHolder{tp: trace.NoopProvider{}})
	return v
}

func defaultMeterValue() *atomic.Value {
	v := &atomic.Value{}
	v.Store(meterProviderHolder{mp: &meterProvider{}})
	return v
}

// ResetForTest restores the initial global state, for testing purposes.
func ResetForTest() {
	globalTracer = defaultTracerValue()
	globalMeter = defaultMeterValue()
	delegateMeterOnce = sync.Once{}
}
