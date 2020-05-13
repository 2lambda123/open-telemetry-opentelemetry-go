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

package key_test

import (
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"

	"go.opentelemetry.io/otel/api/key"
	"go.opentelemetry.io/otel/api/kv"
)

func TestKeyValueConstructors(t *testing.T) {
	tt := []struct {
		name     string
		actual   kv.KeyValue
		expected kv.KeyValue
	}{
		{
			name:   "Bool",
			actual: key.Bool("k1", true),
			expected: kv.KeyValue{
				Key:   "k1",
				Value: kv.Bool(true),
			},
		},
		{
			name:   "Int64",
			actual: key.Int64("k1", 123),
			expected: kv.KeyValue{
				Key:   "k1",
				Value: kv.Int64(123),
			},
		},
		{
			name:   "Uint64",
			actual: key.Uint64("k1", 1),
			expected: kv.KeyValue{
				Key:   "k1",
				Value: kv.Uint64(1),
			},
		},
		{
			name:   "Float64",
			actual: key.Float64("k1", 123.5),
			expected: kv.KeyValue{
				Key:   "k1",
				Value: kv.Float64(123.5),
			},
		},
		{
			name:   "Int32",
			actual: key.Int32("k1", 123),
			expected: kv.KeyValue{
				Key:   "k1",
				Value: kv.Int32(123),
			},
		},
		{
			name:   "Uint32",
			actual: key.Uint32("k1", 123),
			expected: kv.KeyValue{
				Key:   "k1",
				Value: kv.Uint32(123),
			},
		},
		{
			name:   "Float32",
			actual: key.Float32("k1", 123.5),
			expected: kv.KeyValue{
				Key:   "k1",
				Value: kv.Float32(123.5),
			},
		},
		{
			name:   "String",
			actual: key.String("k1", "123.5"),
			expected: kv.KeyValue{
				Key:   "k1",
				Value: kv.String("123.5"),
			},
		},
		{
			name:   "Int",
			actual: key.Int("k1", 123),
			expected: kv.KeyValue{
				Key:   "k1",
				Value: kv.Int(123),
			},
		},
		{
			name:   "Uint",
			actual: key.Uint("k1", 123),
			expected: kv.KeyValue{
				Key:   "k1",
				Value: kv.Uint(123),
			},
		},
	}

	for _, test := range tt {
		t.Run(test.name, func(t *testing.T) {
			if diff := cmp.Diff(test.actual, test.expected, cmp.AllowUnexported(kv.Value{})); diff != "" {
				t.Fatal(diff)
			}
		})
	}
}

func TestInfer(t *testing.T) {
	builder := &strings.Builder{}
	builder.WriteString("foo")
	for _, testcase := range []struct {
		key       string
		value     interface{}
		wantType  kv.ValueType
		wantValue interface{}
	}{
		{
			key:       "bool type inferred",
			value:     true,
			wantType:  kv.BOOL,
			wantValue: true,
		},
		{
			key:       "int64 type inferred",
			value:     int64(42),
			wantType:  kv.INT64,
			wantValue: int64(42),
		},
		{
			key:       "uint64 type inferred",
			value:     uint64(42),
			wantType:  kv.UINT64,
			wantValue: uint64(42),
		},
		{
			key:       "float64 type inferred",
			value:     float64(42.1),
			wantType:  kv.FLOAT64,
			wantValue: 42.1,
		},
		{
			key:       "int32 type inferred",
			value:     int32(42),
			wantType:  kv.INT32,
			wantValue: int32(42),
		},
		{
			key:       "uint32 type inferred",
			value:     uint32(42),
			wantType:  kv.UINT32,
			wantValue: uint32(42),
		},
		{
			key:       "float32 type inferred",
			value:     float32(42.1),
			wantType:  kv.FLOAT32,
			wantValue: float32(42.1),
		},
		{
			key:       "string type inferred",
			value:     "foo",
			wantType:  kv.STRING,
			wantValue: "foo",
		},
		{
			key:       "stringer type inferred",
			value:     builder,
			wantType:  kv.STRING,
			wantValue: "foo",
		},
		{
			key:       "unknown value serialized as %v",
			value:     nil,
			wantType:  kv.STRING,
			wantValue: "<nil>",
		},
	} {
		t.Logf("Running test case %s", testcase.key)
		kv := key.Infer(testcase.key, testcase.value)
		if kv.Value.Type() != testcase.wantType {
			t.Errorf("wrong value type, got %#v, expected %#v", kv.Value.Type(), testcase.wantType)
		}
		got := kv.Value.AsInterface()
		if diff := cmp.Diff(testcase.wantValue, got); diff != "" {
			t.Errorf("+got, -want: %s", diff)
		}
	}
}
