package internal

import (
	"errors"
	"math"
	"sync"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/metric/aggregation"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
)

const (
	expoMaxScale = 20
	expoMinScale = -10

	smallestNonZeroNormalFloat64 = 0x1p-1022
)

// expoHistogramValues summarizes a set of measurements as an histValues with
// explicitly defined buckets.
type expoHistogramValues[N int64 | float64] struct {
	maxSize       int
	maxScale      int
	zeroThreshold float64

	values   map[attribute.Set]*expoHistogramDataPoint[N]
	valuesMu sync.Mutex
}

func (e *expoHistogramValues[N]) Aggregate(value N, attr attribute.Set) {
	e.valuesMu.Lock()
	defer e.valuesMu.Unlock()

	v, ok := e.values[attr]
	if !ok {
		v = NewExpoHistogramDataPoint[N](e.maxSize, e.maxScale, e.zeroThreshold)
		e.values[attr] = v
	}
	v.record(value)
}

// expoHistogramDataPoint is a single bucket in an exponential histogram.
type expoHistogramDataPoint[N int64 | float64] struct {
	count uint64
	min   N
	max   N
	sum   N

	maxSize       int
	zeroThreshold float64

	scale int

	posBuckets expoBucket
	negBuckets expoBucket
	zeroCount  uint64
}

func NewExpoHistogramDataPoint[N int64 | float64](maxSize, maxScale int, zeroThreshold float64) *expoHistogramDataPoint[N] {

	f := math.MaxFloat64
	max := N(f)
	if N(math.MaxInt64) > N(f) {
		max = N(math.MaxInt64)
	}
	min := N(-f)
	if N(math.MinInt64) < N(-f) {
		min = N(math.MinInt64)
	}
	return &expoHistogramDataPoint[N]{
		min:           max,
		max:           min,
		maxSize:       maxSize,
		zeroThreshold: zeroThreshold,
		scale:         maxScale,
	}
}

func (p *expoHistogramDataPoint[N]) record(v N) {
	p.count++
	if v < p.min {
		p.min = v
	}
	if v > p.max {
		p.max = v
	}
	p.sum += v

	absV := math.Abs(float64(v))
	if float64(absV) <= p.zeroThreshold {
		p.zeroCount++
		return
	}

	if absV < smallestNonZeroNormalFloat64 {
		absV = smallestNonZeroNormalFloat64
	}

	index := getIndex(absV, p.scale)

	bucket := &p.posBuckets
	if v < 0 {
		bucket = &p.negBuckets
	}

	// If the new index would make the counts larger than maxScale, we need to
	// downscale current measurements.
	if needRescale(index, bucket.startIndex, len(bucket.counts), p.maxSize) {
		scale := scaleChange(index, bucket.startIndex, len(bucket.counts), p.maxSize)
		if p.scale-scale < expoMinScale {
			otel.Handle(errors.New("exponential histogram scale underflow"))
			return
		}
		//Downscale
		p.scale -= scale
		p.posBuckets.downscale(scale)
		p.negBuckets.downscale(scale)

		index = getIndex(absV, p.scale)
	}

	bucket.record(index)
}

func getIndex(v float64, scale int) int {
	if scale <= 0 {
		return getExpoIndex(v, scale)
	}
	return getLogIndex(v, scale)
}

// scaleFactors are constants used in calculating the index.  They are equivalent to
// 2^index/log(2)
var scaleFactors = [21]float64{
	math.Ldexp(math.Log2E, 0),
	math.Ldexp(math.Log2E, 1),
	math.Ldexp(math.Log2E, 2),
	math.Ldexp(math.Log2E, 3),
	math.Ldexp(math.Log2E, 4),
	math.Ldexp(math.Log2E, 5),
	math.Ldexp(math.Log2E, 6),
	math.Ldexp(math.Log2E, 7),
	math.Ldexp(math.Log2E, 8),
	math.Ldexp(math.Log2E, 9),
	math.Ldexp(math.Log2E, 10),
	math.Ldexp(math.Log2E, 11),
	math.Ldexp(math.Log2E, 12),
	math.Ldexp(math.Log2E, 13),
	math.Ldexp(math.Log2E, 14),
	math.Ldexp(math.Log2E, 15),
	math.Ldexp(math.Log2E, 16),
	math.Ldexp(math.Log2E, 17),
	math.Ldexp(math.Log2E, 18),
	math.Ldexp(math.Log2E, 19),
	math.Ldexp(math.Log2E, 20),
}

func getExpoIndex(v float64, scale int) int {
	// Extract the raw exponent.
	rawExp := getNormalBase2(v)

	// In case the value is an exact power of two, compute a
	// correction of -1:
	correction := (getSignificand(v) - 1) >> significandWidth

	// Note: bit-shifting does the right thing for negative
	// exponents, e.g., -1 >> 1 == -1.
	return (rawExp + correction) >> (-scale)
}

func getLogIndex(v float64, scale int) int {
	// Exact power-of-two correctness: an optional special case.
	if getSignificand(v) == 0 {
		exp := getNormalBase2(v)
		return (exp << scale) - 1
	}

	// Non-power of two cases.  Use Floor(x) to round the scaled
	// logarithm.  We could use Ceil(x)-1 to achieve the same
	// result, though Ceil() is typically defined as -Floor(-x)
	// and typically not performed in hardware, so this is likely
	// less code.
	return int(math.Floor(math.Log(v) * scaleFactors[scale]))
}

func needRescale(index, startIndex, length, maxSize int) bool {
	if length == 0 {
		return false
	}

	endIndex := startIndex + length - 1
	return index-startIndex >= maxSize || endIndex-index >= maxSize
}

func scaleChange(index, startIndex, length, maxSize int) int {
	var low, high int
	if index > startIndex {
		low = startIndex
		high = index
	} else {
		low = index
		high = startIndex + length - 1
	}

	if low > high {
		low, high = high, low
	}
	count := 0
	for high-low >= maxSize {
		low = low >> 1
		high = high >> 1
		count++
	}
	return count
}

// expoBucket is a single bucket in an exponential histogram.
type expoBucket struct {
	startIndex int
	counts     []uint64
}

func (b *expoBucket) record(index int) {
	if len(b.counts) == 0 {
		b.counts = []uint64{1}
		b.startIndex = index
		return
	}

	endIndex := b.startIndex + len(b.counts) - 1

	// if the new index is inside the current range
	if index >= b.startIndex && index <= endIndex {
		b.counts[index-b.startIndex]++
		return
	}
	// if the new index is before the current start add spaces to the counts
	if index < b.startIndex {
		// TODO: if counts has the capacity just prepend.
		temp := make([]uint64, endIndex-index+1)
		copy(temp[b.startIndex-index:], b.counts)
		b.counts = temp
		b.startIndex = index
		b.counts[0] = 1
		return
	}
	// if the new is after the end add spaces to the end
	if index >= endIndex {
		// TODO, if counts has the capacity just append.
		end := make([]uint64, index-b.startIndex-len(b.counts)+1)
		b.counts = append(b.counts, end...)
		b.counts[index-b.startIndex] = 1
	}
}

func (b *expoBucket) downscale(s int) {
	if len(b.counts) <= 1 || s < 1 {
		b.startIndex = b.startIndex >> s
		return
	}

	steps := 1 << s
	offset := b.startIndex % steps
	offset = (offset + steps) % steps // to make offset positive
	for i := 1; i < len(b.counts); i++ {
		idx := i + offset
		if idx%steps == 0 {
			b.counts[idx/steps] = b.counts[i]
			continue
		}
		b.counts[idx/steps] += b.counts[i]
	}

	lastIdx := (len(b.counts) - 1 + offset) / steps
	b.counts = b.counts[:lastIdx+1]
	b.startIndex = b.startIndex >> s
}

func NewDeltaExponentialHistogram[N int64 | float64](cfg aggregation.ExponentialHistogram) Aggregator[N] {
	if cfg.MaxScale > expoMaxScale {
		cfg.MaxScale = expoMaxScale
	}
	if cfg.MaxScale < expoMinScale {
		cfg.MaxScale = expoMinScale
	}
	if cfg.MaxSize <= 0 {
		cfg.MaxSize = 160
	}

	return &deltaExponentialHistogram[N]{
		expoHistogramValues: &expoHistogramValues[N]{
			maxSize:       cfg.MaxSize,
			maxScale:      cfg.MaxScale,
			zeroThreshold: math.Abs(cfg.ZeroThreshold),

			values: make(map[attribute.Set]*expoHistogramDataPoint[N]),
		},
		noMinMax: cfg.NoMinMax,
		start:    now(),
	}
}

// deltaExponentialHistogram summarizes a set of measurements made in a single
// aggregation cycle as an Exponential histogram with explicitly defined buckets.
type deltaExponentialHistogram[N int64 | float64] struct {
	*expoHistogramValues[N]

	noMinMax bool
	start    time.Time
}

func (e *deltaExponentialHistogram[N]) Aggregation() metricdata.Aggregation {
	e.valuesMu.Lock()
	defer e.valuesMu.Unlock()

	if len(e.values) == 0 {
		return nil
	}
	t := now()
	h := metricdata.ExponentialHistogram[N]{
		Temporality: metricdata.DeltaTemporality,
		DataPoints:  make([]metricdata.ExponentialHistogramDataPoint[N], 0, len(e.values)),
	}
	for a, b := range e.values {
		ehdp := metricdata.ExponentialHistogramDataPoint[N]{
			Attributes:    a,
			StartTime:     e.start,
			Time:          t,
			Count:         b.count,
			Sum:           b.sum,
			Scale:         int32(b.scale),
			ZeroCount:     b.zeroCount,
			ZeroThreshold: b.zeroThreshold,
			PositiveBucket: metricdata.ExponentialBucket{
				Offset: int32(b.posBuckets.startIndex),
				Counts: make([]uint64, len(b.posBuckets.counts)),
			},
			NegativeBucket: metricdata.ExponentialBucket{
				Offset: int32(b.negBuckets.startIndex),
				Counts: make([]uint64, len(b.negBuckets.counts)),
			},
		}
		copy(ehdp.PositiveBucket.Counts, b.posBuckets.counts)
		copy(ehdp.NegativeBucket.Counts, b.negBuckets.counts)

		if !e.noMinMax {
			ehdp.Min = metricdata.NewExtrema(b.min)
			ehdp.Max = metricdata.NewExtrema(b.max)
		}
		h.DataPoints = append(h.DataPoints, ehdp)

		delete(e.values, a)
	}
	e.start = t
	return h
}

func NewCumulativeExponentialHistogram[N int64 | float64](cfg aggregation.ExponentialHistogram) Aggregator[N] {
	if cfg.MaxScale > expoMaxScale {
		cfg.MaxScale = expoMaxScale
	}
	if cfg.MaxScale < expoMinScale {
		cfg.MaxScale = expoMinScale
	}
	if cfg.MaxSize <= 0 {
		cfg.MaxSize = 160
	}

	return &cumulativeExponentialHistogram[N]{
		expoHistogramValues: &expoHistogramValues[N]{
			maxSize:       cfg.MaxSize,
			maxScale:      cfg.MaxScale,
			zeroThreshold: math.Abs(cfg.ZeroThreshold),

			values: make(map[attribute.Set]*expoHistogramDataPoint[N]),
		},
		noMinMax: cfg.NoMinMax,
		start:    now(),
	}
}

// cumulativeExponentialHistogram summarizes a set of measurements made in a single
// aggregation cycle as an Exponential histogram with explicitly defined buckets.
type cumulativeExponentialHistogram[N int64 | float64] struct {
	*expoHistogramValues[N]

	noMinMax bool
	start    time.Time
}

func (e *cumulativeExponentialHistogram[N]) Aggregation() metricdata.Aggregation {
	e.valuesMu.Lock()
	defer e.valuesMu.Unlock()

	if len(e.values) == 0 {
		return nil
	}
	t := now()
	h := metricdata.ExponentialHistogram[N]{
		Temporality: metricdata.DeltaTemporality,
		DataPoints:  make([]metricdata.ExponentialHistogramDataPoint[N], 0, len(e.values)),
	}
	for a, b := range e.values {
		ehdp := metricdata.ExponentialHistogramDataPoint[N]{
			Attributes:    a,
			StartTime:     e.start,
			Time:          t,
			Count:         b.count,
			Sum:           b.sum,
			Scale:         int32(b.scale),
			ZeroCount:     b.zeroCount,
			ZeroThreshold: b.zeroThreshold,
			PositiveBucket: metricdata.ExponentialBucket{
				Offset: int32(b.posBuckets.startIndex),
				Counts: make([]uint64, len(b.posBuckets.counts)),
			},
			NegativeBucket: metricdata.ExponentialBucket{
				Offset: int32(b.negBuckets.startIndex),
				Counts: make([]uint64, len(b.negBuckets.counts)),
			},
		}
		copy(ehdp.PositiveBucket.Counts, b.posBuckets.counts)
		copy(ehdp.NegativeBucket.Counts, b.negBuckets.counts)

		if !e.noMinMax {
			ehdp.Min = metricdata.NewExtrema(b.min)
			ehdp.Max = metricdata.NewExtrema(b.max)
		}
		h.DataPoints = append(h.DataPoints, ehdp)
		// TODO (#3006): This will use an unbounded amount of memory if there
		// are unbounded number of attribute sets being aggregated. Attribute
		// sets that become "stale" need to be forgotten so this will not
		// overload the system.
	}

	return h
}
