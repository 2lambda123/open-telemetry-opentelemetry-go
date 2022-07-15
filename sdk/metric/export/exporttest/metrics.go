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

package exporttest

import (
	"testing"

	"go.opentelemetry.io/otel/sdk/metric/export"
)

// CompareMetrics returns true when Metrics are equivalent. It returns false
// when they differ, along with messages describing the difference.
func CompareMetrics(a, b export.Metrics) (equal bool, explination []string) {
	equal = true
	if a.Name != b.Name {
		equal, explination = false, append(
			explination,
			notEqualStr("Name", a.Name, b.Name),
		)
	}
	if a.Description != b.Description {
		equal, explination = false, append(
			explination,
			notEqualStr("Description", a.Description, b.Description),
		)
	}
	if a.Unit != b.Unit {
		equal, explination = false, append(
			explination,
			notEqualStr("Unit", a.Unit, b.Unit),
		)
	}

	var exp []string
	equal, exp = CompareAggregations(a.Data, b.Data)
	if !equal {
		explination = append(explination, "Metrics Data not equal:")
		explination = append(explination, exp...)
	}
	return equal, explination
}

// AssertMetricsEqual asserts that two Metrics are equal.
func AssertMetricsEqual(t *testing.T, expected, actual export.Metrics) bool {
	return assertCompare(CompareMetrics(expected, actual))(t)
}
