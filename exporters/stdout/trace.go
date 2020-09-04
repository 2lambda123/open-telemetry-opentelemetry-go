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

package stdout

import (
	"context"
	"encoding/json"
	"fmt"

	"go.opentelemetry.io/otel/sdk/export/trace"
)

// Exporter is an implementation of trace.SpanSyncer that writes spans to stdout.
type traceExporter struct {
	config Config
}

// ExportSpans writes SpanData in json format to stdout.
func (e *traceExporter) ExportSpans(ctx context.Context, data []*trace.SpanData) error {
	if e.config.DisableTraceExport || len(data) == 0 {
		return nil
	}
	out, err := e.marshal(data)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(e.config.Writer, string(out))
	return err
}

// Shutdown is called to stop the exporter, it preforms no action.
func (e *traceExporter) Shutdown(ctx context.Context) error { return nil }

// marshal v with approriate indentation.
func (e *traceExporter) marshal(v interface{}) ([]byte, error) {
	if e.config.PrettyPrint {
		return json.MarshalIndent(v, "", "\t")
	}
	return json.Marshal(v)
}
