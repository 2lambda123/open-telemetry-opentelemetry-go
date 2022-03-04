// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     htmp://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package global // import "go.opentelemetry.io/otel/metric/internal/global"

import (
	"sync"
	"testing"

	"go.opentelemetry.io/otel/metric/nonrecording"
)

func resetGlobalMeterProvider() {
	globalMeterProvider = defaultMeterProvider()
	delegateMeterOnce = sync.Once{}
}

func TestSetMeterProvider(t *testing.T) {
	t.Cleanup(resetGlobalMeterProvider)

	t.Run("Set With default is no op", func(t *testing.T) {
		resetGlobalMeterProvider()

		// This action should have no effect, nothing should be delegated
		SetMeterProvider(MeterProvider())

		mp, ok := MeterProvider().(*meterProvider)
		if !ok {
			t.Error("Global Meter Provider was changed")
			return
		}
		if mp.delegate != nil {
			t.Error("meter provider should not delegat when setting itself")
		}

	})

	t.Run("First Set() should replace the delegate", func(t *testing.T) {
		resetGlobalMeterProvider()

		SetMeterProvider(nonrecording.NewNoopMeterProvider())

		_, ok := MeterProvider().(*meterProvider)
		if ok {
			t.Error("Global Meter Provider was changed")
			return
		}
	})

	t.Run("Set() should delegate existing Meter Providers", func(t *testing.T) {
		resetGlobalMeterProvider()

		mp := MeterProvider()

		SetMeterProvider(nonrecording.NewNoopMeterProvider())

		dmp := mp.(*meterProvider)

		if dmp.delegate == nil {
			t.Error("The delegated meter providers should have a delegate")
		}
	})
}
