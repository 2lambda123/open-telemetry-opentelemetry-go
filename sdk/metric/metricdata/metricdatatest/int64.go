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

//go:build go1.18
// +build go1.18

package metricdatatest // import "go.opentelemetry.io/otel/sdk/metric/metricdata/metricdatatest"

import (
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
)

// equalInt64 returns true when Int64s are equal. It returns false when they
// differ, along with the reasons why they differ.
func equalInt64(a, b metricdata.Int64) (equal bool, reasons []string) {
	equal = a == b
	if !equal {
		reasons = append(reasons, notEqualStr("Int64 value", a, b))
	}
	return equal, reasons
}
