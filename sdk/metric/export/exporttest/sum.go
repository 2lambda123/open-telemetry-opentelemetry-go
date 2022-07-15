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
	"fmt"
	"testing"

	"go.opentelemetry.io/otel/sdk/metric/export"
)

// CompareSum returns true when Sums are equivalent. It returns false when
// they differ, along with messages describing the difference.
//
// The DataPoints each Sum contains are compared based on containing the same
// DataPoints, not the order they are stored in.
func CompareSum(a, b export.Sum) (equal bool, explination []string) {
	equal = true
	if a.Temporality != b.Temporality {
		equal, explination = false, append(
			explination,
			notEqualStr("Temporality", a.Temporality, b.Temporality),
		)
	}
	if a.IsMonotonic != b.IsMonotonic {
		equal, explination = false, append(
			explination,
			notEqualStr("IsMonotonic", a.IsMonotonic, b.IsMonotonic),
		)
	}

	var exp string
	equal, exp = compareDiff(diffSlices(
		a.DataPoints,
		b.DataPoints,
		func(a, b export.DataPoint) bool {
			equal, _ := CompareDataPoint(a, b)
			return equal
		},
	))
	if !equal {
		explination = append(explination, fmt.Sprintf(
			"Sum DataPoints not equal:\n%s", exp,
		))
	}
	return equal, explination
}

// AssertSumsEqual asserts that two Sum are equal.
func AssertSumsEqual(t *testing.T, expected, actual export.Sum) bool {
	return assertCompare(CompareSum(expected, actual))(t)
}
