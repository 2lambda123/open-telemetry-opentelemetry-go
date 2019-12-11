// Copyright 2019, OpenTelemetry Authors
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

package httptrace

import (
	"context"
	"net/http"

	bpropagation "go.opentelemetry.io/otel/api/context/baggage/propagation"
	"go.opentelemetry.io/otel/api/context/propagation"
	"go.opentelemetry.io/otel/api/core"
	"go.opentelemetry.io/otel/api/global"
	"go.opentelemetry.io/otel/api/key"
	tpropagation "go.opentelemetry.io/otel/api/trace/propagation"
)

const (
	Vendor = "ot"
)

var (
	HostKey = key.New("http.host")
	URLKey  = key.New("http.url")
)

// Returns the Attributes, Context Entries, and SpanContext that were encoded by Inject.
func Extract(ctx context.Context, req *http.Request) ([]core.KeyValue, []core.KeyValue, core.SpanContext) {
	ctx = propagation.Extract(ctx, global.Propagators(), req.Header)

	attrs := []core.KeyValue{
		URLKey.String(req.URL.String()),
		// Etc.
	}

	var correlationCtxKVs []core.KeyValue
	bpropagation.FromContext(ctx).Foreach(func(kv core.KeyValue) bool {
		correlationCtxKVs = append(correlationCtxKVs, kv)
		return true
	})

	return attrs, correlationCtxKVs, tpropagation.FromContext(ctx)
}

func Inject(ctx context.Context, req *http.Request) {
	propagation.Inject(ctx, global.Propagators(), req.Header)
}
