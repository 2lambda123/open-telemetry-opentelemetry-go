module go.opentelemetry.io/otel/exporters/trace/zipkin

go 1.14

replace (
	go.opentelemetry.io/otel => ../../..
	go.opentelemetry.io/otel/baggage => ../../../baggage
	go.opentelemetry.io/otel/label => ../../../label
	go.opentelemetry.io/otel/sdk => ../../../sdk
	go.opentelemetry.io/otel/semconv => ../../../semconv
	go.opentelemetry.io/otel/trace => ../../../trace
)

require (
	github.com/google/go-cmp v0.5.4
	github.com/openzipkin/zipkin-go v0.2.5
	github.com/stretchr/testify v1.7.0
	go.opentelemetry.io/otel v0.16.0
	go.opentelemetry.io/otel/label v0.16.0
	go.opentelemetry.io/otel/sdk v0.16.0
	go.opentelemetry.io/otel/trace v0.16.0
)
