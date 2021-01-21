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

package basic // import "go.opentelemetry.io/otel/sdk/metric/controller/basic"

import (
	"time"

	export "go.opentelemetry.io/otel/sdk/export/metric"
	"go.opentelemetry.io/otel/sdk/resource"
)

// Config contains configuration for a basic Controller.
type Config struct {
	// Resource is the OpenTelemetry resource associated with all Meters
	// created by the Controller.
	Resource *resource.Resource

	// Enricher is an optional function that can be provided to apply baggage attributes
	// as metric labels.
	Enricher export.Enricher

	// CollectPeriod is the interval between calls to Collect a
	// checkpoint.
	//
	// When pulling metrics and not exporting, this is the minimum
	// time between calls to Collect.  In a pull-only
	// configuration, collection is performed on demand; set
	// CollectPeriod to 0 always recompute the export record set.
	//
	// When exporting metrics, this must be > 0.
	//
	// Default value is 10s.
	CollectPeriod time.Duration

	// CollectTimeout is the timeout of the Context passed to
	// Collect() and subsequently to Observer instrument callbacks.
	//
	// Default value is 10s.  If zero, no Collect timeout is applied.
	CollectTimeout time.Duration

	// Pusher is used for exporting metric data.
	//
	// Note: Exporters such as Prometheus that pull data do not implement
	// export.Exporter.  These will directly call Collect() and ForEach().
	Pusher export.Exporter

	// PushTimeout is the timeout of the Context when a Pusher is configured.
	//
	// Default value is 10s.  If zero, no Export timeout is applied.
	PushTimeout time.Duration
}

// Option is the interface that applies the value to a configuration option.
type Option interface {
	// Apply sets the Option value of a Config.
	Apply(*Config)
}

// WithResource sets the Resource configuration option of a Config.
func WithResource(r *resource.Resource) Option {
	return resourceOption{r}
}

type resourceOption struct{ *resource.Resource }

func (o resourceOption) Apply(config *Config) {
	config.Resource = o.Resource
}

// WithEnricher sets the Enricher configuration option of a Config
func WithEnricher(e export.Enricher) Option {
	return enricherOption(e)
}

type enricherOption export.Enricher

func (e enricherOption) Apply(config *Config) {
	config.Enricher = export.Enricher(e)
}

// WithCollectPeriod sets the CollectPeriod configuration option of a Config.
func WithCollectPeriod(period time.Duration) Option {
	return collectPeriodOption(period)
}

type collectPeriodOption time.Duration

func (o collectPeriodOption) Apply(config *Config) {
	config.CollectPeriod = time.Duration(o)
}

// WithCollectTimeout sets the CollectTimeout configuration option of a Config.
func WithCollectTimeout(timeout time.Duration) Option {
	return collectTimeoutOption(timeout)
}

type collectTimeoutOption time.Duration

func (o collectTimeoutOption) Apply(config *Config) {
	config.CollectTimeout = time.Duration(o)
}

// WithPusher sets the Pusher configuration option of a Config.
func WithPusher(pusher export.Exporter) Option {
	return pusherOption{pusher}
}

type pusherOption struct{ pusher export.Exporter }

func (o pusherOption) Apply(config *Config) {
	config.Pusher = o.pusher
}

// WithPushTimeout sets the PushTimeout configuration option of a Config.
func WithPushTimeout(timeout time.Duration) Option {
	return pushTimeoutOption(timeout)
}

type pushTimeoutOption time.Duration

func (o pushTimeoutOption) Apply(config *Config) {
	config.PushTimeout = time.Duration(o)
}
