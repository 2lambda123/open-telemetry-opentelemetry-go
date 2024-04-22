// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal // import "go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp/internal"

//go:generate gotmpl --body=../../../../../internal/shared/otlp/partialsuccess.go.tmpl "--data={}" --out=partialsuccess.go
//go:generate gotmpl --body=../../../../../internal/shared/otlp/partialsuccess_test.go.tmpl "--data={}" --out=partialsuccess_test.go

//go:generate gotmpl --body=../../../../../internal/shared/otlp/retry/retry.go.tmpl "--data={}" --out=retry/retry.go
//go:generate gotmpl --body=../../../../../internal/shared/otlp/retry/retry_test.go.tmpl "--data={}" --out=retry/retry_test.go

//go:generate gotmpl --body=../../../../../internal/shared/otlp/envconfig/envconfig.go.tmpl "--data={}" --out=envconfig/envconfig.go
//go:generate gotmpl --body=../../../../../internal/shared/otlp/envconfig/envconfig_test.go.tmpl "--data={}" --out=envconfig/envconfig_test.go

//go:generate gotmpl --body=../../../../../internal/shared/otlp/otlpmetric/oconf/optiontypes.go.tmpl "--data={}" --out=oconf/optiontypes.go
//go:generate gotmpl --body=../../../../../internal/shared/otlp/otlpmetric/oconf/tls.go.tmpl "--data={}" --out=oconf/tls.go

//go:generate gotmpl --body=../../../../../internal/shared/otlp/otlpmetric/otest/client.go.tmpl "--data={\"protoImportPrefix\": \"go.opentelemetry.io/proto/slim\"}" --out=otest/client.go
//go:generate gotmpl --body=../../../../../internal/shared/otlp/otlpmetric/otest/client_test.go.tmpl "--data={\"protoImportPrefix\": \"go.opentelemetry.io/proto/slim\", \"internalImportPath\": \"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp/internal\"}" --out=otest/client_test.go

//go:generate gotmpl --body=../../../../../internal/shared/otlp/otlpmetric/transform/attribute.go.tmpl "--data={\"protoImportPrefix\": \"go.opentelemetry.io/proto/slim\"}" --out=transform/attribute.go
//go:generate gotmpl --body=../../../../../internal/shared/otlp/otlpmetric/transform/attribute_test.go.tmpl "--data={\"protoImportPrefix\": \"go.opentelemetry.io/proto/slim\"}" --out=transform/attribute_test.go
//go:generate gotmpl --body=../../../../../internal/shared/otlp/otlpmetric/transform/error.go.tmpl "--data={\"protoImportPrefix\": \"go.opentelemetry.io/proto/slim\"}" --out=transform/error.go
//go:generate gotmpl --body=../../../../../internal/shared/otlp/otlpmetric/transform/error_test.go.tmpl "--data={}" --out=transform/error_test.go
//go:generate gotmpl --body=../../../../../internal/shared/otlp/otlpmetric/transform/metricdata.go.tmpl "--data={\"protoImportPrefix\": \"go.opentelemetry.io/proto/slim\"}" --out=transform/metricdata.go
//go:generate gotmpl --body=../../../../../internal/shared/otlp/otlpmetric/transform/metricdata_test.go.tmpl "--data={\"protoImportPrefix\": \"go.opentelemetry.io/proto/slim\"}" --out=transform/metricdata_test.go
