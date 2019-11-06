// Copyright 2019, OpenTelemetry Authors
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

package propagation

import (
	"go.opentelemetry.io/otel"
)

type binaryPropagator struct{}

var _ otel.BinaryFormatPropagator = binaryPropagator{}

// BinaryPropagator creates a new propagator. The propagator implements
// ToBytes and FromBytes method to transform SpanContext to/from byte array.
func BinaryPropagator() otel.BinaryFormatPropagator {
	return binaryPropagator{}
}

// ToBytes implements ToBytes method of propagation.BinaryFormatPropagator.
// It serializes otel.SpanContext into a byte array.
func (bp binaryPropagator) ToBytes(sc otel.SpanContext) []byte {
	if sc == otel.EmptySpanContext() {
		return nil
	}
	var b [29]byte
	copy(b[2:18], sc.TraceID[:])
	b[18] = 1
	copy(b[19:27], sc.SpanID[:])
	b[27] = 2
	b[28] = sc.TraceFlags
	return b[:]
}

// FromBytes implements FromBytes method of propagation.BinaryFormatPropagator.
// It de-serializes bytes into otel.SpanContext.
func (bp binaryPropagator) FromBytes(b []byte) (sc otel.SpanContext) {
	if len(b) == 0 {
		return otel.EmptySpanContext()
	}
	b = b[1:]
	if len(b) >= 17 && b[0] == 0 {
		copy(sc.TraceID[:], b[1:17])
		b = b[17:]
	} else {
		return otel.EmptySpanContext()
	}
	if len(b) >= 9 && b[0] == 1 {
		copy(sc.SpanID[:], b[1:9])
		b = b[9:]
	}
	if len(b) >= 2 && b[0] == 2 {
		sc.TraceFlags = b[1]
	}
	if sc.IsValid() {
		return sc
	}
	return otel.EmptySpanContext()
}
