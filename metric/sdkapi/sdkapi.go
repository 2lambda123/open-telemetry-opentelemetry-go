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

package sdkapi // import "go.opentelemetry.io/otel/metric/sdkapi"

import (
	"context"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric/number"
)

// MeterImpl is the interface an SDK must implement to supply a Meter
// implementation.
type MeterImpl interface {
	// RecordBatch atomically records a batch of measurements.
	RecordBatch(ctx context.Context, labels []attribute.KeyValue, measurement ...Measurement)

	// NewInstrument returns a newly constructed instrument
	// implementation or an error, should one occur.
	NewInstrument(descriptor Descriptor) (Instrument, error)

	NewCallback(insts []Instrument, callback func(context.Context)) (Callback, error)
}

type Callback interface {
	Instruments() []Instrument
}

// Instrument is a common interface for synchronous and
// asynchronous instruments.
type Instrument interface {
	// Implementation returns the underlying implementation of the
	// instrument, which allows the implementation to gain access
	// to its own representation especially from a `Measurement`.
	Implementation() interface{}

	// Descriptor returns a copy of the instrument's Descriptor.
	Descriptor() Descriptor

	// RecordOne captures a single metric event.
	RecordOne(ctx context.Context, number number.Number, labels []attribute.KeyValue)
}

// NewMeasurement constructs a single observation, a binding between
// an asynchronous instrument and a number.
func NewMeasurement(instrument Instrument, number number.Number) Measurement {
	return Measurement{
		instrument: instrument,
		number:     number,
	}
}

// Measurement is a low-level type used with synchronous instruments
// as a direct interface to the SDK via `RecordBatch`.
type Measurement struct {
	// number needs to be aligned for 64-bit atomic operations.
	number     number.Number
	instrument Instrument
}

// SyncImpl returns the instrument that created this measurement.
// This returns an implementation-level object for use by the SDK,
// users should not refer to this.
func (m Measurement) Impl() Instrument {
	return m.instrument
}

// Number returns a number recorded in this measurement.
func (m Measurement) Number() number.Number {
	return m.number
}
