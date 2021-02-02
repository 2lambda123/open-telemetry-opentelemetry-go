module go.opentelemetry.io/otel/exporters/trace/jaeger

go 1.14

replace (
	go.opentelemetry.io/otel => ../../..
	go.opentelemetry.io/otel/label => ../../../label
	go.opentelemetry.io/otel/sdk => ../../../sdk
	go.opentelemetry.io/otel/semconv => ../../../semconv
	go.opentelemetry.io/otel/trace => ../../../trace
)

require (
	github.com/apache/thrift v0.13.0
	github.com/google/go-cmp v0.5.4
	github.com/stretchr/testify v1.7.0
	go.opentelemetry.io/otel v0.16.0
	go.opentelemetry.io/otel/label v0.16.0
	go.opentelemetry.io/otel/sdk v0.16.0
	go.opentelemetry.io/otel/trace v0.16.0
	google.golang.org/api v0.38.0
)
