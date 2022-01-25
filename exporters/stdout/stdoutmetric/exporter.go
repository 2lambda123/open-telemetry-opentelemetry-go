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

//go:build go1.17
// +build go1.17

package stdoutmetric // import "go.opentelemetry.io/otel/exporters/stdout/stdoutmetric"

import "go.opentelemetry.io/otel/sdk/metric/export"

type Exporter struct {
	metricExporter
}

var (
	_ export.Exporter = &Exporter{}
)

// New creates an Exporter with the passed options.
func New(options ...Option) (*Exporter, error) {
	cfg, err := newConfig(options...)
	if err != nil {
		return nil, err
	}
	return &Exporter{
		metricExporter: metricExporter{cfg},
	}, nil
}
