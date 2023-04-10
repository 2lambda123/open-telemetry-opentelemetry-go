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

package instrument // import "go.opentelemetry.io/otel/metric/instrument"

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"go.opentelemetry.io/otel/attribute"
)

type attrConf interface {
	Attributes() attribute.Set
}

func TestWithAttributeSet(t *testing.T) {
	t.Run("Float64AddConfig", testConfAttr(func(mo ...MeasurementOption) attrConf {
		opts := make([]Float64AddOption, len(mo))
		for i := range mo {
			opts[i] = mo[i].(Float64AddOption)
		}
		return NewFloat64AddConfig(opts...)
	}))

	t.Run("Int64AddConfig", testConfAttr(func(mo ...MeasurementOption) attrConf {
		opts := make([]Int64AddOption, len(mo))
		for i := range mo {
			opts[i] = mo[i].(Int64AddOption)
		}
		return NewInt64AddConfig(opts...)
	}))

	t.Run("Float64RecordConfig", testConfAttr(func(mo ...MeasurementOption) attrConf {
		opts := make([]Float64RecordOption, len(mo))
		for i := range mo {
			opts[i] = mo[i].(Float64RecordOption)
		}
		return NewFloat64RecordConfig(opts...)
	}))

	t.Run("Int64RecordConfig", testConfAttr(func(mo ...MeasurementOption) attrConf {
		opts := make([]Int64RecordOption, len(mo))
		for i := range mo {
			opts[i] = mo[i].(Int64RecordOption)
		}
		return NewInt64RecordConfig(opts...)
	}))

	t.Run("Float64ObserveConfig", testConfAttr(func(mo ...MeasurementOption) attrConf {
		opts := make([]Float64ObserveOption, len(mo))
		for i := range mo {
			opts[i] = mo[i].(Float64ObserveOption)
		}
		return NewFloat64ObserveConfig(opts...)
	}))

	t.Run("Int64ObserveConfig", testConfAttr(func(mo ...MeasurementOption) attrConf {
		opts := make([]Int64ObserveOption, len(mo))
		for i := range mo {
			opts[i] = mo[i].(Int64ObserveOption)
		}
		return NewInt64ObserveConfig(opts...)
	}))
}

func testConfAttr(newConf func(...MeasurementOption) attrConf) func(t *testing.T) {
	return func(t *testing.T) {
		t.Run("ZeroConfigEmpty", func(t *testing.T) {
			c := newConf()
			assert.Equal(t, *attribute.EmptySet(), c.Attributes())
		})

		t.Run("EmptySet", func(t *testing.T) {
			c := newConf(WithAttributeSet(*attribute.EmptySet()))
			assert.Equal(t, *attribute.EmptySet(), c.Attributes())
		})

		aliceAttr := attribute.String("user", "Alice")
		alice := attribute.NewSet(aliceAttr)
		t.Run("SingleWithAttributeSet", func(t *testing.T) {
			c := newConf(WithAttributeSet(alice))
			assert.Equal(t, alice, c.Attributes())
		})

		t.Run("SingleWithAttributes", func(t *testing.T) {
			c := newConf(WithAttributes(aliceAttr))
			assert.Equal(t, alice, c.Attributes())
		})

		bobAttr := attribute.String("user", "Bob")
		bob := attribute.NewSet(bobAttr)
		t.Run("MultiWithAttributeSet", func(t *testing.T) {
			c := newConf(WithAttributeSet(alice), WithAttributeSet(bob))
			assert.Equal(t, bob, c.Attributes())
		})

		t.Run("MergedWithAttributes", func(t *testing.T) {
			c := newConf(WithAttributes(aliceAttr, bobAttr))
			assert.Equal(t, bob, c.Attributes())
		})

		t.Run("MultiWithAttributeSet", func(t *testing.T) {
			c := newConf(WithAttributes(aliceAttr), WithAttributes(bobAttr))
			assert.Equal(t, bob, c.Attributes())
		})

		t.Run("MergedEmpty", func(t *testing.T) {
			c := newConf(WithAttributeSet(alice), WithAttributeSet(*attribute.EmptySet()))
			assert.Equal(t, alice, c.Attributes())
		})
	}
}
