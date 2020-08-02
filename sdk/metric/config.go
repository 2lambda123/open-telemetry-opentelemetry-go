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

package metric

import (
	"regexp"

	"go.opentelemetry.io/otel/sdk/resource"
)

// Config contains configuration for an SDK.
type Config struct {
	// Resource describes all the metric records processed by the
	// Accumulator.
	Resource *resource.Resource

	// KeyRegexp is
	KeyRegexp *regexp.Regexp
}

// Option is the interface that applies the value to a configuration option.
type Option interface {
	// Apply sets the Option value of a Config.
	Apply(*Config)
}

// WithResource sets the Resource configuration option of a Config.
func WithResource(res *resource.Resource) Option {
	return resourceOption{res}
}

type resourceOption struct {
	*resource.Resource
}

func (o resourceOption) Apply(config *Config) {
	config.Resource = o.Resource
}

// WithRegexp sets the Regexp configuration option of a Config.
func WithRegexp(res *resource.Regexp) Option {
	return resourceOption{res}
}

type resourceOption struct {
	*regexp.Regexp
}

func (o resourceOption) Apply(config *Config) {
	config.Regexp = o.Regexp
}
