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

package testtrace

import (
	"encoding/binary"

	"go.opentelemetry.io/api/core"
)

type Generator interface {
	TraceID() core.TraceID
	SpanID() uint64
}

var _ Generator = (*CountGenerator)(nil)

type CountGenerator struct {
	traceIDHigh uint64
	traceIDLow  uint64
	spanID      uint64
}

func NewCountGenerator() *CountGenerator {
	return &CountGenerator{}
}

func (g *CountGenerator) TraceID() core.TraceID {
	if g.traceIDHigh == g.traceIDLow {
		g.traceIDHigh++
	} else {
		g.traceIDLow++
	}

	var traceID core.TraceID

	binary.PutUvarint(traceID[0:8], g.traceIDLow)
	binary.PutUvarint(traceID[8:], g.traceIDHigh)

	return traceID
}

func (g *CountGenerator) SpanID() uint64 {
	g.spanID++

	return g.spanID
}
