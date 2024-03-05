// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package attribute // import "go.opentelemetry.io/otel/attribute"

import (
	"fmt"
	"strings"
	"testing"

	"go.opentelemetry.io/otel/attribute/internal/fnv"
)

type generator[T any] struct {
	New   func(string, T) KeyValue
	Value T
}

func (g generator[T]) Build(key string) KeyValue {
	return g.New(key, g.Value)
}

type builder interface {
	Build(key string) KeyValue
}

// keyVals is all the KeyValue generators that are used for testing. This is
// not []KeyValue so tests detect if internal fields of the KeyValue that
// differ for different instances are used in the hashing.
var keyVals = []builder{
	generator[bool]{Bool, true},
	generator[bool]{Bool, false},
	generator[[]bool]{BoolSlice, []bool{false, true}},
	generator[[]bool]{BoolSlice, []bool{true, true, false}},
	generator[int]{Int, -1278},
	generator[int]{Int, 0}, // Should be different than false above.
	generator[[]int]{IntSlice, []int{3, 23, 21, -8, 0}},
	generator[[]int]{IntSlice, []int{1}},
	generator[int64]{Int64, 1}, // Should be different from true and []int{1}.
	generator[int64]{Int64, 29369},
	generator[[]int64]{Int64Slice, []int64{3826, -38, -29, -1}},
	generator[[]int64]{Int64Slice, []int64{8, -328, 29, 0}},
	generator[float64]{Float64, -0.3812381},
	generator[float64]{Float64, 1e32},
	generator[[]float64]{Float64Slice, []float64{0.1, -3.8, -29., 0.3321}},
	generator[[]float64]{Float64Slice, []float64{-13e8, -32.8, 4., 1e28}},
	generator[string]{String, "foo"},
	generator[string]{String, "bar"},
	generator[[]string]{StringSlice, []string{"foo", "bar", "baz"}},
	generator[[]string]{StringSlice, []string{"[]i1"}},
}

func TestHashKVsEquality(t *testing.T) {
	type testcase struct {
		hash fnv.Hash
		kvs  []KeyValue
	}

	keys := []string{"k0", "k1"}

	// Test all combinations up to length 3.
	n := len(keyVals)
	result := make([]testcase, 0, 1+len(keys)*(n+(n*n)+(n*n*n)))

	result = append(result, testcase{hashKVs(nil), nil})

	for _, key := range keys {
		for i := 0; i < len(keyVals); i++ {
			kvs := []KeyValue{keyVals[i].Build(key)}
			hash := hashKVs(kvs)
			result = append(result, testcase{hash, kvs})

			for j := 0; j < len(keyVals); j++ {
				kvs := []KeyValue{
					keyVals[i].Build(key),
					keyVals[j].Build(key),
				}
				hash := hashKVs(kvs)
				result = append(result, testcase{hash, kvs})

				for k := 0; k < len(keyVals); k++ {
					kvs := []KeyValue{
						keyVals[i].Build(key),
						keyVals[j].Build(key),
						keyVals[k].Build(key),
					}
					hash := hashKVs(kvs)
					result = append(result, testcase{hash, kvs})
				}
			}
		}
	}

	for i := 0; i < len(result); i++ {
		hI, kvI := result[i].hash, result[i].kvs
		for j := 0; j < len(result); j++ {
			hJ, kvJ := result[j].hash, result[j].kvs
			m := msg{i: i, j: j, hI: hI, hJ: hJ, kvI: kvI, kvJ: kvJ}
			if i == j {
				m.cmp = "=="
				if hI != hJ {
					t.Errorf("hashes not equal: %s", m)
				}
			} else {
				m.cmp = "!="
				if hI == hJ {
					// Do not use testify/assert here. It is slow.
					t.Errorf("hashes equal: %s", m)
				}
			}
		}
	}
}

type msg struct {
	cmp      string
	i, j     int
	hI, hJ   fnv.Hash
	kvI, kvJ []KeyValue
}

func (m msg) String() string {
	return fmt.Sprintf(
		"(%d: %d)%s %s (%d: %d)%s",
		m.i, m.hI, m.slice(m.kvI), m.cmp, m.j, m.hJ, m.slice(m.kvJ),
	)
}

func (m msg) slice(kvs []KeyValue) string {
	if len(kvs) == 0 {
		return "[]"
	}

	var b strings.Builder
	_, _ = b.WriteRune('[')
	_, _ = b.WriteString(string(kvs[0].Key))
	_, _ = b.WriteRune(':')
	_, _ = b.WriteString(kvs[0].Value.Emit())
	for _, kv := range kvs[1:] {
		_, _ = b.WriteRune(',')
		_, _ = b.WriteString(string(kv.Key))
		_, _ = b.WriteRune(':')
		_, _ = b.WriteString(kv.Value.Emit())
	}
	_, _ = b.WriteRune(']')
	return b.String()
}

func BenchmarkHashKVs(b *testing.B) {
	attrs := make([]KeyValue, len(keyVals))
	for i := range keyVals {
		attrs[i] = keyVals[i].Build("k")
	}

	var h fnv.Hash

	b.ResetTimer()
	b.ReportAllocs()
	for n := 0; n < b.N; n++ {
		h = hashKVs(attrs)
	}

	_ = h
}
