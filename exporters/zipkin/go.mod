module go.opentelemetry.io/otel/exporters/zipkin

go 1.18

require (
	github.com/google/go-cmp v0.5.8
	github.com/openzipkin/zipkin-go v0.4.0
	github.com/stretchr/testify v1.7.1
	go.opentelemetry.io/otel v1.10.0
	go.opentelemetry.io/otel/sdk v1.10.0
	go.opentelemetry.io/otel/trace v1.10.0
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/go-logr/logr v1.2.3 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	golang.org/x/sys v0.0.0-20210615035016-665e8c7367d1 // indirect
	gopkg.in/yaml.v3 v3.0.0-20210107192922-496545a6307b // indirect
)

replace go.opentelemetry.io/otel/trace => ../../trace

replace go.opentelemetry.io/otel => ../..

replace go.opentelemetry.io/otel/sdk => ../../sdk

replace go.opentelemetry.io/otel/bridge/opencensus => ../../bridge/opencensus

replace go.opentelemetry.io/otel/bridge/opencensus/opencensusmetric => ../../bridge/opencensus/opencensusmetric

replace go.opentelemetry.io/otel/bridge/opencensus/test => ../../bridge/opencensus/test

replace go.opentelemetry.io/otel/bridge/opentracing => ../../bridge/opentracing

replace go.opentelemetry.io/otel/example/fib => ../../example/fib

replace go.opentelemetry.io/otel/example/jaeger => ../../example/jaeger

replace go.opentelemetry.io/otel/example/namedtracer => ../../example/namedtracer

replace go.opentelemetry.io/otel/example/otel-collector => ../../example/otel-collector

replace go.opentelemetry.io/otel/example/passthrough => ../../example/passthrough

replace go.opentelemetry.io/otel/example/prometheus => ../../example/prometheus

replace go.opentelemetry.io/otel/example/zipkin => ../../example/zipkin

replace go.opentelemetry.io/otel/exporters/jaeger => ../jaeger

replace go.opentelemetry.io/otel/exporters/otlp/internal => ../otlp/internal

replace go.opentelemetry.io/otel/exporters/otlp/internal/retry => ../otlp/internal/retry

replace go.opentelemetry.io/otel/exporters/otlp/otlpmetric => ../otlp/otlpmetric

replace go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc => ../otlp/otlpmetric/otlpmetricgrpc

replace go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp => ../otlp/otlpmetric/otlpmetrichttp

replace go.opentelemetry.io/otel/exporters/otlp/otlptrace => ../otlp/otlptrace

replace go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc => ../otlp/otlptrace/otlptracegrpc

replace go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp => ../otlp/otlptrace/otlptracehttp

replace go.opentelemetry.io/otel/exporters/prometheus => ../prometheus

replace go.opentelemetry.io/otel/exporters/stdout/stdoutmetric => ../stdout/stdoutmetric

replace go.opentelemetry.io/otel/exporters/stdout/stdouttrace => ../stdout/stdouttrace

replace go.opentelemetry.io/otel/exporters/zipkin => ./

replace go.opentelemetry.io/otel/internal/tools => ../../internal/tools

replace go.opentelemetry.io/otel/metric => ../../metric

replace go.opentelemetry.io/otel/schema => ../../schema

replace go.opentelemetry.io/otel/sdk/metric => ../../sdk/metric
