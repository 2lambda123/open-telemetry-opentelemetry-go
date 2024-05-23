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
	semconv "go.opentelemetry.io/otel/semconv/v1.25.0"
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

type instantDetector struct{}

// instant detector don't do anything
// its benchmark the overhead of the resource.New implementation
func (f instantDetector) Detect(_ context.Context) (*resource.Resource, error) {
	return resource.NewSchemaless(), nil
}

type fastDetector struct{}

func (f fastDetector) Detect(_ context.Context) (*resource.Resource, error) {
	time.Sleep(time.Millisecond)
	return resource.NewSchemaless(), nil
}

type mediumDetector struct{}

func (f mediumDetector) Detect(_ context.Context) (*resource.Resource, error) {
	time.Sleep(time.Millisecond * 30)
	return resource.NewSchemaless(semconv.ServerAddress("localhost")), nil
}

type slowDetector struct{}

func (f slowDetector) Detect(_ context.Context) (*resource.Resource, error) {
	time.Sleep(time.Millisecond * 500)
	return resource.NewSchemaless(semconv.ServerAddress("localhost"), semconv.MessageID(rand.Int())), nil
}

var _ resource.Detector = &fakeDetector{}

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

func BenchmarkNewResourceOverHead_1(b *testing.B) {
	benchmarkOverhead(context.Background(), b, instantDetector{}, 1)
}

func BenchmarkNewResourceOverHead_2(b *testing.B) {
	benchmarkOverhead(context.Background(), b, instantDetector{}, 2)
}

func BenchmarkNewResourceOverHead_4(b *testing.B) {
	benchmarkOverhead(context.Background(), b, instantDetector{}, 4)
}

func BenchmarkNewResourceOverHead_8(b *testing.B) {
	benchmarkOverhead(context.Background(), b, instantDetector{}, 8)
}

func BenchmarkNewResourceOverHead_16(b *testing.B) {
	benchmarkOverhead(context.Background(), b, instantDetector{}, 16)
}

// fast

func BenchmarkFastDetector_1(b *testing.B) {
	benchmarkOverhead(context.Background(), b, fastDetector{}, 1)
}

func BenchmarkFastDetector_2(b *testing.B) {
	benchmarkOverhead(context.Background(), b, fastDetector{}, 2)
}

func BenchmarkFastDetector_4(b *testing.B) {
	benchmarkOverhead(context.Background(), b, fastDetector{}, 4)
}

func BenchmarkFastDetector_8(b *testing.B) {
	benchmarkOverhead(context.Background(), b, fastDetector{}, 8)
}

func BenchmarkFastDetector_16(b *testing.B) {
	benchmarkOverhead(context.Background(), b, fastDetector{}, 16)
}

// medium
func BenchmarkMediumDetector_1(b *testing.B) {
	benchmarkOverhead(context.Background(), b, mediumDetector{}, 1)
}

func BenchmarkMediumDetector_2(b *testing.B) {
	benchmarkOverhead(context.Background(), b, mediumDetector{}, 2)
}

func BenchmarkMediumDetector_4(b *testing.B) {
	benchmarkOverhead(context.Background(), b, mediumDetector{}, 4)
}

func BenchmarkMediumDetector_8(b *testing.B) {
	benchmarkOverhead(context.Background(), b, fastDetector{}, 8)
}

func BenchmarkMediumDetector_16(b *testing.B) {
	benchmarkOverhead(context.Background(), b, fastDetector{}, 16)
}

//slow

func BenchmarkSlowDetector_1(b *testing.B) {
	benchmarkOverhead(context.Background(), b, slowDetector{}, 1)
}

func BenchmarkSlowDetector_2(b *testing.B) {
	benchmarkOverhead(context.Background(), b, slowDetector{}, 2)
}

func BenchmarkSlowDetector_4(b *testing.B) {
	benchmarkOverhead(context.Background(), b, slowDetector{}, 4)
}

func BenchmarkSlowDetector_8(b *testing.B) {
	benchmarkOverhead(context.Background(), b, slowDetector{}, 8)
}

func BenchmarkSlowDetector_16(b *testing.B) {
	benchmarkOverhead(context.Background(), b, slowDetector{}, 16)
}

func BenchmarkDefaultResource(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = resource.Default()
	}
}
