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

package metric

import (
	"context"
)

type Float64Measure struct {
	CommonMetric
}

type Int64Measure struct {
	CommonMetric
}

type Float64MeasureHandle struct {
	Handle
}

type Int64MeasureHandle struct {
	Handle
}

type MeasureOptionApplier interface {
	ApplyMeasureOption(*Descriptor)
}

type measureOptionWrapper struct {
	F Option
}

var _ MeasureOptionApplier = measureOptionWrapper{}

func (o measureOptionWrapper) ApplyMeasureOption(d *Descriptor) {
	o.F(d)
}

func newMeasure(name string, valueKind ValueKind, mos ...MeasureOptionApplier) CommonMetric {
	m := registerCommonMetric(name, MeasureKind, valueKind)
	for _, opt := range mos {
		opt.ApplyMeasureOption(m.Descriptor)
	}
	return m
}

func NewFloat64Measure(name string, mos ...MeasureOptionApplier) (c Float64Measure) {
	c.CommonMetric = newMeasure(name, Float64ValueKind, mos...)
	return
}

func NewInt64Measure(name string, mos ...MeasureOptionApplier) (c Int64Measure) {
	c.CommonMetric = newMeasure(name, Int64ValueKind, mos...)
	return
}

func (c *Float64Measure) GetHandle(labels LabelSet) (h Float64MeasureHandle) {
	h.Handle = c.getHandle(labels)
	return
}

func (c *Int64Measure) GetHandle(labels LabelSet) (h Int64MeasureHandle) {
	h.Handle = c.getHandle(labels)
	return
}

func (c *Float64Measure) Measurement(value float64) Measurement {
	return c.float64Measurement(value)
}

func (c *Int64Measure) Measurement(value int64) Measurement {
	return c.int64Measurement(value)
}

func (c *Float64Measure) Record(ctx context.Context, value float64, labels LabelSet) {
	c.recordOne(ctx, NewFloat64MeasurementValue(value), labels)
}

func (c *Int64Measure) Record(ctx context.Context, value int64, labels LabelSet) {
	c.recordOne(ctx, NewInt64MeasurementValue(value), labels)
}

func (h *Float64MeasureHandle) Record(ctx context.Context, value float64) {
	h.RecordOne(ctx, NewFloat64MeasurementValue(value))
}

func (h *Int64MeasureHandle) Record(ctx context.Context, value int64) {
	h.RecordOne(ctx, NewInt64MeasurementValue(value))
}
