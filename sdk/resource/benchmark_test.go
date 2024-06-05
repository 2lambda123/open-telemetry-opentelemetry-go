// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package resource_test

import (
	"context"
	"fmt"
	"math/rand"
	"testing"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/resource"
)

const conflict = 0.5

func makeAttrs(n int) (_, _ *resource.Resource) {
	used := map[string]bool{}
	l1 := make([]attribute.KeyValue, n)
	l2 := make([]attribute.KeyValue, n)
	for i := 0; i < n; i++ {
		var k string
		for {
			k = fmt.Sprint("k", rand.Intn(1000000000))
			if !used[k] {
				used[k] = true
				break
			}
		}
		l1[i] = attribute.String(k, fmt.Sprint("v", rand.Intn(1000000000)))

		if rand.Float64() < conflict {
			l2[i] = l1[i]
		} else {
			l2[i] = attribute.String(k, fmt.Sprint("v", rand.Intn(1000000000)))
		}
	}
	return resource.NewSchemaless(l1...), resource.NewSchemaless(l2...)
}

func benchmarkMergeResource(b *testing.B, size int) {
	r1, r2 := makeAttrs(size)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _ = resource.Merge(r1, r2)
	}
}

func BenchmarkMergeResource_1(b *testing.B) {
	benchmarkMergeResource(b, 1)
}

func BenchmarkMergeResource_2(b *testing.B) {
	benchmarkMergeResource(b, 2)
}

func BenchmarkMergeResource_3(b *testing.B) {
	benchmarkMergeResource(b, 3)
}

func BenchmarkMergeResource_4(b *testing.B) {
	benchmarkMergeResource(b, 4)
}

func BenchmarkMergeResource_6(b *testing.B) {
	benchmarkMergeResource(b, 6)
}

func BenchmarkMergeResource_8(b *testing.B) {
	benchmarkMergeResource(b, 8)
}

func BenchmarkMergeResource_16(b *testing.B) {
	benchmarkMergeResource(b, 16)
}

type testDetector struct {
	durationMilliSecond time.Duration
}

// instant detector don't do anything
// its benchmark the overhead of the resource.New implementation.
func (f testDetector) Detect(_ context.Context) (*resource.Resource, error) {
	time.Sleep(f.durationMilliSecond * time.Millisecond)
	return resource.NewSchemaless(), nil
}

func benchmarkOverhead(ctx context.Context, b *testing.B, testedDetector resource.Detector, n int) {
	detectors := []resource.Detector{}
	for i := 0; i < n; i++ {
		detectors = append(detectors, testedDetector)
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = resource.New(ctx, resource.WithDetectors(detectors...))
	}
}

func BenchmarkDetectors(b *testing.B) {
	type detectorType struct {
		name string
		resource.Detector
	}
	detectors := []detectorType{
		{
			name:     "instantDetector",
			Detector: testDetector{durationMilliSecond: 0},
		},
		{
			name:     "fastDetector",
			Detector: testDetector{durationMilliSecond: 1},
		},
		{
			name:     "mediumDetector",
			Detector: testDetector{durationMilliSecond: 30},
		},
		{
			name:     "slowDetector",
			Detector: testDetector{durationMilliSecond: 500},
		},
	}
	for _, detectorType := range detectors {
		b.Run(detectorType.name, func(b *testing.B) {
			for _, numberOfDetectors := range []int{1, 2, 4, 8, 16} {
				b.Run(fmt.Sprintf("%d-detectors", numberOfDetectors), func(b *testing.B) {
					benchmarkOverhead(context.Background(), b, testDetector{durationMilliSecond: 0}, numberOfDetectors)
				})
			}
		})
	}
}

func BenchmarkDefaultResource(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = resource.Default()
	}
}
