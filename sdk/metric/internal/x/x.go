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

// Package x contains support for OTel metric SDK experimental features.
//
// This package should only be used for features defined in the specification.
// It should not be used for experiments or new project ideas.
package x // import "go.opentelemetry.io/otel/sdk/metric/internal/x"

import (
	"os"
	"strconv"
	"strings"
)

var (
	// Exemplars is an experimental feature flag that defines if exemplars
	// should be recorded for metric data-points.
	//
	// To enable this feature set the OTEL_GO_X_EXEMPLAR environment variable
	// to the case-insensitive string value of "true" (i.e. "True" and "TRUE"
	// will also enable this).
	Exemplars = newFeature("EXEMPLAR", func(v string) (string, bool) {
		if strings.ToLower(v) == "true" {
			return v, true
		}
		return "", false
	})

	// CardinalityLimit is an experimental feature flag that defines if
	// cardinality limits should be applied to the recorded metric data-points.
	//
	// To enable this feature set the OTEL_GO_X_CARDINALITY_LIMIT environment
	// variable to the integer limit value you want to use.
	CardinalityLimit = newFeature("CARDINALITY_LIMIT", func(v string) (int, bool) {
		n, err := strconv.Atoi(v)
		if err != nil {
			return 0, false
		}
		return n, true
	})
)

// Feature is an experimental feature control flag. It provides a uniform way
// to interact with these feature flags and parse their values.
type Feature[T any] struct {
	key   string
	parse func(v string) (T, bool)
}

func newFeature[T any](suffix string, parse func(string) (T, bool)) Feature[T] {
	const envKeyRoot = "OTEL_GO_X_"
	return Feature[T]{
		key:   envKeyRoot + suffix,
		parse: parse,
	}
}

// Key returns the environment variable key that needs to be set to enable the
// feature.
func (f Feature[T]) Key() string { return f.key }

// Lookup returns the user configured value for the feature and true if the
// user has enabled the feature. Otherwise, if the feature is not enabled, a
// zero-value and false are returned.
func (f Feature[T]) Lookup() (v T, ok bool) {
	vRaw, present := os.LookupEnv(f.key)
	if !present {
		return v, ok
	}
	return f.parse(vRaw)
}

// Enabled returns if the feature is enabled.
func (f Feature[T]) Enabled() bool {
	_, ok := f.Lookup()
	return ok
}
