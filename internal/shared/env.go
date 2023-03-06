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

package shared // import "go.opentelemetry.io/otel/internal/shared"

import (
	"errors"
	"os"
	"strconv"
	"time"

	"go.opentelemetry.io/otel/internal/global"
)

// errNonPositiveDuration is logged when an environmental variable
// has non-positive duration value.
var errNonPositiveDuration = errors.New("non-positive duration")

// envString returns the first non-empty environment variable
// value from keys.
// Otherwise, defaultValue is returned.
func envString(defaultValue string, keys ...string) string {
	for _, k := range keys {
		v := os.Getenv(k)
		if v == "" {
			continue
		}
		return v
	}

	return defaultValue
}

// envInt returns the first valid environment variable
// integer value from keys.
// Otherwise, defaultValue is returned.
//
// Use this function for configuring limits.
func envInt(defaultValue int, keys ...string) int {
	for _, k := range keys {
		v := os.Getenv(k)
		if v == "" {
			continue
		}
		n, err := strconv.Atoi(v)
		if err != nil {
			global.Error(err, "parse int", "environment variable", k, "value", v)
			continue
		}
		return n
	}
	return defaultValue
}

// envDuration returns the first valid environment variable
// duration value from keys.
// Otherwise, defaultValue is returned.
//
// The environment variable value is interpreted
// as a number of milliseconds. Only positive values are valid.
//
// Use this function for configuring timeouts and intervals.
func envDuration(defaultValue time.Duration, keys ...string) time.Duration {
	for _, k := range keys {
		v := os.Getenv(k)
		if v == "" {
			continue
		}
		n, err := strconv.Atoi(v)
		if err != nil {
			global.Error(err, "parse int", "environment variable", k, "value", v)
			continue
		}
		if n <= 0 {
			global.Error(errNonPositiveDuration, "non-positive duration", "environment variable", k, "value", v)
			continue
		}
		return time.Duration(n) * time.Millisecond
	}
	return defaultValue
}
