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

package exemplar

import (
	"context"
	"math"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFixedSize(t *testing.T) {
	t.Run("Int64", testReservoir[int64](func(n int) (Reservoir[int64], int) {
		return FixedSize[int64](n), n
	}))

	t.Run("Float64", testReservoir[float64](func(n int) (Reservoir[float64], int) {
		return FixedSize[float64](n), n
	}))
}

func TestFixedSizeSamplingCorrectness(t *testing.T) {
	intensity := 0.1
	sampleSize := 1000

	data := make([]float64, sampleSize*1000)
	for i := range data {
		data[i] = (-1.0 / intensity) * math.Log(rng.Float64())
	}
	// Sort to avoid position bias.
	sort.Float64s(data)

	r := FixedSize[float64](sampleSize)
	for _, value := range data {
		r.Offer(context.Background(), staticTime, value, alice)
	}

	var sum float64
	for _, m := range r.(*randRes[float64]).store {
		sum += m.Value
	}
	mean := sum / float64(sampleSize)

	assert.InDelta(t, 1/mean, intensity, 0.01)
}
