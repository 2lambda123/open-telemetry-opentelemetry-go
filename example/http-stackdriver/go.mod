module go.opentelemetry.io/otel/example/http-stackdriver

go 1.13

replace (
	go.opentelemetry.io/otel => ../..
	go.opentelemetry.io/otel/exporter/trace/stackdriver => ../../exporter/trace/stackdriver
)

require (
	go.opentelemetry.io/otel v0.0.0-00010101000000-000000000000
	go.opentelemetry.io/otel/exporter/trace/stackdriver v0.0.0-00010101000000-000000000000
	google.golang.org/grpc v1.24.0
)
