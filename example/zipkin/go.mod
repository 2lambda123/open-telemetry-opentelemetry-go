module go.opentelemetry.io/otel/example/zipkin

go 1.20

replace (
	go.opentelemetry.io/otel => ../..
	go.opentelemetry.io/otel/exporters/zipkin => ../../exporters/zipkin
	go.opentelemetry.io/otel/sdk => ../../sdk
)

require (
	go.opentelemetry.io/otel v1.19.0-rc.1
	go.opentelemetry.io/otel/exporters/zipkin v1.19.0-rc.1
	go.opentelemetry.io/otel/sdk v1.19.0-rc.1
	go.opentelemetry.io/otel/trace v1.19.0-rc.1
)

require (
	github.com/Masterminds/semver/v3 v3.2.1 // indirect
	github.com/go-logr/logr v1.2.4 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/openzipkin/zipkin-go v0.4.2 // indirect
	go.opentelemetry.io/otel/metric v1.19.0-rc.1 // indirect
	go.opentelemetry.io/otel/schema v0.0.6 // indirect
	golang.org/x/sys v0.12.0 // indirect
)

replace go.opentelemetry.io/otel/trace => ../../trace

replace go.opentelemetry.io/otel/metric => ../../metric

replace go.opentelemetry.io/otel/schema => ../../schema
