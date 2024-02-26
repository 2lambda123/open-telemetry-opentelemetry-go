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

package aggregate // import "go.opentelemetry.io/otel/sdk/metric/internal/aggregate"

import (
	"context"
	"slices"
	"sort"
	"sync"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/metric/internal/exemplar"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
)

type buckets[N int64 | float64] struct {
	res exemplar.Reservoir[N]

	counts   []uint64
	count    uint64
	total    N
	min, max N
}

// newBuckets returns buckets with n bins.
func newBuckets[N int64 | float64](n int) *buckets[N] {
	return &buckets[N]{counts: make([]uint64, n)}
}

func (b *buckets[N]) sum(value N) { b.total += value }

func (b *buckets[N]) bin(idx int, value N) {
	b.counts[idx]++
	b.count++
	if value < b.min {
		b.min = value
	} else if value > b.max {
		b.max = value
	}
}

// histValues summarizes a set of measurements as an histValues with
// explicitly defined buckets.
type histValues[N int64 | float64] struct {
	noSum  bool
	bounds []float64

	newRes   func() exemplar.Reservoir[N]
	limit    limiter[*buckets[N]]
	values   map[attribute.Set]*buckets[N]
	valuesMu sync.Mutex
}

func newHistValues[N int64 | float64](bounds []float64, noSum bool, limit int, r func() exemplar.Reservoir[N]) *histValues[N] {
	// The responsibility of keeping all buckets correctly associated with the
	// passed boundaries is ultimately this type's responsibility. Make a copy
	// here so we can always guarantee this. Or, in the case of failure, have
	// complete control over the fix.
	b := slices.Clone(bounds)
	slices.Sort(b)
	return &histValues[N]{
		noSum:  noSum,
		bounds: b,
		newRes: r,
		limit:  newLimiter[*buckets[N]](limit),
		values: make(map[attribute.Set]*buckets[N]),
	}
}

// Aggregate records the measurement value, scoped by attr, and aggregates it
// into a histogram.
func (s *histValues[N]) measure(ctx context.Context, value N, fltrAttr attribute.Set, droppedAttr []attribute.KeyValue) {
	// This search will return an index in the range [0, len(s.bounds)], where
	// it will return len(s.bounds) if value is greater than the last element
	// of s.bounds. This aligns with the buckets in that the length of buckets
	// is len(s.bounds)+1, with the last bucket representing:
	// (s.bounds[len(s.bounds)-1], +∞).
	idx := sort.SearchFloat64s(s.bounds, float64(value))

	t := now()

	s.valuesMu.Lock()
	defer s.valuesMu.Unlock()

	attr := s.limit.Attributes(fltrAttr, s.values)
	b, ok := s.values[attr]
	if !ok {
		// N+1 buckets. For example:
		//
		//   bounds = [0, 5, 10]
		//
		// Then,
		//
		//   buckets = (-∞, 0], (0, 5.0], (5.0, 10.0], (10.0, +∞)
		b = newBuckets[N](len(s.bounds) + 1)
		b.res = s.newRes()

		// Ensure min and max are recorded values (not zero), for new buckets.
		b.min, b.max = value, value
		s.values[attr] = b
	}
	b.bin(idx, value)
	if !s.noSum {
		b.sum(value)
	}
	b.res.Offer(ctx, t, value, droppedAttr)
}

// newHistogram returns an Aggregator that summarizes a set of measurements as
// an histogram.
func newHistogram[N int64 | float64](boundaries []float64, noMinMax, noSum bool, limit int, r func() exemplar.Reservoir[N]) *histogram[N] {
	return &histogram[N]{
		histValues: newHistValues[N](boundaries, noSum, limit, r),
		noMinMax:   noMinMax,
		start:      now(),
	}
}

// histogram summarizes a set of measurements as an histogram with explicitly
// defined buckets.
type histogram[N int64 | float64] struct {
	*histValues[N]

	noMinMax bool
	start    time.Time
}

func (s *histogram[N]) delta(dest *metricdata.Aggregation) int {
	t := now()

	// If *dest is not a metricdata.Histogram, memory reuse is missed. In that
	// case, use the zero-value h and hope for better alignment next cycle.
	h, _ := (*dest).(metricdata.Histogram[N])
	h.Temporality = metricdata.DeltaTemporality

	s.valuesMu.Lock()
	defer s.valuesMu.Unlock()

	// Do not allow modification of our copy of bounds.
	bounds := slices.Clone(s.bounds)

	n := len(s.values)
	hDPts := reset(h.DataPoints, n, n)

	var i int
	for a, b := range s.values {
		hDPts[i].Attributes = a
		hDPts[i].StartTime = s.start
		hDPts[i].Time = t
		hDPts[i].Count = b.count
		hDPts[i].Bounds = bounds
		hDPts[i].BucketCounts = b.counts

		if !s.noSum {
			hDPts[i].Sum = b.total
		}

		if !s.noMinMax {
			hDPts[i].Min = metricdata.NewExtrema(b.min)
			hDPts[i].Max = metricdata.NewExtrema(b.max)
		}

		b.res.Collect(&hDPts[i].Exemplars)

		// Unused attribute sets do not report.
		delete(s.values, a)
		i++
	}
	// The delta collection cycle resets.
	s.start = t

	h.DataPoints = hDPts
	*dest = h

	return n
}

func (s *histogram[N]) cumulative(dest *metricdata.Aggregation) int {
	t := now()

	// If *dest is not a metricdata.Histogram, memory reuse is missed. In that
	// case, use the zero-value h and hope for better alignment next cycle.
	h, _ := (*dest).(metricdata.Histogram[N])
	h.Temporality = metricdata.CumulativeTemporality

	s.valuesMu.Lock()
	defer s.valuesMu.Unlock()

	// Do not allow modification of our copy of bounds.
	bounds := slices.Clone(s.bounds)

	n := len(s.values)
	hDPts := reset(h.DataPoints, n, n)

	var i int
	for a, b := range s.values {
		hDPts[i].Attributes = a
		hDPts[i].StartTime = s.start
		hDPts[i].Time = t
		hDPts[i].Count = b.count
		hDPts[i].Bounds = bounds

		// The HistogramDataPoint field values returned need to be copies of
		// the buckets value as we will keep updating them.
		//
		// TODO (#3047): Making copies for bounds and counts incurs a large
		// memory allocation footprint. Alternatives should be explored.
		hDPts[i].BucketCounts = slices.Clone(b.counts)

		if !s.noSum {
			hDPts[i].Sum = b.total
		}

		if !s.noMinMax {
			hDPts[i].Min = metricdata.NewExtrema(b.min)
			hDPts[i].Max = metricdata.NewExtrema(b.max)
		}

		b.res.Collect(&hDPts[i].Exemplars)

		i++
		// TODO (#3006): This will use an unbounded amount of memory if there
		// are unbounded number of attribute sets being aggregated. Attribute
		// sets that become "stale" need to be forgotten so this will not
		// overload the system.
	}

	h.DataPoints = hDPts
	*dest = h

	return n
}
