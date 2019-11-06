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

package trace

import (
	"context"

	"go.opentelemetry.io/otel"
)

type tracer struct {
	provider  *Provider
	name      string
	component string
	resources []otel.KeyValue
}

var _ otel.Tracer = &tracer{}

func (tr *tracer) Start(ctx context.Context, name string, o ...otel.SpanOption) (context.Context, otel.Span) {
	var opts otel.SpanOptions
	var parent otel.SpanContext
	var remoteParent bool

	//TODO [rghetia] : Add new option for parent. If parent is configured then use that parent.
	for _, op := range o {
		op(&opts)
	}

	if relation := opts.Relation; relation.SpanContext != otel.EmptySpanContext() {
		switch relation.RelationshipType {
		case otel.ChildOfRelationship, otel.FollowsFromRelationship:
			parent = relation.SpanContext
			remoteParent = true
		default:
			// Future relationship types may have different behavior,
			// e.g., adding a `Link` instead of setting the `parent`
		}
	} else {
		if p := otel.CurrentSpan(ctx); p != nil {
			if sdkSpan, ok := p.(*span); ok {
				sdkSpan.addChild()
				parent = sdkSpan.spanContext
			}
		}
	}

	spanName := tr.spanNameWithPrefix(name)
	span := startSpanInternal(tr, spanName, parent, remoteParent, opts)
	for _, l := range opts.Links {
		span.AddLink(l)
	}
	span.SetAttributes(opts.Attributes...)

	span.tracer = tr

	if span.IsRecording() {
		sps, _ := tr.provider.spanProcessors.Load().(spanProcessorMap)
		for sp := range sps {
			sp.OnStart(span.data)
		}
	}

	ctx, end := startExecutionTracerTask(ctx, spanName)
	span.executionTracerTaskEnd = end
	return otel.SetCurrentSpan(ctx, span), span
}

func (tr *tracer) WithSpan(ctx context.Context, name string, body func(ctx context.Context) error) error {
	ctx, span := tr.Start(ctx, name)
	defer span.End()

	if err := body(ctx); err != nil {
		// TODO: set event with boolean attribute for error.
		return err
	}
	return nil
}

func (tr *tracer) WithService(name string) otel.Tracer {
	tr.name = name
	return tr
}

// WithResources does nothing and returns noop implementation of otel.Tracer.
func (tr *tracer) WithResources(res ...otel.KeyValue) otel.Tracer {
	tr.resources = res
	return tr
}

// WithComponent does nothing and returns noop implementation of otel.Tracer.
func (tr *tracer) WithComponent(component string) otel.Tracer {
	tr.component = component
	return tr
}

func (tr *tracer) spanNameWithPrefix(name string) string {
	if tr.name != "" {
		return tr.name + "/" + name
	}
	return name
}
