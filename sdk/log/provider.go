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

package log // import "go.opentelemetry.io/otel/sdk/log"

import (
	"go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/log/embedded"
	"go.opentelemetry.io/otel/sdk/resource"
)

// Compile-time check LoggerProvider implements log.LoggerProvider.
var _ log.LoggerProvider = (*LoggerProvider)(nil)

// LoggerProvider handles the creation and coordination of Loggers. All Loggers
// created by a LoggerProvider will be associated with the same Resource.
type LoggerProvider struct {
	embedded.LoggerProvider
}

// NewLoggerProvider returns a new and configured LoggerProvider.
//
// By default, the returned LoggerProvider is configured with the default
// Resource and no Processors. Processors cannot be added after a LoggerProvider is
// created. This means the returned MeterProvider, one created with no
// Processors, will perform no operations.
func NewLoggerProvider(...Option) *LoggerProvider {
	return &LoggerProvider{}
}

// Logger returns a new [log.Logger] with the provided name and configuration.
//
// This method can be called concurrently.
func (*LoggerProvider) Logger(name string, options ...log.LoggerOption) log.Logger {
	return nil
}

// Option applies a configuration option value to a LoggerProvider.
type Option interface{}

// WithResource associates a Resource with a LoggerProvider. This Resource
// represents the entity producing telemetry and is associated with all Loggers
// the LoggerProvider will create.
//
// By default, if this Option is not used, the default Resource from the
// go.opentelemetry.io/otel/sdk/resource package will be used.
func WithResource(*resource.Resource) Option {
	return nil
}

// WithExporter associates Exporter with a LoggerProvider.
//
// By default, if this option is not used, the LoggerProvider will perform no
// operations; no data will be exported without an Exporter.
//
// Use NewBatchingExporter to batch log records before they are exported.
func WithExporter(Exporter) Option {
	return nil
}

// WithAttributeCountLimit sets the maximum allowed log record attribute count.
// Any attribute added to a log record once this limit is reached will be dropped.
//
// Setting this to zero means no attributes will be recorded.
//
// Setting this to a negative value means no limit is applied.
func WithAttributeCountLimit(limit int) Option {
	return nil
}

// AttributeValueLengthLimit sets the maximum allowed attribute value length.
//
// This limit only applies to string and string slice attribute values.
// Any string longer than this value will be truncated to this length.
//
// Setting this to a negative value means no limit is applied.
func WithAttributeValueLengthLimit(limit int) Option {
	return nil
}
