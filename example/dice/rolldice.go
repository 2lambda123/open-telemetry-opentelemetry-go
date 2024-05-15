// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"go.opentelemetry.io/contrib/bridges/otelslog"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/sdk/instrumentation"
)

const name = "rolldice"

var (
	tracer = otel.Tracer(name)
	meter  = otel.Meter(name)
	logger = otelslog.NewLogger(
		otelslog.WithInstrumentationScope(instrumentation.Scope{
			Name: name,
		}),
	)
	rollCnt metric.Int64Counter
)

func init() {
	var err error
	rollCnt, err = meter.Int64Counter("dice.rolls",
		metric.WithDescription("The number of rolls by roll value"),
		metric.WithUnit("{roll}"))
	if err != nil {
		panic(err)
	}
}
