// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal // import "go.opentelemetry.io/otel/bridge/opencensus/internal"

import "go.opentelemetry.io/otel"

// Handle is the package level function to handle errors. It can be
// overwritten for testing.
var Handle = otel.Handle
